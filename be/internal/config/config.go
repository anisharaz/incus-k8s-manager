package config

import (
	"os"
)

// Config holds the application configuration
type Config struct {
	Port   string
	Env    string
	DBUrl  string
	DBUser string
	DBPass string
	DBHost string
	DBPort string
	DBName string
}

// NewConfig creates a new configuration instance
func NewConfig() *Config {
	return &Config{
		Port:   getEnv("PORT", "8000"),
		Env:    getEnv("ENV", "development"),
		DBUrl:  getEnv("DATABASE_URL", ""),
		DBUser: getEnv("DB_USER", "postgres"),
		DBPass: getEnv("DB_PASSWORD", "postgres"),
		DBHost: getEnv("DB_HOST", "localhost"),
		DBPort: getEnv("DB_PORT", "5432"),
		DBName: getEnv("DB_NAME", "incus_k8s"),
	}
}

// GetDatabaseDSN returns the PostgreSQL connection string
func (c *Config) GetDatabaseDSN() string {
	if c.DBUrl != "" {
		return c.DBUrl
	}
	// Format: user=<user> password=<pass> host=<host> port=<port> dbname=<dbname> sslmode=disable
	return "user=" + c.DBUser + " password=" + c.DBPass + " host=" + c.DBHost + " port=" + c.DBPort + " dbname=" + c.DBName + " sslmode=disable"
}

// getEnv retrieves environment variable or returns a default value
func getEnv(key, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultVal
}
