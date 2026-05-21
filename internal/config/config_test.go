package config

import (
	"log/slog"
	"testing"
	"time"

	"github.com/ThankCat/unio-api/internal/failure"
)

func TestLoadDefaultRedisDB(t *testing.T) {
	t.Setenv("REDIS_DB", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Redis.DB != 0 {
		t.Fatalf("expected redis db %d, got %d", 0, cfg.Redis.DB)
	}
}

func TestLoadRedisDBFromEnv(t *testing.T) {
	t.Setenv("REDIS_DB", "2")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Redis.DB != 2 {
		t.Fatalf("expected redis db %d, got %d", 2, cfg.Redis.DB)
	}
}

func TestLoadInvalidRedisDB(t *testing.T) {
	t.Setenv("REDIS_DB", "abc")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	assertConfigFailure(t, err, failure.CodeConfigInvalid)
}

func TestLoadLogLevelDebug(t *testing.T) {
	t.Setenv("LOG_LEVEL", "debug")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Log.Level != slog.LevelDebug {
		t.Fatalf("expected log level %v, got %v", slog.LevelDebug, cfg.Log.Level)
	}
}

func TestLoadInvalidLogLevel(t *testing.T) {
	t.Setenv("LOG_LEVEL", "trace")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	assertConfigFailure(t, err, failure.CodeConfigUnsupported)
}

func TestLoadInfrastructureDefaults(t *testing.T) {
	clearInfrastructureEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.HTTP.ReadTimeout != 10*time.Second {
		t.Fatalf("expected HTTP read timeout %v, got %v", 10*time.Second, cfg.HTTP.ReadTimeout)
	}
	if cfg.HTTP.WriteTimeout != 30*time.Second {
		t.Fatalf("expected HTTP write timeout %v, got %v", 30*time.Second, cfg.HTTP.WriteTimeout)
	}
	if cfg.HTTP.IdleTimeout != 60*time.Second {
		t.Fatalf("expected HTTP idle timeout %v, got %v", 60*time.Second, cfg.HTTP.IdleTimeout)
	}
	if cfg.HTTP.ShutdownTimeout != 10*time.Second {
		t.Fatalf("expected HTTP shutdown timeout %v, got %v", 10*time.Second, cfg.HTTP.ShutdownTimeout)
	}
	if cfg.DB.MaxConns != 10 {
		t.Fatalf("expected postgres max conns %d, got %d", 10, cfg.DB.MaxConns)
	}
	if cfg.DB.MinConns != 1 {
		t.Fatalf("expected postgres min conns %d, got %d", 1, cfg.DB.MinConns)
	}
	if cfg.DB.MaxConnLifetime != time.Hour {
		t.Fatalf("expected postgres max conn lifetime %v, got %v", time.Hour, cfg.DB.MaxConnLifetime)
	}
	if cfg.DB.MaxConnIdleTime != 30*time.Minute {
		t.Fatalf("expected postgres max conn idle time %v, got %v", 30*time.Minute, cfg.DB.MaxConnIdleTime)
	}
	if cfg.DB.HealthCheckPeriod != 5*time.Second {
		t.Fatalf("expected postgres health check period %v, got %v", 5*time.Second, cfg.DB.HealthCheckPeriod)
	}
	if cfg.Redis.DialTimeout != 5*time.Second {
		t.Fatalf("expected redis dial timeout %v, got %v", 5*time.Second, cfg.Redis.DialTimeout)
	}
	if cfg.Redis.ReadTimeout != 3*time.Second {
		t.Fatalf("expected redis read timeout %v, got %v", 3*time.Second, cfg.Redis.ReadTimeout)
	}
	if cfg.Redis.WriteTimeout != 3*time.Second {
		t.Fatalf("expected redis write timeout %v, got %v", 3*time.Second, cfg.Redis.WriteTimeout)
	}
	if cfg.Redis.PoolSize != 10 {
		t.Fatalf("expected redis pool size %d, got %d", 10, cfg.Redis.PoolSize)
	}
	if cfg.Redis.MaxRetries != 3 {
		t.Fatalf("expected redis max retries %d, got %d", 3, cfg.Redis.MaxRetries)
	}
	if cfg.Redis.MinRetryBackoff != 8*time.Millisecond {
		t.Fatalf("expected redis min retry backoff %v, got %v", 8*time.Millisecond, cfg.Redis.MinRetryBackoff)
	}
	if cfg.Redis.MaxRetryBackoff != 512*time.Millisecond {
		t.Fatalf("expected redis max retry backoff %v, got %v", 512*time.Millisecond, cfg.Redis.MaxRetryBackoff)
	}
	if cfg.Redis.KeyNamespace != "unio:dev" {
		t.Fatalf("expected redis key namespace %q, got %q", "unio:dev", cfg.Redis.KeyNamespace)
	}
	if cfg.RateLimit.DefaultLimit != 60 {
		t.Fatalf("expected rate limit default limit %d, got %d", 60, cfg.RateLimit.DefaultLimit)
	}
	if cfg.RateLimit.DefaultWindow != time.Minute {
		t.Fatalf("expected rate limit default window %v, got %v", time.Minute, cfg.RateLimit.DefaultWindow)
	}
	if cfg.RateLimit.FailurePolicy != "fail_closed" {
		t.Fatalf("expected rate limit failure policy %q, got %q", "fail_closed", cfg.RateLimit.FailurePolicy)
	}
}

func TestLoadInfrastructureOverrides(t *testing.T) {
	clearInfrastructureEnv(t)

	t.Setenv("HTTP_READ_TIMEOUT", "3s")
	t.Setenv("HTTP_WRITE_TIMEOUT", "4s")
	t.Setenv("HTTP_IDLE_TIMEOUT", "5s")
	t.Setenv("HTTP_SHUTDOWN_TIMEOUT", "6s")
	t.Setenv("POSTGRES_MAX_CONNS", "20")
	t.Setenv("POSTGRES_MIN_CONNS", "2")
	t.Setenv("POSTGRES_MAX_CONN_LIFETIME", "2h")
	t.Setenv("POSTGRES_MAX_CONN_IDLE_TIME", "45m")
	t.Setenv("POSTGRES_HEALTH_CHECK_PERIOD", "15s")
	t.Setenv("REDIS_DIAL_TIMEOUT", "7s")
	t.Setenv("REDIS_READ_TIMEOUT", "8s")
	t.Setenv("REDIS_WRITE_TIMEOUT", "9s")
	t.Setenv("REDIS_POOL_SIZE", "30")
	t.Setenv("REDIS_MAX_RETRIES", "5")
	t.Setenv("REDIS_MIN_RETRY_BACKOFF", "10ms")
	t.Setenv("REDIS_MAX_RETRY_BACKOFF", "1s")
	t.Setenv("REDIS_KEY_NAMESPACE", "unio:test")
	t.Setenv("RATE_LIMIT_DEFAULT_LIMIT", "120")
	t.Setenv("RATE_LIMIT_DEFAULT_WINDOW", "30s")
	t.Setenv("RATE_LIMIT_FAILURE_POLICY", "fail_open")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.HTTP.ReadTimeout != 3*time.Second {
		t.Fatalf("expected HTTP read timeout %v, got %v", 3*time.Second, cfg.HTTP.ReadTimeout)
	}
	if cfg.HTTP.WriteTimeout != 4*time.Second {
		t.Fatalf("expected HTTP write timeout %v, got %v", 4*time.Second, cfg.HTTP.WriteTimeout)
	}
	if cfg.HTTP.IdleTimeout != 5*time.Second {
		t.Fatalf("expected HTTP idle timeout %v, got %v", 5*time.Second, cfg.HTTP.IdleTimeout)
	}
	if cfg.HTTP.ShutdownTimeout != 6*time.Second {
		t.Fatalf("expected HTTP shutdown timeout %v, got %v", 6*time.Second, cfg.HTTP.ShutdownTimeout)
	}
	if cfg.DB.MaxConns != 20 {
		t.Fatalf("expected postgres max conns %d, got %d", 20, cfg.DB.MaxConns)
	}
	if cfg.DB.MinConns != 2 {
		t.Fatalf("expected postgres min conns %d, got %d", 2, cfg.DB.MinConns)
	}
	if cfg.DB.MaxConnLifetime != 2*time.Hour {
		t.Fatalf("expected postgres max conn lifetime %v, got %v", 2*time.Hour, cfg.DB.MaxConnLifetime)
	}
	if cfg.DB.MaxConnIdleTime != 45*time.Minute {
		t.Fatalf("expected postgres max conn idle time %v, got %v", 45*time.Minute, cfg.DB.MaxConnIdleTime)
	}
	if cfg.DB.HealthCheckPeriod != 15*time.Second {
		t.Fatalf("expected postgres health check period %v, got %v", 15*time.Second, cfg.DB.HealthCheckPeriod)
	}
	if cfg.Redis.DialTimeout != 7*time.Second {
		t.Fatalf("expected redis dial timeout %v, got %v", 7*time.Second, cfg.Redis.DialTimeout)
	}
	if cfg.Redis.ReadTimeout != 8*time.Second {
		t.Fatalf("expected redis read timeout %v, got %v", 8*time.Second, cfg.Redis.ReadTimeout)
	}
	if cfg.Redis.WriteTimeout != 9*time.Second {
		t.Fatalf("expected redis write timeout %v, got %v", 9*time.Second, cfg.Redis.WriteTimeout)
	}
	if cfg.Redis.PoolSize != 30 {
		t.Fatalf("expected redis pool size %d, got %d", 30, cfg.Redis.PoolSize)
	}
	if cfg.Redis.MaxRetries != 5 {
		t.Fatalf("expected redis max retries %d, got %d", 5, cfg.Redis.MaxRetries)
	}
	if cfg.Redis.MinRetryBackoff != 10*time.Millisecond {
		t.Fatalf("expected redis min retry backoff %v, got %v", 10*time.Millisecond, cfg.Redis.MinRetryBackoff)
	}
	if cfg.Redis.MaxRetryBackoff != time.Second {
		t.Fatalf("expected redis max retry backoff %v, got %v", time.Second, cfg.Redis.MaxRetryBackoff)
	}
	if cfg.Redis.KeyNamespace != "unio:test" {
		t.Fatalf("expected redis key namespace %q, got %q", "unio:test", cfg.Redis.KeyNamespace)
	}
	if cfg.RateLimit.DefaultLimit != 120 {
		t.Fatalf("expected rate limit default limit %d, got %d", 120, cfg.RateLimit.DefaultLimit)
	}
	if cfg.RateLimit.DefaultWindow != 30*time.Second {
		t.Fatalf("expected rate limit default window %v, got %v", 30*time.Second, cfg.RateLimit.DefaultWindow)
	}
	if cfg.RateLimit.FailurePolicy != "fail_open" {
		t.Fatalf("expected rate limit failure policy %q, got %q", "fail_open", cfg.RateLimit.FailurePolicy)
	}
}

func TestLoadInvalidDuration(t *testing.T) {
	clearInfrastructureEnv(t)

	t.Setenv("HTTP_READ_TIMEOUT", "not-a-duration")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	assertConfigFailure(t, err, failure.CodeConfigInvalid)
}

func TestLoadInvalidPostgresMaxConns(t *testing.T) {
	clearInfrastructureEnv(t)

	t.Setenv("POSTGRES_MAX_CONNS", "2147483648")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	assertConfigFailure(t, err, failure.CodeConfigInvalid)
}

func TestLoadInvalidRedisPoolSize(t *testing.T) {
	clearInfrastructureEnv(t)

	t.Setenv("REDIS_POOL_SIZE", "not-an-int")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	assertConfigFailure(t, err, failure.CodeConfigInvalid)
}

func TestLoadInvalidRateLimitDefaultLimit(t *testing.T) {
	clearInfrastructureEnv(t)

	t.Setenv("RATE_LIMIT_DEFAULT_LIMIT", "not-an-int64")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	assertConfigFailure(t, err, failure.CodeConfigInvalid)
}

func TestLoadInvalidRateLimitDefaultWindow(t *testing.T) {
	clearInfrastructureEnv(t)

	t.Setenv("RATE_LIMIT_DEFAULT_WINDOW", "not-a-duration")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	assertConfigFailure(t, err, failure.CodeConfigInvalid)
}

func TestLoadInvalidRateLimitFailurePolicy(t *testing.T) {
	clearInfrastructureEnv(t)

	t.Setenv("RATE_LIMIT_FAILURE_POLICY", "unknown")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	assertConfigFailure(t, err, failure.CodeConfigUnsupported)
}

func assertConfigFailure(t *testing.T, err error, wantCode failure.Code) {
	t.Helper()

	if failure.CodeOf(err) != wantCode {
		t.Fatalf("expected failure code %q, got %q", wantCode, failure.CodeOf(err))
	}
	if failure.CategoryOf(err) != failure.CategoryConfig {
		t.Fatalf("expected failure category %q, got %q", failure.CategoryConfig, failure.CategoryOf(err))
	}
	if fields := failure.FieldsOf(err); len(fields) != 0 {
		t.Fatalf("expected no failure fields, got %#v", fields)
	}
}

// clearInfrastructureEnv 清空基础设施配置环境变量，避免测试受本机 shell 环境影响。
func clearInfrastructureEnv(t *testing.T) {
	t.Helper()

	for _, key := range []string{
		"HTTP_ADDR",
		"HTTP_READ_TIMEOUT",
		"HTTP_WRITE_TIMEOUT",
		"HTTP_IDLE_TIMEOUT",
		"HTTP_SHUTDOWN_TIMEOUT",
		"LOG_LEVEL",
		"DATABASE_URL",
		"POSTGRES_MAX_CONNS",
		"POSTGRES_MIN_CONNS",
		"POSTGRES_MAX_CONN_LIFETIME",
		"POSTGRES_MAX_CONN_IDLE_TIME",
		"POSTGRES_HEALTH_CHECK_PERIOD",
		"REDIS_ADDR",
		"REDIS_PASSWORD",
		"REDIS_DB",
		"REDIS_DIAL_TIMEOUT",
		"REDIS_READ_TIMEOUT",
		"REDIS_WRITE_TIMEOUT",
		"REDIS_POOL_SIZE",
		"REDIS_MAX_RETRIES",
		"REDIS_MIN_RETRY_BACKOFF",
		"REDIS_MAX_RETRY_BACKOFF",
		"REDIS_KEY_NAMESPACE",
		"RATE_LIMIT_DEFAULT_LIMIT",
		"RATE_LIMIT_DEFAULT_WINDOW",
		"RATE_LIMIT_FAILURE_POLICY",
	} {
		t.Setenv(key, "")
	}
}
