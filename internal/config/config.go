package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config 保存服务启动所需的全部配置。
type Config struct {
	HTTP      HTTPConfig
	Log       LogConfig
	DB        DBConfig
	Redis     RedisConfig
	RateLimit RateLimitConfig
}

// HTTPConfig 保存 HTTP server 监听配置。
type HTTPConfig struct {
	Addr            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

// LogConfig 保存结构化日志配置。
type LogConfig struct {
	Level slog.Level
}

// DBConfig 保存 PostgreSQL 连接配置。
type DBConfig struct {
	URL               string
	MaxConns          int32
	MinConns          int32
	MaxConnLifetime   time.Duration
	MaxConnIdleTime   time.Duration
	HealthCheckPeriod time.Duration
}

// RedisConfig 保存 Redis client 连接配置。
type RedisConfig struct {
	Addr            string
	Password        string
	DB              int
	DialTimeout     time.Duration
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	PoolSize        int
	MaxRetries      int
	MinRetryBackoff time.Duration
	MaxRetryBackoff time.Duration
	KeyNamespace    string
}

// RateLimitConfig 保存全局默认限流配置。
type RateLimitConfig struct {
	DefaultLimit  int64
	DefaultWindow time.Duration
	FailurePolicy string
}

// TODO(阶段6/production): [GAP-6-004] provider/channel 业务数据进入 config 会阻断后台动态管理；接入数据库 channel 时；config 只保留 KMS/master key 和全局默认上游 timeout 等启动级配置。

// Load 从环境变量加载配置，并对需要解析的字段做启动期校验。
func Load() (Config, error) {
	redisDB, err := getEnvInt("REDIS_DB", 0)
	if err != nil {
		return Config{}, err
	}

	redisDialTimeout, err := getEnvDuration("REDIS_DIAL_TIMEOUT", 5*time.Second)
	if err != nil {
		return Config{}, err
	}

	redisReadTimeout, err := getEnvDuration("REDIS_READ_TIMEOUT", 3*time.Second)
	if err != nil {
		return Config{}, err
	}

	redisWriteTimeout, err := getEnvDuration("REDIS_WRITE_TIMEOUT", 3*time.Second)
	if err != nil {
		return Config{}, err
	}

	redisPoolSize, err := getEnvInt("REDIS_POOL_SIZE", 10)
	if err != nil {
		return Config{}, err
	}

	redisMaxRetries, err := getEnvInt("REDIS_MAX_RETRIES", 3)
	if err != nil {
		return Config{}, err
	}

	redisMinRetryBackoff, err := getEnvDuration("REDIS_MIN_RETRY_BACKOFF", 8*time.Millisecond)
	if err != nil {
		return Config{}, err
	}

	redisMaxRetryBackoff, err := getEnvDuration("REDIS_MAX_RETRY_BACKOFF", 512*time.Millisecond)
	if err != nil {
		return Config{}, err
	}

	logLevel, err := parseLogLevel(getEnv("LOG_LEVEL", "info"))
	if err != nil {
		return Config{}, err
	}

	httpReadTimeout, err := getEnvDuration("HTTP_READ_TIMEOUT", 10*time.Second)
	if err != nil {
		return Config{}, err
	}

	httpWriteTimeout, err := getEnvDuration("HTTP_WRITE_TIMEOUT", 30*time.Second)
	if err != nil {
		return Config{}, err
	}

	httpIdleTimeout, err := getEnvDuration("HTTP_IDLE_TIMEOUT", 60*time.Second)
	if err != nil {
		return Config{}, err
	}

	httpShutdownTimeout, err := getEnvDuration("HTTP_SHUTDOWN_TIMEOUT", 10*time.Second)
	if err != nil {
		return Config{}, err
	}

	postgresMaxConns, err := getEnvInt32("POSTGRES_MAX_CONNS", 10)
	if err != nil {
		return Config{}, err
	}

	postgresMinConns, err := getEnvInt32("POSTGRES_MIN_CONNS", 1)
	if err != nil {
		return Config{}, err
	}

	postgresMaxConnLifetime, err := getEnvDuration("POSTGRES_MAX_CONN_LIFETIME", time.Hour)
	if err != nil {
		return Config{}, err
	}

	postgresMaxConnIdleTime, err := getEnvDuration("POSTGRES_MAX_CONN_IDLE_TIME", 30*time.Minute)
	if err != nil {
		return Config{}, err
	}

	postgresHealthCheckPeriod, err := getEnvDuration("POSTGRES_HEALTH_CHECK_PERIOD", 5*time.Second)
	if err != nil {
		return Config{}, err
	}

	rateLimitDefaultLimit, err := getEnvInt64("RATE_LIMIT_DEFAULT_LIMIT", 60)
	if err != nil {
		return Config{}, err
	}

	rateLimitDefaultWindow, err := getEnvDuration("RATE_LIMIT_DEFAULT_WINDOW", time.Minute)
	if err != nil {
		return Config{}, err
	}

	rateLimitFailurePolicy, err := parseRateLimitFailurePolicy(getEnv("RATE_LIMIT_FAILURE_POLICY", "fail_closed"))
	if err != nil {
		return Config{}, err
	}

	return Config{
		HTTP: HTTPConfig{
			Addr:            getEnv("HTTP_ADDR", ":8520"),
			ReadTimeout:     httpReadTimeout,
			WriteTimeout:    httpWriteTimeout,
			IdleTimeout:     httpIdleTimeout,
			ShutdownTimeout: httpShutdownTimeout,
		},
		Log: LogConfig{
			Level: logLevel,
		},
		DB: DBConfig{
			URL:               getEnv("DATABASE_URL", ""),
			MaxConns:          postgresMaxConns,
			MinConns:          postgresMinConns,
			MaxConnLifetime:   postgresMaxConnLifetime,
			MaxConnIdleTime:   postgresMaxConnIdleTime,
			HealthCheckPeriod: postgresHealthCheckPeriod,
		},
		Redis: RedisConfig{
			Addr:            getEnv("REDIS_ADDR", "localhost:6380"),
			Password:        getEnv("REDIS_PASSWORD", ""),
			DB:              redisDB,
			DialTimeout:     redisDialTimeout,
			ReadTimeout:     redisReadTimeout,
			WriteTimeout:    redisWriteTimeout,
			PoolSize:        redisPoolSize,
			MaxRetries:      redisMaxRetries,
			MinRetryBackoff: redisMinRetryBackoff,
			MaxRetryBackoff: redisMaxRetryBackoff,
			KeyNamespace:    getEnv("REDIS_KEY_NAMESPACE", "unio:dev"),
		},
		RateLimit: RateLimitConfig{
			DefaultLimit:  rateLimitDefaultLimit,
			DefaultWindow: rateLimitDefaultWindow,
			FailurePolicy: rateLimitFailurePolicy,
		},
	}, nil
}

// getEnv 读取字符串环境变量；未设置时返回 fallback。
func getEnv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

// getEnvInt 读取整数配置；格式错误时让启动流程尽早失败。
func getEnvInt(key string, fallback int) (int, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}

	n, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s as int: %w", key, err)
	}

	return n, nil
}

