package ratelimit

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// fakeRedisCmdable 只实现本测试需要的 TxPipelined，其他 Redis 能力由嵌入接口占位。
type fakeRedisCmdable struct {
	redis.Cmdable

	pipe     *fakeRedisPipeline
	txCalled bool
	execErr  error
}

// TxPipelined 记录 RedisStore 是否使用事务 pipeline，并执行测试替身 pipeline。
func (c *fakeRedisCmdable) TxPipelined(ctx context.Context, fn func(redis.Pipeliner) error) ([]redis.Cmder, error) {
	c.txCalled = true

	if err := fn(c.pipe); err != nil {
		return nil, err
	}

	if c.execErr != nil {
		return nil, c.execErr
	}

	return c.pipe.commands, nil
}

// fakeRedisPipeline 记录 Increment 发出的 Redis 命令，并返回预设结果。
type fakeRedisPipeline struct {
	redis.Pipeliner

	count    int64
	ttl      time.Duration
	commands []redis.Cmder
	names    []string
}

// Incr 模拟 Redis INCR 命令。
func (p *fakeRedisPipeline) Incr(ctx context.Context, key string) *redis.IntCmd {
	cmd := redis.NewIntCmd(ctx, "incr", key)
	cmd.SetVal(p.count)

	p.commands = append(p.commands, cmd)
	p.names = append(p.names, "incr:"+key)

	return cmd
}

// ExpireNX 模拟 Redis EXPIRE key seconds NX 命令。
func (p *fakeRedisPipeline) ExpireNX(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd {
	cmd := redis.NewBoolCmd(ctx, "expire", key, expiration, "nx")
	cmd.SetVal(true)

	p.commands = append(p.commands, cmd)
	p.names = append(p.names, "expire_nx:"+key+":"+expiration.String())

	return cmd
}

// TTL 模拟 Redis TTL 命令。
func (p *fakeRedisPipeline) TTL(ctx context.Context, key string) *redis.DurationCmd {
	cmd := redis.NewDurationCmd(ctx, time.Second, "ttl", key)
	cmd.SetVal(p.ttl)

	p.commands = append(p.commands, cmd)
	p.names = append(p.names, "ttl:"+key)

	return cmd
}

func TestRedisStoreIncrementUsesTransactionPipeline(t *testing.T) {
	now := time.Date(2026, 5, 20, 10, 0, 0, 0, time.UTC)
	pipe := &fakeRedisPipeline{
		count: 3,
		ttl:   45 * time.Second,
	}
	client := &fakeRedisCmdable{pipe: pipe}
	store := NewRedisStore(client, "unio:test")
	store.now = func() time.Time {
		return now
	}

	result, err := store.Increment(context.Background(), "api_key:1", time.Minute)
	if err != nil {
		t.Fatalf("increment: %v", err)
	}

	if !client.txCalled {
		t.Fatal("expected TxPipelined to be called")
	}

	wantKey := "unio:test:ratelimit:api_key:1"
	wantNames := []string{
		"incr:" + wantKey,
		"expire_nx:" + wantKey + ":1m0s",
		"ttl:" + wantKey,
	}
	if !reflect.DeepEqual(pipe.names, wantNames) {
		t.Fatalf("expected redis commands %v, got %v", wantNames, pipe.names)
	}

	if result.Count != 3 {
		t.Fatalf("expected count %d, got %d", 3, result.Count)
	}

	wantResetAt := now.Add(45 * time.Second)
	if !result.ResetAt.Equal(wantResetAt) {
		t.Fatalf("expected reset_at %v, got %v", wantResetAt, result.ResetAt)
	}
}

func TestRedisStoreIncrementReturnsTransactionError(t *testing.T) {
	execErr := errors.New("redis transaction failed")
	client := &fakeRedisCmdable{
		pipe:    &fakeRedisPipeline{},
		execErr: execErr,
	}
	store := NewRedisStore(client, "unio:test")

	_, err := store.Increment(context.Background(), "api_key:1", time.Minute)
	if !errors.Is(err, execErr) {
		t.Fatalf("expected redis transaction error, got %v", err)
	}
}

func TestRedisKeyForSubjectUsesNamespace(t *testing.T) {
	got := redisKeyForSubject("unio:prod", "api_key:1")
	want := "unio:prod:ratelimit:api_key:1"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRedisKeyForSubjectTrimsNamespaceColon(t *testing.T) {
	got := redisKeyForSubject(":unio:test:", "api_key:1")
	want := "unio:test:ratelimit:api_key:1"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRedisKeyForSubjectUsesFallbackNamespace(t *testing.T) {
	got := redisKeyForSubject("", "api_key:1")
	want := "unio:ratelimit:api_key:1"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
