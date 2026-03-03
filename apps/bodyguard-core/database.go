package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

// DatabaseType represents the type of database
type DatabaseType string

const (
	SQLite  DatabaseType = "sqlite"
	Postgres DatabaseType = "postgres"
)

// DBConfig holds database configuration
type DBConfig struct {
	Type     DatabaseType
	SQLite   SQLiteConfig
	Postgres PostgresConfig
}

// SQLiteConfig holds SQLite-specific configuration
type SQLiteConfig struct {
	Path string
}

// PostgresConfig holds PostgreSQL-specific configuration
type PostgresConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Database string
	SSLMode  string
}

// OpenDB opens a database connection based on the configuration
func OpenDB(config DBConfig) (*sql.DB, error) {
	switch config.Type {
	case SQLite:
		return openSQLite(config.SQLite.Path)
	case Postgres:
		return openPostgres(config.Postgres)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", config.Type)
	}
}

// openSQLite opens a SQLite database
func openSQLite(path string) (*sql.DB, error) {
	// Ensure directory exists
	if err := os.MkdirAll(getDir(path), 0o755); err != nil {
		return nil, fmt.Errorf("failed to create db directory: %w", err)
	}

	// Open database with connection pooling settings
	db, err := sql.Open("sqlite3", path+"?_busy_timeout=5000&_foreign_keys=on&_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

// openPostgres opens a PostgreSQL database
func openPostgres(config PostgresConfig) (*sql.DB, error) {
	// Build connection string
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		config.Host,
		config.Port,
		config.User,
		config.Password,
		config.Database,
		config.SSLMode,
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

// GetDBConfig returns database configuration from environment variables
func GetDBConfig() DBConfig {
	dbType := DatabaseType(env("DB_TYPE", "sqlite"))

	config := DBConfig{Type: dbType}

	switch dbType {
	case SQLite:
		config.SQLite = SQLiteConfig{
			Path: env("BG_DB", "./data/bodyguard.db"),
		}
	case Postgres:
		config.Postgres = PostgresConfig{
			Host:     env("POSTGRES_HOST", "localhost"),
			Port:     env("POSTGRES_PORT", "5432"),
			User:     env("POSTGRES_USER", "ammangate"),
			Password: env("POSTGRES_PASSWORD", ""),
			Database: env("POSTGRES_DB", "ammangate"),
			SSLMode:  env("POSTGRES_SSLMODE", "disable"),
		}
	}

	return config
}

// getDir returns the directory part of a path
func getDir(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			return path[:i]
		}
	}
	return "."
}

// GetPlaceholders returns the appropriate placeholder for the database type
func GetPlaceholders(dbType DatabaseType, count int) string {
	if dbType == Postgres {
		// PostgreSQL uses $1, $2, $3, etc.
		return buildPostgresPlaceholders(count)
	}
	// SQLite uses ?, ?, ?, etc.
	return buildSQLitePlaceholders(count)
}

// buildSQLitePlaceholders builds SQLite placeholders (?, ?, ?)
func buildSQLitePlaceholders(count int) string {
	result := ""
	for i := 0; i < count; i++ {
		if i > 0 {
			result += ", "
		}
		result += "?"
	}
	return result
}

// buildPostgresPlaceholders builds PostgreSQL placeholders ($1, $2, $3)
func buildPostgresPlaceholders(count int) string {
	result := ""
	for i := 1; i <= count; i++ {
		if i > 1 {
			result += ", "
		}
		result += fmt.Sprintf("$%d", i)
	}
	return result
}

// IsRetryableError checks if an error is retryable (e.g., lock conflict)
func IsRetryableError(dbType DatabaseType, err error) bool {
	if err == nil {
		return false
	}

	// SQLite retryable errors
	if dbType == SQLite {
		// Check for "database is locked" or "busy" errors
		errStr := err.Error()
		return contains(errStr, "database is locked") ||
			contains(errStr, "database is busy") ||
			contains(errStr, "locked")
	}

	// PostgreSQL retryable errors (connection issues, etc.)
	if dbType == Postgres {
		// Add specific PostgreSQL error codes here if needed
		return false
	}

	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
