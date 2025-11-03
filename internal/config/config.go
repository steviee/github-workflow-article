package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all application configuration loaded from environment variables
type Config struct {
	Port                 string        // HTTP server port
	CacheTTL             time.Duration // Cache expiration time
	MaxImageSize         int64         // Max source image size in bytes
	MaxOutputDimension   int           // Max output width/height in pixels
	LogLevel             string        // Logging level (debug, info, warn, error)
	CacheCleanupInterval time.Duration // Cache cleanup frequency
}

// LoadConfig loads configuration from environment variables with sensible defaults
// and validates all values before returning.
func LoadConfig() (*Config, error) {
	cfg := &Config{
		Port:                 getEnvOrDefault("PORT", "8080"),
		CacheTTL:             parseDurationOrDefault("CACHE_TTL", 5*time.Minute),
		MaxImageSize:         parseInt64OrDefault("MAX_IMAGE_SIZE", 52428800), // 50MB
		MaxOutputDimension:   parseIntOrDefault("MAX_OUTPUT_DIMENSION", 1400),
		LogLevel:             getEnvOrDefault("LOG_LEVEL", "info"),
		CacheCleanupInterval: parseDurationOrDefault("CACHE_CLEANUP_INTERVAL", 30*time.Second),
	}

	// Validate configuration values
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// Validate checks that all configuration values are reasonable
func (c *Config) Validate() error {
	// Validate port
	if c.Port == "" {
		return fmt.Errorf("PORT cannot be empty")
	}

	// Validate cache TTL
	if c.CacheTTL <= 0 {
		return fmt.Errorf("CACHE_TTL must be positive, got %v", c.CacheTTL)
	}
	if c.CacheTTL > 24*time.Hour {
		return fmt.Errorf("CACHE_TTL too large, max 24h, got %v", c.CacheTTL)
	}

	// Validate max image size
	if c.MaxImageSize <= 0 {
		return fmt.Errorf("MAX_IMAGE_SIZE must be positive, got %d", c.MaxImageSize)
	}
	if c.MaxImageSize > 1024*1024*1024 { // 1GB max
		return fmt.Errorf("MAX_IMAGE_SIZE too large, max 1GB, got %d", c.MaxImageSize)
	}

	// Validate max output dimension
	if c.MaxOutputDimension <= 0 {
		return fmt.Errorf("MAX_OUTPUT_DIMENSION must be positive, got %d", c.MaxOutputDimension)
	}
	if c.MaxOutputDimension > 10000 {
		return fmt.Errorf("MAX_OUTPUT_DIMENSION too large, max 10000, got %d", c.MaxOutputDimension)
	}

	// Validate log level
	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLogLevels[c.LogLevel] {
		return fmt.Errorf("LOG_LEVEL must be one of [debug, info, warn, error], got %q", c.LogLevel)
	}

	// Validate cache cleanup interval
	if c.CacheCleanupInterval <= 0 {
		return fmt.Errorf("CACHE_CLEANUP_INTERVAL must be positive, got %v", c.CacheCleanupInterval)
	}
	if c.CacheCleanupInterval > 1*time.Hour {
		return fmt.Errorf("CACHE_CLEANUP_INTERVAL too large, max 1h, got %v", c.CacheCleanupInterval)
	}

	return nil
}

// getEnvOrDefault retrieves an environment variable or returns the default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// parseDurationOrDefault parses a duration from an environment variable or returns default
func parseDurationOrDefault(key string, defaultValue time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		// If parsing fails, return default and log warning would be nice
		// but we don't have logger at config loading time, so just return default
		return defaultValue
	}

	return duration
}

// parseIntOrDefault parses an integer from an environment variable or returns default
func parseIntOrDefault(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	intValue, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}

	return intValue
}

// parseInt64OrDefault parses an int64 from an environment variable or returns default
func parseInt64OrDefault(key string, defaultValue int64) int64 {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	intValue, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return defaultValue
	}

	return intValue
}
