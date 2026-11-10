package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"ai-governance-saas/backend/internal/infrastructure/notify"
	"ai-governance-saas/backend/internal/infrastructure/postgres"
	"ai-governance-saas/backend/internal/infrastructure/postgres/repositories"
	"ai-governance-saas/backend/internal/interfaces/worker"
	"ai-governance-saas/backend/internal/platform/config"
	"ai-governance-saas/backend/internal/platform/runtimestats"
)

// withInternalStats serves GET /internal/stats beside the service's own
// routes, for the platform monitor. Without a token the handler is unchanged.
func withInternalStats(h http.Handler, token string, pool *pgxpool.Pool) http.Handler {
	if token == "" {
		return h
	}
	mux := http.NewServeMux()
	mux.Handle("GET /internal/stats", runtimestats.Handler(token, pool))
	mux.Handle("/", h)
	return mux
}

// newPlatformMonitor builds the worker's health monitor from config: who to
// poll, how to reach Redis, and who to email.
func newPlatformMonitor(cfg *config.Config, store *postgres.Store, log *slog.Logger) (*worker.Monitor, error) {
	targets, err := worker.ParseMonitorTargets(cfg.Monitor.Targets)
	if err != nil {
		return nil, err
	}

	deps := worker.MonitorDeps{
		Repo:             repositories.NewPlatformMonitorRepository(store),
		Outbox:           repositories.NewPlatformAdminRepository(store).Outbox,
		Pool:             store.Pool(),
		Targets:          targets,
		Token:            cfg.Monitor.Token,
		AdminEmails:      cfg.Auth.PlatformAdminEmails,
		ConsoleURL:       cfg.Monitor.ConsoleURL,
		MemoryAlertBytes: int64(cfg.Monitor.MemoryAlertMB) << 20,
		Interval:         cfg.Monitor.Interval,
		Log:              log,
	}

	if cfg.Redis.Enabled {
		client := redis.NewClient(&redis.Options{Addr: cfg.Redis.Addr, Password: cfg.Redis.Password, DB: cfg.Redis.DB})
		deps.Redis = func(ctx context.Context) (map[string]any, error) {
			return redisStats(ctx, client)
		}
	}

	if cfg.SMTP.Host != "" {
		deps.Mailer = notify.NewSMTPMailer(cfg.SMTP.Host, cfg.SMTP.Port, cfg.SMTP.Username, cfg.SMTP.Password, cfg.SMTP.From)
	} else {
		log.Warn("monitor: SMTP_HOST is not set; platform alerts stay in the admin console and are not emailed")
	}
	if cfg.Monitor.Token == "" {
		log.Warn("monitor: MONITOR_TOKEN is not set; services are checked for health but report no CPU or memory")
	}
	return worker.NewMonitor(deps), nil
}

// redisStats reads the few INFO fields the admin console shows.
func redisStats(ctx context.Context, client *redis.Client) (map[string]any, error) {
	info, err := client.Info(ctx, "memory", "clients", "server").Result()
	if err != nil {
		return nil, fmt.Errorf("redis info: %w", err)
	}
	stats := map[string]any{}
	for _, line := range strings.Split(info, "\n") {
		key, value, ok := strings.Cut(strings.TrimSpace(line), ":")
		if !ok {
			continue
		}
		switch key {
		case "used_memory", "maxmemory", "connected_clients", "uptime_in_seconds":
			if n, err := strconv.ParseInt(value, 10, 64); err == nil {
				stats[key] = n
			}
		case "redis_version":
			stats[key] = value
		}
	}
	return stats, nil
}
