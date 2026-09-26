// Package testutil provides shared helpers for integration tests.
//
// It boots a real embedded PostgreSQL server (no Docker, no external
// services) so repository tests exercise the actual migrations, seed data,
// RLS policies, and triggers.
package testutil

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"

	"ai-governance-saas/backend/internal/infrastructure/postgres"
)

// TestPostgres is a booted embedded PostgreSQL server with the application
// store bound to it. Migrations are always applied; seed data is applied
// when requested at boot.
type TestPostgres struct {
	store *postgres.Store
	pg    *embeddedpostgres.EmbeddedPostgres
	dir   string
}

// Store returns the application store bound to the test server.
func (tp *TestPostgres) Store() *postgres.Store {
	return tp.store
}

// Stop shuts down the embedded server, closes the store, and removes the
// temporary data directory. It is safe to call multiple times.
func (tp *TestPostgres) Stop() {
	if tp.store != nil {
		tp.store.Close()
		tp.store = nil
	}
	if tp.pg != nil {
		_ = tp.pg.Stop()
		tp.pg = nil
	}
	if tp.dir != "" {
		_ = os.RemoveAll(tp.dir)
		tp.dir = ""
	}
}

// StartPostgres boots an embedded PostgreSQL server on a free local port,
// applies all forward migrations and, when seed is true, loads the seed
// data. The data directory is created fresh per boot; the Postgres
// distribution bytes are cached under the user cache directory so they are
// downloaded once. The caller must call Stop when done.
func StartPostgres(ctx context.Context, seed bool) (*TestPostgres, error) {
	port, err := freePort()
	if err != nil {
		return nil, fmt.Errorf("reserve postgres port: %w", err)
	}

	dir, err := os.MkdirTemp("", "aigw-test-postgres-*")
	if err != nil {
		return nil, fmt.Errorf("create postgres data dir: %w", err)
	}

	cacheRoot, err := os.UserCacheDir()
	if err != nil {
		cacheRoot = os.TempDir()
	}
	cachePath := filepath.Join(cacheRoot, "aigw", "embedded-postgres")
	binariesPath := filepath.Join(cachePath, "binaries")
	if err := os.MkdirAll(binariesPath, 0o755); err != nil {
		_ = os.RemoveAll(dir)
		return nil, fmt.Errorf("create binaries dir: %w", err)
	}

	dataPath := filepath.Join(dir, "data")
	runtimePath := filepath.Join(dir, "runtime")
	if err := os.MkdirAll(dataPath, 0o755); err != nil {
		_ = os.RemoveAll(dir)
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	if err := os.MkdirAll(runtimePath, 0o755); err != nil {
		_ = os.RemoveAll(dir)
		return nil, fmt.Errorf("create runtime dir: %w", err)
	}

	cfg := embeddedpostgres.DefaultConfig().
		DataPath(dataPath).
		RuntimePath(runtimePath).
		BinariesPath(binariesPath).
		CachePath(cachePath).
		Port(port).
		StartTimeout(5 * time.Minute).
		Logger(io.Discard)

	epg := embeddedpostgres.NewDatabase(cfg)
	if err := epg.Start(); err != nil {
		_ = os.RemoveAll(dir)
		return nil, fmt.Errorf("start embedded postgres: %w", err)
	}
	cleanup := func() {
		_ = epg.Stop()
		_ = os.RemoveAll(dir)
	}

	dsn := fmt.Sprintf("postgres://postgres:postgres@127.0.0.1:%d/postgres?sslmode=disable", port)
	store, err := postgres.NewStore(ctx, dsn)
	if err != nil {
		cleanup()
		return nil, fmt.Errorf("connect to postgres: %w", err)
	}
	tp := &TestPostgres{store: store, pg: epg, dir: dir}

	if err := postgres.Migrate(ctx, store.Pool()); err != nil {
		tp.Stop()
		return nil, fmt.Errorf("run migrations: %w", err)
	}
	if seed {
		if err := postgres.Seed(ctx, store.Pool()); err != nil {
			tp.Stop()
			return nil, fmt.Errorf("seed postgres: %w", err)
		}
	}
	return tp, nil
}

// StoreForTest boots an embedded PostgreSQL server for one test and
// registers teardown via t.Cleanup. It returns the application store.
func StoreForTest(t *testing.T, seed bool) *postgres.Store {
	t.Helper()
	tp, err := StartPostgres(context.Background(), seed)
	if err != nil {
		t.Fatalf("boot embedded postgres: %v", err)
	}
	t.Cleanup(tp.Stop)
	return tp.Store()
}

// freePort reserves an ephemeral localhost port and returns its number.
func freePort() (uint32, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer ln.Close()
	return uint32(ln.Addr().(*net.TCPAddr).Port), nil
}
