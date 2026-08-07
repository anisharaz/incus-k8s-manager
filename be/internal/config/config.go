package config

import (
	"crypto/rand"
	"log"
	"net/url"
	"os"
)

// Config holds the application configuration
type Config struct {
	Port            string
	Env             string
	DBUrl           string
	DBUser          string
	DBPass          string
	DBHost          string
	DBPort          string
	DBName          string
	IncusSocketPath string
	JWTSecret       []byte
	// CookieSecure sets the session cookie's Secure attribute (HTTPS-only).
	// An explicit "is there TLS in front of me right now" flag (COOKIE_SECURE
	// env var) — set by the operator (the bundled Caddy proxy in
	// docker-compose, or a future Kubernetes Ingress), not inferred from ENV
	// or trusted from an X-Forwarded-Proto header (which would require
	// trusting the edge proxy to strip/overwrite client-supplied values; an
	// explicit flag is simpler and can't be spoofed). Defaults to false so
	// running the binary directly (make dev/run, no proxy in front) keeps
	// working over plain HTTP with no extra config.
	CookieSecure bool
}

// NewConfig creates a new configuration instance
func NewConfig() *Config {
	env := getEnv("ENV", "development")

	return &Config{
		Port:   getEnv("PORT", "8000"),
		Env:    env,
		DBUrl:  getEnv("DATABASE_URL", ""),
		DBUser: getEnv("DB_USER", "postgres"),
		DBPass: getEnv("DB_PASSWORD", "postgres"),
		DBHost: getEnv("DB_HOST", "localhost"),
		DBPort: getEnv("DB_PORT", "5432"),
		DBName: getEnv("DB_NAME", "incus_k8s_manager"),
		// Path to the Incus unix socket shared by the incus container (see
		// meta/incusDocker/docker-compose.yml's incus-socket-share volume).
		IncusSocketPath: getEnv("INCUS_SOCKET_PATH", "/shared-socket/incus.sock"),
		JWTSecret:       loadJWTSecret(),
		CookieSecure:    getEnv("COOKIE_SECURE", "false") == "true",
	}
}

// loadJWTSecret reads JWT_SECRET, or generates a random one if unset. A
// generated secret means every session is invalidated on restart — fine
// for now, but set JWT_SECRET in any deployment that should survive one.
func loadJWTSecret() []byte {
	if secret := getEnv("JWT_SECRET", ""); secret != "" {
		return []byte(secret)
	}

	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		log.Fatalf("failed to generate a JWT secret: %v", err)
	}

	log.Println("WARNING: JWT_SECRET not set — generated a random one for this run. " +
		"All sessions will be invalidated on restart. Set JWT_SECRET in any persistent deployment.")

	return secret
}

// GetDatabaseDSN returns the PostgreSQL connection string, always as a
// "postgres://" URL — both GORM's postgres driver and golang-migrate's
// (used by runMigrations in cmd/server/main.go) require that format.
func (c *Config) GetDatabaseDSN() string {
	if c.DBUrl != "" {
		return c.DBUrl
	}

	dsn := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(c.DBUser, c.DBPass),
		Host:     c.DBHost + ":" + c.DBPort,
		Path:     "/" + c.DBName,
		RawQuery: "sslmode=disable",
	}
	return dsn.String()
}

// getEnv retrieves environment variable or returns a default value
func getEnv(key, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultVal
}
