// Package bootstrap wires configuration, logging and the HTTP server for each
// process, including graceful shutdown on SIGINT/SIGTERM. Each cmd/*/main.go
// is a thin shell over the matching Run* function.
package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"ai-governance-saas/backend/internal/application/activity"
	"ai-governance-saas/backend/internal/application/alert"
	"ai-governance-saas/backend/internal/application/auditlog"
	"ai-governance-saas/backend/internal/application/billing"
	"ai-governance-saas/backend/internal/application/budget"
	"ai-governance-saas/backend/internal/application/conversation"
	"ai-governance-saas/backend/internal/application/guardrails"
	"ai-governance-saas/backend/internal/application/identity"
	"ai-governance-saas/backend/internal/application/policy"
	"ai-governance-saas/backend/internal/application/pricing"
	"ai-governance-saas/backend/internal/application/provider"
	"ai-governance-saas/backend/internal/application/securityevent"
	"ai-governance-saas/backend/internal/application/usage"
	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/infrastructure/auth"
	"ai-governance-saas/backend/internal/infrastructure/invalidation"
	"ai-governance-saas/backend/internal/infrastructure/payment/razorpay"
	"ai-governance-saas/backend/internal/infrastructure/payment/stripe"
	"ai-governance-saas/backend/internal/infrastructure/postgres"
	"ai-governance-saas/backend/internal/infrastructure/postgres/repositories"
	"ai-governance-saas/backend/internal/infrastructure/provider/anthropic"
	"ai-governance-saas/backend/internal/infrastructure/provider/gemini"
	"ai-governance-saas/backend/internal/infrastructure/provider/huggingface"
	"ai-governance-saas/backend/internal/infrastructure/provider/openai"
	"ai-governance-saas/backend/internal/infrastructure/secrets"
	"ai-governance-saas/backend/internal/interfaces/controlapi"
	"ai-governance-saas/backend/internal/interfaces/gateway"
	"ai-governance-saas/backend/internal/interfaces/gatewayv2"
	"ai-governance-saas/backend/internal/interfaces/worker"
	"ai-governance-saas/backend/internal/platform/config"
	"ai-governance-saas/backend/internal/platform/credentialcache"
	"ai-governance-saas/backend/internal/platform/health"
	"ai-governance-saas/backend/internal/platform/httpx"
	"ai-governance-saas/backend/internal/platform/logging"
	"ai-governance-saas/backend/internal/ports/payment"
	providerclient "ai-governance-saas/backend/internal/ports/provider"
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
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("gateway-v2: %w", err)
	}
	log := logging.New(cfg.Observability)
	log.Info("starting process", "process", "gateway-v2", "addr", cfg.Server.Addr)

	var jobs sync.WaitGroup
	reg := health.NewRegistry()

	var deps gatewayv2.Deps
	if cfg.Database.Enabled {
		store, err := postgres.NewStore(ctx, cfg.Database.URL,
			postgres.WithMaxConns(int32(cfg.Database.MaxOpenConns)),
			postgres.WithMinConns(int32(cfg.Database.MaxIdleConns)),
			postgres.WithMaxConnLifetime(cfg.Database.ConnMaxLifetime),
		)
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

		var bus usageBus
		if cfg.Eventbus.Enabled {
			bus, err = newUsageBus(ctx, &cfg.Eventbus, log)
			if err != nil {
				return fmt.Errorf("gateway-v2: %w", err)
			}
			defer bus.Close()
			if bus.ping != nil {
				reg.Register("eventbus", bus.Ping)
			}
		} else {
			log.Warn("eventbus disabled: usage events will not be recorded")
		}

		var redisClient *redis.Client
		if cfg.Redis.Enabled {
			redisClient = redis.NewClient(&redis.Options{
				Addr:     cfg.Redis.Addr,
				Password: cfg.Redis.Password,
				DB:       cfg.Redis.DB,
			})
			reg.Register("redis", func(ctx context.Context) error {
				return redisClient.Ping(ctx).Err()
			})
		}

		var rateLimiter gatewayv2.RateLimiter
		var budgetGuard gatewayv2.BudgetGuard
		if redisClient != nil {
			rl := gatewayv2.NewRedisRateLimiter(redisClient, log)
			bg := gatewayv2.NewBudgetGuardWithRedis(redisClient, log)
			if cfg.Redis.IsFailClosed() {
				rl.WithMode(gatewayv2.ModeFailClosed)
				bg.WithMode(gatewayv2.BudgetModeFailClosed)
				log.Info("redis outage policy configured as fail-closed", "rate_limiter", "fail_closed", "budget_guard", "fail_closed")
			} else {
				rl.WithMode(gatewayv2.ModeDegradedLocal)
				bg.WithMode(gatewayv2.BudgetModeDegradedLocal)
				log.Info("redis outage policy configured as degraded-local (fail-open)", "rate_limiter", "degraded_local", "budget_guard", "degraded_local")
			}
			rateLimiter = rl
			budgetGuard = bg
		} else {
			rateLimiter = gatewayv2.NewRateLimiter()
			budgetGuard = gatewayv2.NewBudgetGuard()
		}

		activityTracker := gatewayv2.NewActivityTracker(redisClient, log)

		// Build gateway v2 with all components
		gatewayV2, err := gatewayv2.NewGateway(gatewayv2.GatewayDeps{
			Config: gatewayv2.DefaultGatewayConfig(),
			Logger: log,

			Hasher:              auth.NewSHA256TokenHasher(),
			GatewayCredRepo:     repositories.NewGatewayCredentialRepository(store),
			ProviderAccounts:    repositories.NewProviderAccountRepository(store),
			ProviderCredentials: repositories.NewProviderCredentialRepository(store),
			Secrets:             secretStore,
			ModelRepo:           repositories.NewModelRepository(store),
			ProjectRepo:         repositories.NewProjectRepository(store),
			PolicyRepo:          repositories.NewPolicyRepository(store),
			RuleRepo:            repositories.NewPolicyRuleRepository(store),
			Guardrails:          repositories.NewGuardrailsRepository(store),
			EventBus:            bus.EventBus,
			Activity:            activityTracker,
			RateLimiter:         rateLimiter,
			BudgetGuard:         budgetGuard,
		})
		if err != nil {
			return fmt.Errorf("gateway-v2: create gateway: %w", err)
		}

		if redisClient != nil {
			jobs.Add(1)
			go func() {
				defer jobs.Done()
				pubsub := redisClient.Subscribe(ctx, invalidation.PolicyInvalidationChannel, invalidation.CredentialInvalidationChannel)
				defer pubsub.Close()
				ch := pubsub.Channel()
				for {
					select {
					case <-ctx.Done():
						return
					case msg, ok := <-ch:
						if !ok {
							return
						}
						switch msg.Channel {
						case invalidation.PolicyInvalidationChannel:
							var payload invalidation.PolicyInvalidationPayload
							if err := json.Unmarshal([]byte(msg.Payload), &payload); err == nil {
								if orgID, err := uuid.Parse(payload.OrganizationID); err == nil {
									gatewayV2.InvalidatePolicy(orgID)
									gatewayV2.InvalidateUpstream(orgID)
								}
							}
						case invalidation.CredentialInvalidationChannel:
							var payload invalidation.CredentialInvalidationPayload
							if err := json.Unmarshal([]byte(msg.Payload), &payload); err == nil {
								gatewayV2.InvalidateCredentialByID(ctx, payload.CredentialID)
							}
						}
					}
				}
			}()
		}

		if bus.inProcess {
			usageSvc := usage.NewService(usage.ServiceDeps{
				LlmRequests: repositories.NewLlmRequestRepository(store),
				UsageEvents: repositories.NewUsageEventRepository(store),
				Log:         log,
			})
			secSvc := securityevent.NewService(securityevent.ServiceDeps{Events: repositories.NewSecurityEventRepository(store)})
			jobs.Add(1)
			go func() {
				defer jobs.Done()
				worker.RunUsageConsumer(ctx, bus.EventBus, usageSvc, log)
			}()
			jobs.Add(1)
			go func() {
				defer jobs.Done()
				worker.RunSecurityConsumer(ctx, bus.EventBus, secSvc, repositories.NewAuditLogRepository(store), log)
			}()
		}

		if err := gatewayV2.Start(); err != nil {
			return fmt.Errorf("gateway-v2: start: %w", err)
		}

		deps.Gateway = gatewayV2
	}

	// The gateway is the streaming data plane: no per-request deadline.
	h := gatewayv2.NewHandler(log, newGatewayMiddleware(log, cfg), reg, deps)
	err = serveHTTP(ctx, cfg, h, "gateway-v2")
	cancel()
	jobs.Wait()
	return err
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
		store, err := postgres.NewStore(ctx, cfg.Database.URL,
			postgres.WithMaxConns(int32(cfg.Database.MaxOpenConns)),
			postgres.WithMinConns(int32(cfg.Database.MaxIdleConns)),
			postgres.WithMaxConnLifetime(cfg.Database.ConnMaxLifetime),
		)
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

		var bus usageBus
		if cfg.Eventbus.Enabled {
			bus, err = newUsageBus(ctx, &cfg.Eventbus, log)
			if err != nil {
				return fmt.Errorf("control-api: eventbus: %w", err)
			}
			defer bus.Close()
			if bus.ping != nil {
				reg.Register("eventbus", bus.Ping)
			}
		}

		var redisClient *redis.Client
		if cfg.Redis.Enabled {
			redisClient = redis.NewClient(&redis.Options{
				Addr:     cfg.Redis.Addr,
				Password: cfg.Redis.Password,
				DB:       cfg.Redis.DB,
			})
			reg.Register("redis", func(ctx context.Context) error {
				return redisClient.Ping(ctx).Err()
			})
		}

		redisInvalidator := invalidation.NewRedisInvalidator(redisClient, log)

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

			GatewayCredentials:     repositories.NewGatewayCredentialRepository(store),
			GatewayCredentialCache: credentialcache.NewCredentialCache(redisClient, 5*time.Minute),
			CredentialInvalidator:  redisInvalidator,

			Auth:            repositories.NewAuthRepository(store),
			RefreshTokens:   repositories.NewRefreshTokenRepository(store),
			Subs:            repositories.NewSubscriptionRepository(store),
			Passwords:       auth.NewPBKDF2PasswordHasher(0),
			RefreshTokenTTL: cfg.Auth.RefreshTokenTTL,

			Issuer:   issuer,
			Verifier: verifier,
			Hasher:   hasher,
			Log:      log,
		})

		secretStore, err := secrets.New(secrets.Options{
			Provider:  cfg.Secrets.Provider,
			File:      cfg.Secrets.File,
			Key:       cfg.Secrets.Key,
			URL:       cfg.Secrets.URL,
			Token:     cfg.Secrets.Token,
			MountPath: cfg.Secrets.MountPath,
		})
		if err != nil {
			return fmt.Errorf("control-api: secrets store: %w", err)
		}

		securityEventSvc := securityevent.NewService(securityevent.ServiceDeps{
			Events: repositories.NewSecurityEventRepository(store),
			Log:    log,
		})

		providerSvc := provider.NewService(provider.ServiceDeps{
			Accounts:       repositories.NewProviderAccountRepository(store),
			Credentials:    repositories.NewProviderCredentialRepository(store),
			Models:         repositories.NewModelRepository(store),
			Health:         repositories.NewProviderHealthRepository(store),
			Audits:         repositories.NewAuditLogRepository(store),
			SecurityEvents: securityEventSvc,
			Secrets:        secretStore,
			Providers:      providerAdapters(),
			Invalidator:    redisInvalidator,
			Log:            log,
		})

		activitySvc := activity.NewService(redisClient, log)

		usageSvc := usage.NewService(usage.ServiceDeps{
			Rollups:     repositories.NewUsageRollupRepository(store),
			LlmRequests: repositories.NewLlmRequestRepository(store),
			UsageEvents: repositories.NewUsageEventRepository(store),
			Log:         log,
		})

		conversationSvc := conversation.NewService(conversation.ServiceDeps{
			Organizations: repositories.NewOrganizationRepository(store),
			Conversations: repositories.NewConversationRepository(store),
			Messages:      repositories.NewConversationMessageRepository(store),
			Log:           log,
		})

		auditSvc := auditlog.NewService(auditlog.ServiceDeps{
			Audits: repositories.NewAuditLogRepository(store),
			Log:    log,
		})

		alertRepo := repositories.NewAlertRepository(store)
		alertEval := alert.NewEvaluator(alert.EvaluatorDeps{
			Log:            log,
			Alerts:         alertRepo,
			Notifier:       alert.NewInAppNotifier(log, alertRepo),
			UsageRollups:   repositories.NewUsageRollupRepository(store),
			ProviderHealth: repositories.NewProviderHealthRepository(store),
			SecurityEvents: repositories.NewSecurityEventRepository(store),
			LlmRequests:    repositories.NewLlmRequestRepository(store),
		})

		alertSvc := alert.NewService(alert.ServiceDeps{
			Log:       log,
			Alerts:    alertRepo,
			Audits:    repositories.NewAuditLogRepository(store),
			Evaluator: alertEval,
		})

		billingMode := cfg.Billing.EffectiveMode()
		stripeGw := stripe.New(cfg.Billing.StripeSecretKey, cfg.Billing.StripeWebhookSecret)
		stripeGw.SetMode(billingMode)
		razorpayGw := razorpay.New(cfg.Billing.RazorpayKeyID, cfg.Billing.RazorpayKeySecret, cfg.Billing.RazorpayWebhookSecret)
		razorpayGw.SetMode(billingMode)

		billingSvc := billing.NewService(billing.ServiceDeps{
			Mode:          billingMode,
			Plans:         repositories.NewPlanRepository(store),
			Subscriptions: repositories.NewSubscriptionRepository(store),
			Invoices:      repositories.NewInvoiceRepository(store),
			Checkouts:     repositories.NewCheckoutRepository(store),
			Rollups:       repositories.NewUsageRollupRepository(store),
			BillingEvents: repositories.NewBillingEventRepository(store),
			Audits:        repositories.NewAuditLogRepository(store),
			PaymentGateways: map[string]payment.PaymentGateway{
				"stripe":   stripeGw,
				"razorpay": razorpayGw,
			},
		})

		guardrailsSvc := guardrails.NewService(
			repositories.NewGuardrailsRepository(store),
			repositories.NewAuditLogRepository(store),
		)
		guardrailsEval := guardrails.NewEvaluator()

		policySvc := policy.NewService(
			repositories.NewPolicyRepository(store),
			repositories.NewPolicyRuleRepository(store),
			repositories.NewAuditLogRepository(store),
			redisInvalidator,
			log,
		)

		budgetSvc := budget.NewService(budget.ServiceDeps{
			Budgets:     repositories.NewBudgetRepository(store),
			Policies:    repositories.NewPolicyRepository(store),
			Rules:       repositories.NewPolicyRuleRepository(store),
			Audits:      repositories.NewAuditLogRepository(store),
			Invalidator: redisInvalidator,
		})

		pricingSvc := pricing.NewService(pricing.ServiceDeps{
			PricingModels:   repositories.NewPricingModelRepository(store),
			PricingVersions: repositories.NewPricingVersionRepository(store),
			Audits:          repositories.NewAuditLogRepository(store),
		})

		deps = controlapi.Deps{
			Log:                 log,
			Identity:            svc,
			Providers:           providerSvc,
			Usage:               usageSvc,
			Activity:            activitySvc,
			Conversations:       conversationSvc,
			AuditLogs:           auditSvc,
			Alerts:              alertSvc,
			SecurityEvents:      securityEventSvc,
			Billing:             billingSvc,
			Budget:              budgetSvc,
			Pricing:             pricingSvc,
			Policy:              policySvc,
			Guardrails:          guardrailsSvc,
			GuardrailsEval:      guardrailsEval,
			DevLoginEnabled:     cfg.Auth.DevLoginEnabled,
			SessionCookieName:   cfg.Auth.SessionCookieName,
			SessionCookieSecure: cfg.Auth.SessionCookieSecure,
			CSRFEnabled:         cfg.Auth.CSRFEnabled,
		}
	}

	h := controlapi.NewHandler(log, newMiddleware(log, cfg), reg, deps)
	return serveHTTP(ctx, cfg, h, "control-api")
}

