// Package db — MySQL connection pool and helpers.
package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// Config holds database connection settings.
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
	// MaxOpenConns is the maximum number of open connections. Default: 25.
	MaxOpenConns int
	// MaxIdleConns is the maximum number of idle connections. Default: 10.
	MaxIdleConns int
	// ConnMaxLifetime is the maximum connection lifetime. Default: 5m.
	ConnMaxLifetime time.Duration
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		Host:            "localhost",
		Port:            3306,
		User:            "root",
		Password:        "8759",
		Name:            "directus",
		MaxOpenConns:    25,
		MaxIdleConns:    10,
		ConnMaxLifetime: 5 * time.Minute,
	}
}

// DB wraps *sql.DB with convenience helpers.
type DB struct {
	*sql.DB
}

// Open connects to MySQL and validates the connection.
func Open(cfg Config) (*DB, error) {
	if cfg.MaxOpenConns == 0 {
		cfg.MaxOpenConns = 25
	}
	if cfg.MaxIdleConns == 0 {
		cfg.MaxIdleConns = 10
	}
	if cfg.ConnMaxLifetime == 0 {
		cfg.ConnMaxLifetime = 5 * time.Minute
	}

	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=true&loc=UTC&multiStatements=true",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name,
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("db.Ping: %w", err)
	}

	log.Printf("[db] connected to %s:%d/%s", cfg.Host, cfg.Port, cfg.Name)
	return &DB{db}, nil
}

// MustOpen calls Open and panics on error. Use only in main/tests.
func MustOpen(cfg Config) *DB {
	d, err := Open(cfg)
	if err != nil {
		panic(err)
	}
	return d
}

// Transact executes fn inside a transaction.
// Commits on nil return, rolls back on any error.
func (d *DB) Transact(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := d.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("tx err: %v, rollback err: %v", err, rbErr)
		}
		return err
	}
	return tx.Commit()
}

// QueryContext is a thin wrapper that logs slow queries.
func (d *DB) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	start := time.Now()
	rows, err := d.DB.QueryContext(ctx, query, args...)
	if elapsed := time.Since(start); elapsed > 200*time.Millisecond {
		log.Printf("[db] SLOW QUERY (%s): %s", elapsed, query)
	}
	return rows, err
}

// Migrate runs the DDL script at schemaPath against the database.
func (d *DB) Migrate(schemaPath string) error {
	content, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("read schema file: %w", err)
	}

	// Executing the whole file works because multiStatements=true is in the DSN.
	_, err = d.Exec(string(content))
	if err != nil {
		return fmt.Errorf("exec schema: %w", err)
	}

	log.Printf("[db] schema migrated successfully from %s", schemaPath)
	return nil
}
