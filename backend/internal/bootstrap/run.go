// Package bootstrap wires configuration, logging and the HTTP server for each
// process, including graceful shutdown on SIGINT/SIGTERM. Each cmd/*/main.go
// is a thin shell over the matching Run* function.
package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"ai-governance-saas/backend/internal/application/identity"
	"ai-governance-saas/backend/internal/infrastructure/auth"
	"ai-governance-saas/backend/internal/infrastructure/postgres"
	"ai-governance-saas/backend/internal/infrastructure/postgres/repositories"
	"ai-governance-saas/backend/internal/infrastructure/secrets"
	"ai-governance-saas/backend/internal/interfaces/controlapi"
	"ai-governance-saas/backend/internal/interfaces/gateway"
	"ai-governance-saas/backend/internal/interfaces/gatewayv2"
	"ai-governance-saas/backend/internal/interfaces/worker"
	"ai-governance-saas/backend/internal/platform/config"
	"ai-governance-saas/backend/internal/platform/health"
	"ai-governance-saas/backend/internal/platform/httpx"
	"ai-governance-saas/backend/internal/platform/logging"
)

// RunGateway starts the data-plane HTTP server and blocks until ctx is
// cancelled (graceful shutdown) or the server fails.
func RunGateway(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("gateway: %w", err)
	}
	log := logging.New(cfg.Observability)
	log.Info("starting process", "process", "gateway", "addr", cfg.Server.Addr)

	reg := health.NewRegistry()

	// The proxy is wired from DB-backed state when the database is enabled;
	// otherwise the gateway serves only the health probes.
	var deps gateway.Deps
	if cfg.Database.Enabled {
		store, err := postgres.NewStore(ctx, cfg.Database.URL)
		if err != nil {
			return fmt.Errorf("gateway: connect database: %w", err)
		}
		defer store.Close()
		reg.Register("postgres", store.Pool().Ping)

		secretStore, err := secrets.New(secrets.Options{
			Provider:  cfg.Secrets.Provider,
			File:      cfg.Secrets.File,
			Key:       cfg.Secrets.Key,
			URL:       cfg.Secrets.URL,
			Token:     cfg.Secrets.Token,
			MountPath: cfg.Secrets.MountPath,
		})
		if err != nil {
			return fmt.Errorf("gateway: secrets store: %w", err)
		}

		proxy, err := gateway.NewProxy(gateway.ProxyDeps{
			Log:                 log,
			Hasher:              auth.NewSHA256TokenHasher(),
			Credentials:         repositories.NewAPICredentialRepository(store),
			Accounts:            repositories.NewProviderAccountRepository(store),
			ProviderCredentials: repositories.NewProviderCredentialRepository(store),
			Secrets:             secretStore,
			UpstreamBaseURL:     cfg.Provider.BaseURL,
		})
		if err != nil {
			return fmt.Errorf("gateway: provider proxy: %w", err)
		}
		deps.Proxy = proxy
	}

	// The gateway is the streaming data plane: no per-request deadline.
	h := gateway.NewHandler(log, newGatewayMiddleware(log, cfg), reg, deps)
	return serveHTTP(ctx, cfg, h, "gateway")
}

// RunGatewayV2 starts the latency-sensitive data-plane HTTP server (v2) and blocks
// until ctx is cancelled (graceful shutdown) or the server fails.
func RunGatewayV2(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("gateway-v2: %w", err)
	}
	log := logging.New(cfg.Observability)
	log.Info("starting process", "process", "gateway-v2", "addr", cfg.Server.Addr)

	reg := health.NewRegistry()

	var deps gatewayv2.Deps
	if cfg.Database.Enabled {
		store, err := postgres.NewStore(ctx, cfg.Database.URL)
		if err != nil {
			return fmt.Errorf("gateway-v2: connect database: %w", err)
		}
		defer store.Close()
		reg.Register("postgres", store.Pool().Ping)

		secretStore, err := secrets.New(secrets.Options{
			Provider:  cfg.Secrets.Provider,
			File:      cfg.Secrets.File,
			Key:       cfg.Secrets.Key,
			URL:       cfg.Secrets.URL,
			Token:     cfg.Secrets.Token,
			MountPath: cfg.Secrets.MountPath,
		})
		if err != nil {
			return fmt.Errorf("gateway-v2: secrets store: %w", err)
		}

		// Build gateway v2 with all components
		gatewayV2, err := gatewayv2.NewGateway(gatewayv2.GatewayDeps{
			Config: gatewayv2.DefaultGatewayConfig(),
			Logger: log,

			Hasher:            auth.NewSHA256TokenHasher(),
			GatewayCredRepo:   repositories.NewGatewayCredentialRepository(store),
			ProviderAccounts:  repositories.NewProviderAccountRepository(store),
			ProviderCredentials: repositories.NewProviderCredentialRepository(store),
			Secrets:           secretStore,
			PolicyRepo:        repositories.NewPolicyRepository(store),
			RuleRepo:          repositories.NewPolicyRuleRepository(store),
		})
		if err != nil {
			return fmt.Errorf("gateway-v2: create gateway: %w", err)
		}

		if err := gatewayV2.Start(); err != nil {
			return fmt.Errorf("gateway-v2: start: %w", err)
		}

		// Register shutdown hook
		// The serveHTTP function will handle graceful shutdown
		_ = gatewayV2 // suppress unused warning for now

		deps.Gateway = gatewayV2
	}

	// The gateway is the streaming data plane: no per-request deadline.
	h := gatewayv2.NewHandler(log, newGatewayMiddleware(log, cfg), reg, deps)
	return serveHTTP(ctx, cfg, h, "gateway-v2")
}

