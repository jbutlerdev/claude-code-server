package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	ServerHost         string
	ServerPort         int
	ServerReadTimeout  time.Duration
	ServerWriteTimeout time.Duration

	DatabaseURL                string
	DatabaseMaxConnections     int
	DatabaseMaxIdleConnections int

	ClaudeCodeBinary      string
	ClaudeCodeAPIKey      string
	MaxConcurrentSessions int
	SessionTimeout        time.Duration
	WorktreeBasePath      string
	LogRetentionDays      int

	DefaultGitHubToken string

	JWTSecret          string
	APIRateLimit       int
	APIRateLimitWindow time.Duration

	LogLevel       string
	LogFormat      string
	MetricsEnabled bool
}

func Load() *Config {
	return &Config{
		ServerHost:         getEnv("SERVER_HOST", "0.0.0.0"),
		ServerPort:         getEnvInt("SERVER_PORT", 8080),
		ServerReadTimeout:  getEnvDuration("SERVER_READ_TIMEOUT", 30*time.Second),
		ServerWriteTimeout: getEnvDuration("SERVER_WRITE_TIMEOUT", 30*time.Second),

		DatabaseURL:                getEnv("DATABASE_URL", "postgres://localhost/claude_code_server?sslmode=disable"),
		DatabaseMaxConnections:     getEnvInt("DATABASE_MAX_CONNECTIONS", 25),
		DatabaseMaxIdleConnections: getEnvInt("DATABASE_MAX_IDLE_CONNECTIONS", 5),

		ClaudeCodeBinary:      getEnv("CLAUDE_CODE_BINARY", "/usr/local/bin/claude"),
		ClaudeCodeAPIKey:      getEnv("CLAUDE_CODE_API_KEY", ""),
		MaxConcurrentSessions: getEnvInt("MAX_CONCURRENT_SESSIONS", 10),
		SessionTimeout:        getEnvDuration("SESSION_TIMEOUT", time.Hour),
		WorktreeBasePath:      getEnv("WORKTREE_BASE_PATH", "/var/lib/claude-code-server/worktrees"),
		LogRetentionDays:      getEnvInt("LOG_RETENTION_DAYS", 30),

		DefaultGitHubToken: getEnv("DEFAULT_GITHUB_TOKEN", ""),

		JWTSecret:          getEnv("JWT_SECRET", "change-me-in-production"),
		APIRateLimit:       getEnvInt("API_RATE_LIMIT", 100),
		APIRateLimitWindow: getEnvDuration("API_RATE_LIMIT_WINDOW", time.Minute),

		LogLevel:       getEnv("LOG_LEVEL", "info"),
		LogFormat:      getEnv("LOG_FORMAT", "json"),
		MetricsEnabled: getEnvBool("METRICS_ENABLED", true),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