// providerAdapters returns the upstream client for every provider kind the
// control plane can validate credentials and health-check against. Keys must be
// domain.ProviderKind* values because Service.CreateCredential dispatches on
// that same key. Kinds absent here are rejected at credential-create time
// rather than validated against the wrong upstream.
func providerAdapters() map[string]providerclient.Provider {
	return map[string]providerclient.Provider{
		domain.ProviderKindAnthropic:   anthropic.NewProviderAdapter(),
		domain.ProviderKindOpenAI:      openai.NewProviderAdapter(),
		domain.ProviderKindGemini:      gemini.NewProviderAdapter(),
		domain.ProviderKindHuggingFace: huggingface.NewProviderAdapter(),
		domain.ProviderKindCustom:      openai.NewCustomProviderAdapter("https://api.mistral.ai", ""),
	}
}

// RunWorker starts the async worker and blocks
// until ctx is cancelled (graceful shutdown) or the server fails.
func RunWorker(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("worker: %w", err)
	}
	log := logging.New(cfg.Observability)
	log.Info("starting process", "process", "worker", "addr", cfg.Server.Addr)

	var jobs sync.WaitGroup
	reg := health.NewRegistry()
	if cfg.Database.Enabled {
		store, err := postgres.NewStore(ctx, cfg.Database.URL,
			postgres.WithMaxConns(int32(cfg.Database.MaxOpenConns)),
			postgres.WithMinConns(int32(cfg.Database.MaxIdleConns)),
			postgres.WithMaxConnLifetime(cfg.Database.ConnMaxLifetime),
		)
		if err != nil {
			return fmt.Errorf("worker: connect database: %w", err)
		}
		defer store.Close()

		reg.Register("postgres", store.Pool().Ping)

		rollups := usage.NewService(usage.ServiceDeps{
			Orgs:        repositories.NewOrganizationRepository(store),
			Rollups:     repositories.NewUsageRollupRepository(store),
			LlmRequests: repositories.NewLlmRequestRepository(store),
			UsageEvents: repositories.NewUsageEventRepository(store),
			SecurityEvents: securityevent.NewService(securityevent.ServiceDeps{
				Events: repositories.NewSecurityEventRepository(store),
				Log:    log,
			}),
			Log: log,
		})
		jobs.Add(1)
		go func() {
			defer jobs.Done()
			worker.RunRollup(ctx, rollups, log)
		}()

		retentionJob := worker.NewRetentionJob(worker.RetentionDeps{
			Orgs:          repositories.NewOrganizationRepository(store),
			Conversations: repositories.NewConversationRepository(store),
			UsageEvents:   repositories.NewUsageEventRepository(store),
			LlmRequests:   repositories.NewLlmRequestRepository(store),
			AuditLogs:     repositories.NewAuditLogRepository(store),
			BillingEvents: repositories.NewBillingEventRepository(store),
			Idempotency:   repositories.NewIdempotencyRepository(store),
			RefreshTokens: repositories.NewRefreshTokenRepository(store),
			Invitations:   repositories.NewInvitationRepository(store),
			Log:           log,
		})
		jobs.Add(1)
		go func() {
			defer jobs.Done()
			worker.RunRetention(ctx, retentionJob, 5*time.Minute)
		}()

		workerAlertRepo := repositories.NewAlertRepository(store)
		_ = alert.NewService(alert.ServiceDeps{
			Log:    log,
			Alerts: workerAlertRepo,
			Audits: repositories.NewAuditLogRepository(store),
		})
		notifier := alert.NewInAppNotifier(log, workerAlertRepo)
		evaluator := alert.NewEvaluator(alert.EvaluatorDeps{
			Log:            log,
			Alerts:         workerAlertRepo,
			Notifier:       notifier,
			UsageRollups:   repositories.NewUsageRollupRepository(store),
			ProviderHealth: repositories.NewProviderHealthRepository(store),
			SecurityEvents: repositories.NewSecurityEventRepository(store),
			LlmRequests:    repositories.NewLlmRequestRepository(store),
		})

		jobs.Add(1)
		go func() {
			defer jobs.Done()
			// Initial pass on start
			if orgIDs, err := repositories.NewOrganizationRepository(store).ListAllIDs(ctx); err == nil {
				evaluator.EvaluateAll(ctx, orgIDs)
			}
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					log.Info("alert evaluator loop stopped")
					return
				case <-ticker.C:
					orgIDs, err := repositories.NewOrganizationRepository(store).ListAllIDs(ctx)
					if err != nil {
						log.Error("failed to list organizations for alert evaluation", "error", err)
						continue
					}
					evaluator.EvaluateAll(ctx, orgIDs)
				}
			}
		}()

		if cfg.Eventbus.Enabled {
			bus, err := newUsageBus(ctx, &cfg.Eventbus, log)
			if err != nil {
				return fmt.Errorf("worker: %w", err)
			}
			defer bus.Close()
			if bus.ping != nil {
				reg.Register("eventbus", bus.Ping)
			}
			if !bus.inProcess {
				secSvc := securityevent.NewService(securityevent.ServiceDeps{Events: repositories.NewSecurityEventRepository(store)})
				jobs.Add(1)
				go func() {
					defer jobs.Done()
					worker.RunUsageConsumer(ctx, bus.EventBus, rollups, log)
				}()
				jobs.Add(1)
				go func() {
					defer jobs.Done()
					worker.RunSecurityConsumer(ctx, bus.EventBus, secSvc, repositories.NewAuditLogRepository(store), log)
				}()
			}
		}
	}

	h := worker.NewHandler(log, newMiddleware(log, cfg), reg)
	err = serveHTTP(ctx, cfg, h, "worker")
	cancel()
	jobs.Wait()
	return err
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
