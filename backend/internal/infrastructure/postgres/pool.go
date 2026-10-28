package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// customEnumTypes are the enum types created by migrations that pgx must know
// about to encode and decode their array-typed columns (e.g. scopes in
// gateway_credentials). Without registration, pgx fails to encode
// []db.GatewayCredentialScope with "cannot find encode plan".
var customEnumTypes = []string{
	"gateway_credential_status",
	"gateway_credential_scope",
	"gateway_subject_type",
}

// Store wraps the pgx connection pool and exposes transaction helpers that
// bind the tenant context. Repository implementations live in
// internal/infrastructure/postgres/repositories and are built per
// transaction from *db.Queries, so they never hold the pool directly.
type Store struct {
	pool *pgxpool.Pool
}

// PoolOption configures connection pool parameters.
type PoolOption func(*pgxpool.Config)

// WithMaxConns sets the maximum connection pool size.
func WithMaxConns(n int32) PoolOption {
	return func(c *pgxpool.Config) {
		if n > 0 {
			c.MaxConns = n
		}
	}
}

// WithMinConns sets the minimum number of idle connections kept alive.
func WithMinConns(n int32) PoolOption {
	return func(c *pgxpool.Config) {
		if n >= 0 {
			c.MinConns = n
		}
	}
}

// WithMaxConnLifetime sets the maximum duration a connection may be reused.
func WithMaxConnLifetime(d time.Duration) PoolOption {
	return func(c *pgxpool.Config) {
		if d > 0 {
			c.MaxConnLifetime = d
		}
	}
}

// WithMaxConnIdleTime sets the maximum duration an idle connection is retained.
func WithMaxConnIdleTime(d time.Duration) PoolOption {
	return func(c *pgxpool.Config) {
		if d > 0 {
			c.MaxConnIdleTime = d
		}
	}
}

// WithHealthCheckPeriod sets the interval between background connection liveness checks.
func WithHealthCheckPeriod(d time.Duration) PoolOption {
	return func(c *pgxpool.Config) {
		if d > 0 {
			c.HealthCheckPeriod = d
		}
	}
}

// NewStore parses the connection string, builds the pool with resource-conscious defaults,
// applies any caller-specified options, and verifies the connection with a ping before returning.
func NewStore(ctx context.Context, databaseURL string, opts ...PoolOption) (*Store, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}

	// Resource-optimized connection pooling defaults:
	// - MaxConns bounds maximum concurrent active DB handles
	// - MinConns maintains warm connections ready for low-latency dispatch
	// - MaxConnIdleTime aggressively reclaims idle socket buffers and Postgres backend memory
	// - HealthCheckPeriod detects severed network sockets proactively
	cfg.MaxConns = 20
	cfg.MinConns = 2
	cfg.MaxConnLifetime = 30 * time.Minute
	cfg.MaxConnIdleTime = 5 * time.Minute
	cfg.HealthCheckPeriod = 1 * time.Minute

	for _, opt := range opts {
		opt(cfg)
	}

	cfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		return registerCustomTypes(ctx, conn)
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create connection pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return &Store{pool: pool}, nil
}

// Close releases all pooled connections.
func (s *Store) Close() {
	s.pool.Close()
}

// Pool exposes the underlying pool for advanced use, such as binding a raw
// connection to a non-transactional query runner.
func (s *Store) Pool() *pgxpool.Pool {
	return s.pool
}

// registerCustomTypes registers each custom enum and its array type with the
// connection's type map so pgx can encode query arguments of those types.
// Enums are registered before their arrays because LoadType for an array type
// requires the element type to already be registered. Types that do not exist
// yet (the pool is created before migrations run, e.g. in tests) are skipped;
// Migrate resets the pool afterwards so fresh connections re-run this hook.
func registerCustomTypes(ctx context.Context, conn *pgx.Conn) error {
	for _, name := range customEnumTypes {
		if err := registerTypeIfExists(ctx, conn, name); err != nil {
			return fmt.Errorf("register enum type %s: %w", name, err)
		}
	}
	for _, name := range customEnumTypes {
		if err := registerTypeIfExists(ctx, conn, "_"+name); err != nil {
			return fmt.Errorf("register array type _%s: %w", name, err)
		}
	}
	return nil
}

func registerTypeIfExists(ctx context.Context, conn *pgx.Conn, name string) error {
	var exists bool
	if err := conn.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM pg_type WHERE typname = $1)", name).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return nil
	}
	t, err := conn.LoadType(ctx, name)
	if err != nil {
		return err
	}
	conn.TypeMap().RegisterType(t)
	return nil
}
