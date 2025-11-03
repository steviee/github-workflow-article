package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		expected *Config
		wantErr  bool
	}{
		{
			name: "all environment variables set",
			envVars: map[string]string{
				"PORT":                   "9090",
				"CACHE_TTL":              "10m",
				"MAX_IMAGE_SIZE":         "104857600", // 100MB
				"MAX_OUTPUT_DIMENSION":   "2000",
				"LOG_LEVEL":              "debug",
				"CACHE_CLEANUP_INTERVAL": "1m",
			},
			expected: &Config{
				Port:                 "9090",
				CacheTTL:             10 * time.Minute,
				MaxImageSize:         104857600,
				MaxOutputDimension:   2000,
				LogLevel:             "debug",
				CacheCleanupInterval: 1 * time.Minute,
			},
			wantErr: false,
		},
		{
			name:    "no environment variables - use defaults",
			envVars: map[string]string{},
			expected: &Config{
				Port:                 "8080",
				CacheTTL:             5 * time.Minute,
				MaxImageSize:         52428800, // 50MB
				MaxOutputDimension:   1400,
				LogLevel:             "info",
				CacheCleanupInterval: 30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "partial environment variables set",
			envVars: map[string]string{
				"PORT":      "3000",
				"LOG_LEVEL": "warn",
			},
			expected: &Config{
				Port:                 "3000",
				CacheTTL:             5 * time.Minute,
				MaxImageSize:         52428800,
				MaxOutputDimension:   1400,
				LogLevel:             "warn",
				CacheCleanupInterval: 30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "valid minimum values",
			envVars: map[string]string{
				"CACHE_TTL":              "1s",
				"MAX_IMAGE_SIZE":         "1",
				"MAX_OUTPUT_DIMENSION":   "1",
				"CACHE_CLEANUP_INTERVAL": "1s",
			},
			expected: &Config{
				Port:                 "8080",
				CacheTTL:             1 * time.Second,
				MaxImageSize:         1,
				MaxOutputDimension:   1,
				LogLevel:             "info",
				CacheCleanupInterval: 1 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "empty PORT falls back to default",
			envVars: map[string]string{
				"PORT": "",
			},
			expected: &Config{
				Port:                 "8080",
				CacheTTL:             5 * time.Minute,
				MaxImageSize:         52428800,
				MaxOutputDimension:   1400,
				LogLevel:             "info",
				CacheCleanupInterval: 30 * time.Second,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for key, value := range tt.envVars {
				t.Setenv(key, value)
			}

			// Load config
			cfg, err := LoadConfig()

			// Check error expectation
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, cfg)

			// Validate all fields
			assert.Equal(t, tt.expected.Port, cfg.Port)
			assert.Equal(t, tt.expected.CacheTTL, cfg.CacheTTL)
			assert.Equal(t, tt.expected.MaxImageSize, cfg.MaxImageSize)
			assert.Equal(t, tt.expected.MaxOutputDimension, cfg.MaxOutputDimension)
			assert.Equal(t, tt.expected.LogLevel, cfg.LogLevel)
			assert.Equal(t, tt.expected.CacheCleanupInterval, cfg.CacheCleanupInterval)
		})
	}
}

func TestLoadConfig_InvalidValues(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		wantErr bool
		errMsg  string
	}{
		{
			name: "invalid CACHE_TTL - negative",
			envVars: map[string]string{
				"CACHE_TTL": "-5m",
			},
			wantErr: true,
			errMsg:  "CACHE_TTL must be positive",
		},
		{
			name: "invalid CACHE_TTL - too large",
			envVars: map[string]string{
				"CACHE_TTL": "25h",
			},
			wantErr: true,
			errMsg:  "CACHE_TTL too large",
		},
		{
			name: "invalid MAX_IMAGE_SIZE - negative",
			envVars: map[string]string{
				"MAX_IMAGE_SIZE": "-1000",
			},
			wantErr: true,
			errMsg:  "MAX_IMAGE_SIZE must be positive",
		},
		{
			name: "invalid MAX_IMAGE_SIZE - too large",
			envVars: map[string]string{
				"MAX_IMAGE_SIZE": "2000000000", // 2GB
			},
			wantErr: true,
			errMsg:  "MAX_IMAGE_SIZE too large",
		},
		{
			name: "invalid MAX_OUTPUT_DIMENSION - zero",
			envVars: map[string]string{
				"MAX_OUTPUT_DIMENSION": "0",
			},
			wantErr: true,
			errMsg:  "MAX_OUTPUT_DIMENSION must be positive",
		},
		{
			name: "invalid MAX_OUTPUT_DIMENSION - negative",
			envVars: map[string]string{
				"MAX_OUTPUT_DIMENSION": "-100",
			},
			wantErr: true,
			errMsg:  "MAX_OUTPUT_DIMENSION must be positive",
		},
		{
			name: "invalid MAX_OUTPUT_DIMENSION - too large",
			envVars: map[string]string{
				"MAX_OUTPUT_DIMENSION": "20000",
			},
			wantErr: true,
			errMsg:  "MAX_OUTPUT_DIMENSION too large",
		},
		{
			name: "invalid LOG_LEVEL - unsupported",
			envVars: map[string]string{
				"LOG_LEVEL": "trace",
			},
			wantErr: true,
			errMsg:  "LOG_LEVEL must be one of",
		},
		{
			name: "invalid CACHE_CLEANUP_INTERVAL - negative",
			envVars: map[string]string{
				"CACHE_CLEANUP_INTERVAL": "-10s",
			},
			wantErr: true,
			errMsg:  "CACHE_CLEANUP_INTERVAL must be positive",
		},
		{
			name: "invalid CACHE_CLEANUP_INTERVAL - too large",
			envVars: map[string]string{
				"CACHE_CLEANUP_INTERVAL": "2h",
			},
			wantErr: true,
			errMsg:  "CACHE_CLEANUP_INTERVAL too large",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for key, value := range tt.envVars {
				t.Setenv(key, value)
			}

			// Load config
			cfg, err := LoadConfig()

			// Check error expectation
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, cfg)
			} else {
				require.NoError(t, err)
				require.NotNil(t, cfg)
			}
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: &Config{
				Port:                 "8080",
				CacheTTL:             5 * time.Minute,
				MaxImageSize:         52428800,
				MaxOutputDimension:   1400,
				LogLevel:             "info",
				CacheCleanupInterval: 30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "empty port",
			config: &Config{
				Port:                 "",
				CacheTTL:             5 * time.Minute,
				MaxImageSize:         52428800,
				MaxOutputDimension:   1400,
				LogLevel:             "info",
				CacheCleanupInterval: 30 * time.Second,
			},
			wantErr: true,
			errMsg:  "PORT cannot be empty",
		},
		{
			name: "zero cache TTL",
			config: &Config{
				Port:                 "8080",
				CacheTTL:             0,
				MaxImageSize:         52428800,
				MaxOutputDimension:   1400,
				LogLevel:             "info",
				CacheCleanupInterval: 30 * time.Second,
			},
			wantErr: true,
			errMsg:  "CACHE_TTL must be positive",
		},
		{
			name: "negative cache TTL",
			config: &Config{
				Port:                 "8080",
				CacheTTL:             -1 * time.Minute,
				MaxImageSize:         52428800,
				MaxOutputDimension:   1400,
				LogLevel:             "info",
				CacheCleanupInterval: 30 * time.Second,
			},
			wantErr: true,
			errMsg:  "CACHE_TTL must be positive",
		},
		{
			name: "cache TTL too large",
			config: &Config{
				Port:                 "8080",
				CacheTTL:             25 * time.Hour,
				MaxImageSize:         52428800,
				MaxOutputDimension:   1400,
				LogLevel:             "info",
				CacheCleanupInterval: 30 * time.Second,
			},
			wantErr: true,
			errMsg:  "CACHE_TTL too large",
		},
		{
			name: "zero max image size",
			config: &Config{
				Port:                 "8080",
				CacheTTL:             5 * time.Minute,
				MaxImageSize:         0,
				MaxOutputDimension:   1400,
				LogLevel:             "info",
				CacheCleanupInterval: 30 * time.Second,
			},
			wantErr: true,
			errMsg:  "MAX_IMAGE_SIZE must be positive",
		},
		{
			name: "negative max image size",
			config: &Config{
				Port:                 "8080",
				CacheTTL:             5 * time.Minute,
				MaxImageSize:         -1000,
				MaxOutputDimension:   1400,
				LogLevel:             "info",
				CacheCleanupInterval: 30 * time.Second,
			},
			wantErr: true,
			errMsg:  "MAX_IMAGE_SIZE must be positive",
		},
		{
			name: "max image size too large",
			config: &Config{
				Port:                 "8080",
				CacheTTL:             5 * time.Minute,
				MaxImageSize:         2 * 1024 * 1024 * 1024, // 2GB
				MaxOutputDimension:   1400,
				LogLevel:             "info",
				CacheCleanupInterval: 30 * time.Second,
			},
			wantErr: true,
			errMsg:  "MAX_IMAGE_SIZE too large",
		},
		{
			name: "zero max output dimension",
			config: &Config{
				Port:                 "8080",
				CacheTTL:             5 * time.Minute,
				MaxImageSize:         52428800,
				MaxOutputDimension:   0,
				LogLevel:             "info",
				CacheCleanupInterval: 30 * time.Second,
			},
			wantErr: true,
			errMsg:  "MAX_OUTPUT_DIMENSION must be positive",
		},
		{
			name: "negative max output dimension",
			config: &Config{
				Port:                 "8080",
				CacheTTL:             5 * time.Minute,
				MaxImageSize:         52428800,
				MaxOutputDimension:   -100,
				LogLevel:             "info",
				CacheCleanupInterval: 30 * time.Second,
			},
			wantErr: true,
			errMsg:  "MAX_OUTPUT_DIMENSION must be positive",
		},
		{
			name: "max output dimension too large",
			config: &Config{
				Port:                 "8080",
				CacheTTL:             5 * time.Minute,
				MaxImageSize:         52428800,
				MaxOutputDimension:   20000,
				LogLevel:             "info",
				CacheCleanupInterval: 30 * time.Second,
			},
			wantErr: true,
			errMsg:  "MAX_OUTPUT_DIMENSION too large",
		},
		{
			name: "invalid log level",
			config: &Config{
				Port:                 "8080",
				CacheTTL:             5 * time.Minute,
				MaxImageSize:         52428800,
				MaxOutputDimension:   1400,
				LogLevel:             "trace",
				CacheCleanupInterval: 30 * time.Second,
			},
			wantErr: true,
			errMsg:  "LOG_LEVEL must be one of",
		},
		{
			name: "all valid log levels - debug",
			config: &Config{
				Port:                 "8080",
				CacheTTL:             5 * time.Minute,
				MaxImageSize:         52428800,
				MaxOutputDimension:   1400,
				LogLevel:             "debug",
				CacheCleanupInterval: 30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "all valid log levels - warn",
			config: &Config{
				Port:                 "8080",
				CacheTTL:             5 * time.Minute,
				MaxImageSize:         52428800,
				MaxOutputDimension:   1400,
				LogLevel:             "warn",
				CacheCleanupInterval: 30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "all valid log levels - error",
			config: &Config{
				Port:                 "8080",
				CacheTTL:             5 * time.Minute,
				MaxImageSize:         52428800,
				MaxOutputDimension:   1400,
				LogLevel:             "error",
				CacheCleanupInterval: 30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "zero cache cleanup interval",
			config: &Config{
				Port:                 "8080",
				CacheTTL:             5 * time.Minute,
				MaxImageSize:         52428800,
				MaxOutputDimension:   1400,
				LogLevel:             "info",
				CacheCleanupInterval: 0,
			},
			wantErr: true,
			errMsg:  "CACHE_CLEANUP_INTERVAL must be positive",
		},
		{
			name: "negative cache cleanup interval",
			config: &Config{
				Port:                 "8080",
				CacheTTL:             5 * time.Minute,
				MaxImageSize:         52428800,
				MaxOutputDimension:   1400,
				LogLevel:             "info",
				CacheCleanupInterval: -10 * time.Second,
			},
			wantErr: true,
			errMsg:  "CACHE_CLEANUP_INTERVAL must be positive",
		},
		{
			name: "cache cleanup interval too large",
			config: &Config{
				Port:                 "8080",
				CacheTTL:             5 * time.Minute,
				MaxImageSize:         52428800,
				MaxOutputDimension:   1400,
				LogLevel:             "info",
				CacheCleanupInterval: 2 * time.Hour,
			},
			wantErr: true,
			errMsg:  "CACHE_CLEANUP_INTERVAL too large",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.config.Validate()

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestParseDurationOrDefault_InvalidFormat(t *testing.T) {
	t.Setenv("CACHE_TTL", "invalid-duration")

	cfg, err := LoadConfig()
	require.NoError(t, err)
	// Should fall back to default value when parsing fails
	assert.Equal(t, 5*time.Minute, cfg.CacheTTL)
}

func TestParseIntOrDefault_InvalidFormat(t *testing.T) {
	t.Setenv("MAX_OUTPUT_DIMENSION", "not-a-number")

	cfg, err := LoadConfig()
	require.NoError(t, err)
	// Should fall back to default value when parsing fails
	assert.Equal(t, 1400, cfg.MaxOutputDimension)
}

func TestParseInt64OrDefault_InvalidFormat(t *testing.T) {
	t.Setenv("MAX_IMAGE_SIZE", "not-a-number")

	cfg, err := LoadConfig()
	require.NoError(t, err)
	// Should fall back to default value when parsing fails
	assert.Equal(t, int64(52428800), cfg.MaxImageSize)
}
