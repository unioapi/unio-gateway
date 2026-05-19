package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
)

// Config 保存服务启动所需的全部配置。
type Config struct {
	HTTP  HTTPConfig
	Log   LogConfig
	DB    DBConfig
	Redis RedisConfig
}

// HTTPConfig 保存 HTTP server 监听配置。
type HTTPConfig struct {
	Addr string
}

// LogConfig 保存结构化日志配置。
type LogConfig struct {
	Level slog.Level
}

// DBConfig 保存 PostgreSQL 连接配置。
type DBConfig struct {
	URL string

	// TODO(阶段2/production): [GAP-2-003] 将 PostgreSQL pool 参数纳入配置，包括 max conns、min conns、max lifetime 和健康检查超时。
}

// RedisConfig 保存 Redis client 连接配置。
type RedisConfig struct {
	Addr     string
	Password string
	DB       int

	// TODO(阶段2/production): [GAP-2-004] 将 Redis dial/read/write timeout、pool size、namespace 和故障降级策略纳入配置。
}

// TODO(阶段6/production): [GAP-6-004] provider/channel 业务数据进入 config 会阻断后台动态管理；接入数据库 channel 时；config 只保留 KMS/master key 和全局默认上游 timeout 等启动级配置。

// Load 从环境变量加载配置，并对需要解析的字段做启动期校验。
func Load() (Config, error) {
	redisDB, err := getEnvInt("REDIS_DB", 0)
	if err != nil {
		return Config{}, err
	}

	logLevel, err := parseLogLevel(getEnv("LOG_LEVEL", "info"))
	if err != nil {
		return Config{}, err
	}

	return Config{
		HTTP: HTTPConfig{
			Addr: getEnv("HTTP_ADDR", ":8520"),
		},
		Log: LogConfig{
			Level: logLevel,
		},
		DB: DBConfig{
			URL: getEnv("DATABASE_URL", ""),
		},
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", "localhost:6380"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       redisDB,
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