// getEnvInt32 读取 int32 配置；格式或范围错误时让启动流程尽早失败。
func getEnvInt32(key string, fallback int32) (int32, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}

	n, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("parse %s as int32: %w", key, err)
	}

	return int32(n), nil
}

// getEnvInt64 读取 int64 配置；格式错误时让启动流程尽早失败。
func getEnvInt64(key string, fallback int64) (int64, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}

	n, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse %s as int64: %w", key, err)
	}

	return n, nil
}

// parseLogLevel 将环境变量中的日志级别转换为 slog.Level。
func parseLogLevel(value string) (slog.Level, error) {
	switch strings.ToLower(value) {
	case "", "info":
		return slog.LevelInfo, nil
	case "debug":
		return slog.LevelDebug, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf("parse LOG_LEVEL: unsupported level %q", value)
	}
}

// getEnvDuration 读取 duration 配置；格式错误时让启动流程尽早失败。
func getEnvDuration(key string, fallback time.Duration) (time.Duration, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}

	d, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s as duration: %w", key, err)
	}

	return d, nil
}

// parseRateLimitFailurePolicy 校验 Redis 限流故障时的处理策略。
func parseRateLimitFailurePolicy(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "fail_closed":
		return "fail_closed", nil
	case "fail_open":
		return "fail_open", nil
	default:
		return "", fmt.Errorf("parse RATE_LIMIT_FAILURE_POLICY: unsupported policy %q", value)
	}
}
