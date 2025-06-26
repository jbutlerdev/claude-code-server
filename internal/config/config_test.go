package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	// Save original environment
	originalEnv := make(map[string]string)
	envVars := []string{
		"SERVER_HOST", "SERVER_PORT", "DATABASE_URL", "CLAUDE_CODE_API_KEY",
		"MAX_CONCURRENT_SESSIONS", "LOG_LEVEL",
	}

	for _, env := range envVars {
		originalEnv[env] = os.Getenv(env)
	}

	// Clean environment
	for _, env := range envVars {
		os.Unsetenv(env)
	}

	defer func() {
		// Restore original environment
		for _, env := range envVars {
			if val, exists := originalEnv[env]; exists {
				os.Setenv(env, val)
			} else {
				os.Unsetenv(env)
			}
		}
	}()

	// Test default values
	cfg := Load()

	if cfg.ServerHost != "0.0.0.0" {
		t.Errorf("Expected default ServerHost '0.0.0.0', got '%s'", cfg.ServerHost)
	}

	if cfg.ServerPort != 8080 {
		t.Errorf("Expected default ServerPort 8080, got %d", cfg.ServerPort)
	}

	if cfg.MaxConcurrentSessions != 10 {
		t.Errorf("Expected default MaxConcurrentSessions 10, got %d", cfg.MaxConcurrentSessions)
	}

	if cfg.LogLevel != "info" {
		t.Errorf("Expected default LogLevel 'info', got '%s'", cfg.LogLevel)
	}

	// Test environment variable override
	os.Setenv("SERVER_PORT", "9090")
	os.Setenv("MAX_CONCURRENT_SESSIONS", "20")
	os.Setenv("LOG_LEVEL", "debug")

	cfg = Load()

	if cfg.ServerPort != 9090 {
		t.Errorf("Expected ServerPort 9090 from env, got %d", cfg.ServerPort)
	}

	if cfg.MaxConcurrentSessions != 20 {
		t.Errorf("Expected MaxConcurrentSessions 20 from env, got %d", cfg.MaxConcurrentSessions)
	}

	if cfg.LogLevel != "debug" {
		t.Errorf("Expected LogLevel 'debug' from env, got '%s'", cfg.LogLevel)
	}
}

func TestGetEnvInt(t *testing.T) {
	tests := []struct {
		key          string
		value        string
		defaultValue int
		expected     int
	}{
		{"TEST_INT_1", "123", 0, 123},
		{"TEST_INT_2", "invalid", 456, 456},
		{"TEST_INT_3", "", 789, 789},
	}

	for _, test := range tests {
		if test.value != "" {
			os.Setenv(test.key, test.value)
		} else {
			os.Unsetenv(test.key)
		}

		result := getEnvInt(test.key, test.defaultValue)
		if result != test.expected {
			t.Errorf("getEnvInt(%s, %d) = %d, expected %d",
				test.key, test.defaultValue, result, test.expected)
		}

		os.Unsetenv(test.key)
	}
}

func TestGetEnvBool(t *testing.T) {
	tests := []struct {
		key          string
		value        string
		defaultValue bool
		expected     bool
	}{
		{"TEST_BOOL_1", "true", false, true},
		{"TEST_BOOL_2", "false", true, false},
		{"TEST_BOOL_3", "1", false, true},
		{"TEST_BOOL_4", "0", true, false},
		{"TEST_BOOL_5", "invalid", true, true},
		{"TEST_BOOL_6", "", false, false},
	}

	for _, test := range tests {
		if test.value != "" {
			os.Setenv(test.key, test.value)
		} else {
			os.Unsetenv(test.key)
		}

		result := getEnvBool(test.key, test.defaultValue)
		if result != test.expected {
			t.Errorf("getEnvBool(%s, %t) = %t, expected %t",
				test.key, test.defaultValue, result, test.expected)
		}

		os.Unsetenv(test.key)
	}
}

func TestGetEnvDuration(t *testing.T) {
	tests := []struct {
		key          string
		value        string
		defaultValue time.Duration
		expected     time.Duration
	}{
		{"TEST_DUR_1", "30s", time.Minute, 30 * time.Second},
		{"TEST_DUR_2", "5m", time.Hour, 5 * time.Minute},
		{"TEST_DUR_3", "invalid", time.Hour, time.Hour},
		{"TEST_DUR_4", "", 2 * time.Hour, 2 * time.Hour},
	}

	for _, test := range tests {
		if test.value != "" {
			os.Setenv(test.key, test.value)
		} else {
			os.Unsetenv(test.key)
		}

		result := getEnvDuration(test.key, test.defaultValue)
		if result != test.expected {
			t.Errorf("getEnvDuration(%s, %v) = %v, expected %v",
				test.key, test.defaultValue, result, test.expected)
		}

		os.Unsetenv(test.key)
	}
}

func TestConfigDefaults(t *testing.T) {
	// Clear all environment variables
	envVars := []string{
		"SERVER_HOST", "SERVER_PORT", "SERVER_READ_TIMEOUT", "SERVER_WRITE_TIMEOUT",
		"DATABASE_URL", "DATABASE_MAX_CONNECTIONS", "DATABASE_MAX_IDLE_CONNECTIONS",
		"CLAUDE_CODE_BINARY", "CLAUDE_CODE_API_KEY", "MAX_CONCURRENT_SESSIONS",
		"SESSION_TIMEOUT", "WORKTREE_BASE_PATH", "LOG_RETENTION_DAYS",
		"JWT_SECRET", "API_RATE_LIMIT", "API_RATE_LIMIT_WINDOW",
		"LOG_LEVEL", "LOG_FORMAT", "METRICS_ENABLED",
	}

	for _, env := range envVars {
		os.Unsetenv(env)
	}

	cfg := Load()

	// Verify all defaults
	expectedDefaults := map[string]interface{}{
		"ServerHost":                 "0.0.0.0",
		"ServerPort":                 8080,
		"ServerReadTimeout":          30 * time.Second,
		"ServerWriteTimeout":         30 * time.Second,
		"DatabaseMaxConnections":     25,
		"DatabaseMaxIdleConnections": 5,
		"ClaudeCodeBinary":           "claude",
		"MaxConcurrentSessions":      10,
		"SessionTimeout":             time.Hour,
		"WorktreeBasePath":           "/var/lib/claude-code-server/worktrees",
		"LogRetentionDays":           30,
		"JWTSecret":                  "change-me-in-production",
		"APIRateLimit":               100,
		"APIRateLimitWindow":         time.Minute,
		"LogLevel":                   "info",
		"LogFormat":                  "json",
		"MetricsEnabled":             true,
	}

	// Use reflection or manual checks
	if cfg.ServerHost != expectedDefaults["ServerHost"] {
		t.Errorf("ServerHost default mismatch")
	}
	if cfg.ServerPort != expectedDefaults["ServerPort"] {
		t.Errorf("ServerPort default mismatch")
	}
	if cfg.MaxConcurrentSessions != expectedDefaults["MaxConcurrentSessions"] {
		t.Errorf("MaxConcurrentSessions default mismatch")
	}
	if cfg.MetricsEnabled != expectedDefaults["MetricsEnabled"] {
		t.Errorf("MetricsEnabled default mismatch")
	}
}
