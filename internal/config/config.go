package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all application configurations
// All sensitive values are loaded from .env
type Config struct {
	// Server Configuration
	Environment string
	ServerPort  string

	// DB configuration
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	// Redis configuration
	RedisAddr     string
	RedisPassword string
	RedisDB       int
	CacheTTL      time.Duration

	// Application settings
	BaseURL              string // Base URL for generating short links
	ShortCodeLength      int    // Length of generated short codes
	RateLimitPerMinute   int    // Rate limit per IP address
	URLExpirationDays    int    // Days before URLs expire (0 = never)
	EnableAuthentication bool   // Enable API key authentication
	APIKey               string // API key for protected endpoints	
}

// LoadConfig loads configuration from environment variables
// Returns error if required environment variables are missing
func LoadConfig() (*Config, error) {
	cfg := &Config{
		// Server defaults
		Environment: getEnv("ENVIRONMENT", "development"),
		ServerPort:  getEnv("SERVER_PORT", "8081"),

		// Database configuration (required)
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", ""),
		DBName:     getEnv("DB_NAME", "urlshortener"),
		DBSSLMode:  getEnv("DB_SSL_MODE", "disable"),

		// Redis configuration
		RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvAsInt("REDIS_DB", 0),
		CacheTTL:      time.Duration(getEnvAsInt("CACHE_TTL_SECONDS", 3600)) * time.Second,

		// Application settings
		BaseURL:              getEnv("BASE_URL", "http://localhost:8081"),
		ShortCodeLength:      getEnvAsInt("SHORT_CODE_LENGTH", 7),
		RateLimitPerMinute:   getEnvAsInt("RATE_LIMIT_PER_MINUTE", 60),
		URLExpirationDays:    getEnvAsInt("URL_EXPIRATION_DAYS", 0),
		EnableAuthentication: getEnvAsBool("ENABLE_AUTHENTICATION", false),
		APIKey:               getEnv("API_KEY", ""),
	}

	// Validate required configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return cfg, nil
}

// Validate checks if all required configuration is present and valid
func (c *Config) Validate() error {
	// Validate database password in production
	if c.Environment == "production" && c.DBPassword == "" {
		return fmt.Errorf("DB_PASSWORD is required in production")
	}

	// Validate short code length (must be between 4 and 12)
	if c.ShortCodeLength < 4 || c.ShortCodeLength > 12 {
		return fmt.Errorf("SHORT_CODE_LENGTH must be between 4 and 12, got %d", c.ShortCodeLength)
	}

	// Validate base URL
	if c.BaseURL == "" {
		return fmt.Errorf("BASE_URL is required")
	}

	// Validate API key if authentication is enabled
	if c.EnableAuthentication && c.APIKey == "" {
		return fmt.Errorf("API_KEY is required when ENABLE_AUTHENTICATION is true")
	}

	return nil
}

// IsDevelopment returns true if running in development mode
func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}

// IsProduction returns true if running in production mode
func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}

// Helper functions for reading environment variables

// getEnv reads an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt reads an environment variable as integer or returns default
func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	
	return value
}

// getEnvAsBool reads an environment variable as boolean or returns default
func getEnvAsBool(key string, defaultValue bool) bool {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	
	value, err := strconv.ParseBool(valueStr)
	if err != nil {
		return defaultValue
	}
	
	return value
}