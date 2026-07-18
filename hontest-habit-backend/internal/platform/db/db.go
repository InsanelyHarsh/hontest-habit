package db

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"log/slog"
	"net/url"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/viper"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Config holds Postgres connection parameters.
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
	SSLMode  string
}

// ConfigFromEnv reads DB_* keys via viper.
func ConfigFromEnv() Config {
	return Config{
		Host:     viper.GetString("DB_HOST"),
		Port:     viper.GetInt("DB_PORT"),
		User:     viper.GetString("DB_USER"),
		Password: viper.GetString("DB_PASSWORD"),
		Name:     viper.GetString("DB_NAME"),
		SSLMode:  viper.GetString("DB_SSLMODE"),
	}
}

func (c Config) dsn(scheme string) string {
	u := url.URL{
		Scheme: scheme,
		User:   url.UserPassword(c.User, c.Password),
		Host:   fmt.Sprintf("%s:%d", c.Host, c.Port),
		Path:   "/" + c.Name,
	}
	q := url.Values{}
	q.Set("sslmode", c.SSLMode)
	u.RawQuery = q.Encode()
	return u.String()
}

func (c Config) postgresDSN() string { return c.dsn("postgres") }
func (c Config) migrateDSN() string  { return c.dsn("pgx5") }

// New opens a pgxpool connection pool and verifies connectivity. The caller
// is responsible for calling pool.Close().
func New(ctx context.Context, cfg Config) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, cfg.postgresDSN())
	if err != nil {
		return nil, fmt.Errorf("db: create pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("db: ping: %w", err)
	}
	slog.Info("db: connected", "host", cfg.Host, "port", cfg.Port, "name", cfg.Name)
	return pool, nil
}

// RunMigrations applies all pending embedded migrations. Safe to call on
// every process start; a no-op when the schema is already current.
func RunMigrations(cfg Config) error {
	src, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("db: load migration source: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", src, cfg.migrateDSN())
	if err != nil {
		return fmt.Errorf("db: init migrator: %w", err)
	}
	defer func() {
		if srcErr, dbErr := m.Close(); srcErr != nil || dbErr != nil {
			slog.Error("db: migrator close error", "source_err", srcErr, "db_err", dbErr)
		}
	}()

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			slog.Info("db: no pending migrations")
			return nil
		}
		return fmt.Errorf("db: apply migrations: %w", err)
	}
	slog.Info("db: migrations applied")
	return nil
}