// RunControlAPI starts the control-plane API server and blocks until ctx is
// cancelled (graceful shutdown) or the server fails. When the database is
// enabled it wires Postgres, the identity service, and the v1 HTTP surface;
// otherwise only the health probes are served.
func RunControlAPI(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("control-api: %w", err)
	}
	log := logging.New(cfg.Observability)
	log.Info("starting process", "process", "control-api", "addr", cfg.Server.Addr)

	reg := health.NewRegistry()

	var deps controlapi.Deps
	if cfg.Database.Enabled {
		if cfg.Auth.HMACSigningKey == "" {
			return fmt.Errorf("control-api: DATABASE_ENABLED=true requires AUTH_HMAC_SIGNING_KEY")
		}
		store, err := postgres.NewStore(ctx, cfg.Database.URL)
		if err != nil {
			return fmt.Errorf("control-api: connect database: %w", err)
		}
		defer store.Close()

		reg.Register("postgres", store.Pool().Ping)

		prevKeys := make([][]byte, 0, len(cfg.Auth.HMACPreviousKeys))
		for _, k := range cfg.Auth.HMACPreviousKeys {
			prevKeys = append(prevKeys, []byte(k))
		}
		issuer, verifier, err := auth.NewRotatingHMACTokenManager(cfg.Auth.JWTIssuer, cfg.Auth.JWTTTL, []byte(cfg.Auth.HMACSigningKey), prevKeys...)
		if err != nil {
			return fmt.Errorf("control-api: token manager: %w", err)
		}
		hasher := auth.NewSHA256TokenHasher()

		svc := identity.NewService(identity.ServiceDeps{
			Orgs:        repositories.NewOrganizationRepository(store),
			Users:       repositories.NewUserRepository(store),
			Members:     repositories.NewOrgMemberRepository(store),
			Invitations: repositories.NewInvitationRepository(store),
			Projects:    repositories.NewProjectRepository(store),
			Teams:       repositories.NewTeamRepository(store),
			TeamMembers: repositories.NewTeamMemberRepository(store),
			ProjMembers: repositories.NewProjectMemberRepository(store),
			Accounts:    repositories.NewServiceAccountRepository(store),
			Credentials: repositories.NewAPICredentialRepository(store),
			Audits:      repositories.NewAuditLogRepository(store),

			Issuer:   issuer,
			Verifier: verifier,
			Hasher:   hasher,
			Log:      log,
		})

		deps = controlapi.Deps{
			Log:                 log,
			Identity:            svc,
			DevLoginEnabled:     cfg.Auth.DevLoginEnabled,
			SessionCookieName:   cfg.Auth.SessionCookieName,
			SessionCookieSecure: cfg.Auth.SessionCookieSecure,
			CSRFEnabled:         cfg.Auth.CSRFEnabled,
		}
	}

	h := controlapi.NewHandler(log, newMiddleware(log, cfg), reg, deps)
	return serveHTTP(ctx, cfg, h, "control-api")
}

// RunWorker starts the async worker (health listener only for now) and blocks
// until ctx is cancelled (graceful shutdown) or the server fails.
func RunWorker(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("worker: %w", err)
	}
	log := logging.New(cfg.Observability)
	log.Info("starting process", "process", "worker", "addr", cfg.Server.Addr)

	reg := health.NewRegistry()
	h := worker.NewHandler(log, newMiddleware(log, cfg), reg)
	return serveHTTP(ctx, cfg, h, "worker")
}

func newMiddleware(log *slog.Logger, cfg *config.Config) *httpx.Middleware {
	opts := httpx.Options{
		RequestIDHeader:       cfg.Security.RequestIDHeader,
		DefaultRequestTimeout: cfg.Server.RequestTimeout,
		CORSAllowedOrigins:    cfg.Security.CORSAllowedOrigins,
	}
	return httpx.NewMiddleware(log, opts)
}

// newGatewayMiddleware builds the gateway chain without the per-request
// deadline applied by newMiddleware (Server.RequestTimeout); provider
// responses stream for minutes and must not be cut off.
func newGatewayMiddleware(log *slog.Logger, cfg *config.Config) *httpx.Middleware {
	return httpx.NewMiddleware(log, httpx.Options{
		RequestIDHeader:    cfg.Security.RequestIDHeader,
		CORSAllowedOrigins: cfg.Security.CORSAllowedOrigins,
	})
}

// serveHTTP runs srv until ctx is cancelled, then drains connections for up to
// ShutdownTimeout before returning.
func serveHTTP(ctx context.Context, cfg *config.Config, h http.Handler, name string) error {
	srv := &http.Server{
		Addr:              cfg.Server.Addr,
		Handler:           h,
		ReadHeaderTimeout: cfg.Server.ReadHeaderTimeout,
	}

	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()
	slog.Info("http server listening", "service", name, "addr", cfg.Server.Addr)

	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("%s server: %w", name, err)
		}
		return nil
	case <-ctx.Done():
		slog.Info("shutting down", "service", name, "timeout", cfg.Server.ShutdownTimeout)
		sctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
		defer cancel()
		if err := srv.Shutdown(sctx); err != nil {
			return fmt.Errorf("%s graceful shutdown: %w", name, err)
		}
		if err := <-errCh; err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("%s server: %w", name, err)
		}
		return nil
	}
}
