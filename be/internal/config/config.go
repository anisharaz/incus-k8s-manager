package config

import (
	"os"
)

// Config holds the application configuration
type Config struct {
	Port string
	Env  string
}

// NewConfig creates a new configuration instance
func NewConfig() *Config {
	return &Config{
		Port: getEnv("PORT", "8000"),
		Env:  getEnv("ENV", "development"),
	}
}

// getEnv retrieves environment variable or returns a default value
func getEnv(key, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultVal
}
