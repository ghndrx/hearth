package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"hearth/internal/ai"
	"hearth/internal/api"
	"hearth/internal/api/handlers"
	"hearth/internal/api/middleware"
	"hearth/internal/auth"
	"hearth/internal/cache"
	"hearth/internal/config"
	"hearth/internal/database/postgres"
	"hearth/internal/events"
	"hearth/internal/metrics"
	"hearth/internal/pubsub"
	"hearth/internal/ratelimit"
	"hearth/internal/services"
	"hearth/internal/websocket"
)

var (
	Version = "1.0.0-dev"
	Commit  = "unknown"
)

func main() {
	// Version command
	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Printf("Hearth %s (%s)\n", Version, Commit)
		return
	}

	log.Printf("🔥 Hearth %s (%s)", Version, Commit)

	// Initialize Prometheus metrics early
	wsMetrics := metrics.NewWebSocketMetrics()
	httpMetrics := metrics.NewHTTPMetrics()
	log.Printf("📊 Prometheus metrics initialized (instance: %s)", metrics.GetInstanceLabel())
	_ = wsMetrics // Used implicitly via metrics.GetMetrics()

	// Load configuration
	cfg := config.Load()

	// Connect to database
	db, err := postgres.NewDBFromURL(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Start DB pool metrics collection (every 15s)
	metrics.StartStatsCollector(context.Background(), db.DB, "primary", 15*time.Second)

	// Run migrations
	if err := postgres.Migrate(context.Background(), db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Initialize repositories
	repos := postgres.NewRepositories(db)

	// Initialize event bus
	eventBus := events.NewBus()
	serviceBus := events.NewServiceBusAdapter(eventBus)

	// Initialize bcrypt worker pool (bounded concurrency for password operations)
	// This prevents CPU saturation under load - critical for p99 < 500ms target
	bcryptPoolConfig := auth.PoolConfig{
		Workers:        cfg.BcryptPoolWorkers, // 0 = NumCPU (auto)
		QueueSize:      cfg.BcryptPoolQueue,   // 0 = Workers * 10 (auto)
		DefaultTimeout: cfg.BcryptPoolTimeout,
		Cost:           12, // Production bcrypt cost
	}
	bcryptPool := auth.NewBcryptPool(bcryptPoolConfig)
	auth.SetGlobalPool(bcryptPool)
	defer bcryptPool.Close()
	log.Printf("Bcrypt worker pool initialized: %d workers, queue size %d, timeout %v",
		bcryptPool.Stats().Workers, bcryptPool.Stats().QueueSize, cfg.BcryptPoolTimeout)

	// Initialize auth services
	jwtService := auth.NewJWTService(
		cfg.SecretKey,
		15*time.Minute, // Access token expiry
		7*24*time.Hour, // Refresh token expiry
	)

	// Create context for graceful shutdown (needed for WebSocket hub)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Configure graceful shutdown draining
	drainConfig := &websocket.DrainConfig{
		DrainTimeout: cfg.DrainTimeout,
		GracePeriod:  cfg.DrainGracePeriod,
	}
	log.Printf("⚙️  Drain config: timeout=%v, grace=%v", drainConfig.DrainTimeout, drainConfig.GracePeriod)

	// Initialize WebSocket hub (distributed with Redis, or local fallback)
	var wsHub websocket.HubInterface
	var wsGateway *websocket.Gateway
	var redisCache *cache.RedisCache

	// Try to initialize Redis for distributed messaging
	redisCache, err = cache.NewRedisCache(cfg.RedisURL)
	if err != nil {
		log.Printf("⚠️  Redis not available, using in-memory hub (single-instance mode): %v", err)
		// Fallback to non-distributed hub
		localHub := websocket.NewHubWithDrainConfig(drainConfig)
		wsHub = localHub
		go localHub.Run(ctx)
		wsGateway = websocket.NewGateway(localHub, jwtService, nil)
		_ = websocket.NewEventBridge(localHub, eventBus)
	} else {
		defer redisCache.Close()
		log.Printf("✅ Redis connected: %s", cfg.RedisURL)

		// Generate unique node ID for this instance
		nodeID := os.Getenv("HEARTH_NODE_ID")
		if nodeID == "" {
			hostname, _ := os.Hostname()
			nodeID = fmt.Sprintf("%s-%s", hostname, uuid.New().String()[:8])
		}
		log.Printf("📡 Node ID: %s", nodeID)

		// Initialize Redis Pub/Sub for distributed messaging
		ps, err := pubsub.New(cfg.RedisURL, nodeID)
		if err != nil {
			log.Fatalf("Failed to initialize Redis pub/sub: %v", err)
		}
		defer ps.Close()
		log.Printf("✅ Redis Pub/Sub initialized for distributed messaging")

		// Initialize Distributed WebSocket hub with drain config
		distributedHub := websocket.NewDistributedHubWithDrainConfig(ps, drainConfig)
		wsHub = distributedHub
		go distributedHub.Run(ctx)

		// Initialize WebSocket gateway with distributed hub
		wsGateway = websocket.NewGateway(distributedHub, jwtService, nil)

		// Initialize distributed event bridge (connects domain events to WebSocket via Redis)
		_ = websocket.NewDistributedEventBridge(ctx, distributedHub, eventBus)
	}

	// Initialize services
	quotaService := services.NewQuotaService(cfg.Quotas, nil, nil, nil, nil)
	userService := services.NewUserService(repos.Users, nil, serviceBus)
	authService := services.NewAuthService(repos.Users, jwtService)

	// Initialize OAuth service (if any providers are configured)
	var oauthService *services.OAuthService
	oauthConfig := &services.OAuthProviderConfig{}
	hasOAuthProvider := false

	if cfg.OAuthGitHubClientID != "" && cfg.OAuthGitHubClientSecret != "" {
		oauthConfig.GitHub = &services.OAuthConfig{
			ClientID:     cfg.OAuthGitHubClientID,
			ClientSecret: cfg.OAuthGitHubClientSecret,
			RedirectURI:  cfg.OAuthRedirectBase + "/api/v1/auth/oauth/github/callback",
			Scopes:       []string{"read:user", "user:email"},
		}
		hasOAuthProvider = true
		log.Printf("✅ GitHub OAuth configured")
	}

	if cfg.OAuthGoogleClientID != "" && cfg.OAuthGoogleClientSecret != "" {
		oauthConfig.Google = &services.OAuthConfig{
			ClientID:     cfg.OAuthGoogleClientID,
			ClientSecret: cfg.OAuthGoogleClientSecret,
			RedirectURI:  cfg.OAuthRedirectBase + "/api/v1/auth/oauth/google/callback",
			Scopes:       []string{"openid", "profile", "email"},
		}
		hasOAuthProvider = true
		log.Printf("✅ Google OAuth configured")
	}

	if cfg.OAuthDiscordClientID != "" && cfg.OAuthDiscordClientSecret != "" {
		oauthConfig.Discord = &services.OAuthConfig{
			ClientID:     cfg.OAuthDiscordClientID,
			ClientSecret: cfg.OAuthDiscordClientSecret,
			RedirectURI:  cfg.OAuthRedirectBase + "/api/v1/auth/oauth/discord/callback",
			Scopes:       []string{"identify", "email"},
		}
		hasOAuthProvider = true
		log.Printf("✅ Discord OAuth configured")
	}

	if hasOAuthProvider {
		// Initialize OAuth repository
		oauthRepo := postgres.NewOAuthRepository(db)

		// Use Redis cache if available, otherwise use a simple in-memory implementation
		var cacheService services.CacheService
		if redisCache != nil {
			cacheService = redisCache
		}

		oauthService = services.NewOAuthServiceWithRepo(
			oauthConfig,
			repos.Users,
			oauthRepo,
			cacheService,
			jwtService,
		)
		log.Printf("✅ OAuth service initialized")
	}
	// Initialize permission service first (needed by other services)
	permService := services.NewPermissionService(
		repos.Servers,
		repos.Roles,
		repos.Channels,
		redisCache, // cache
	)
	roleService := services.NewRoleService(
		repos.Roles,
		repos.Servers,
		nil, // cache
		serviceBus,
		permService,
	)
	serverService := services.NewServerService(
		repos.Servers,
		repos.Channels,
		repos.Roles,
		repos.Messages,
		quotaService,
		permService,
		redisCache, // cache
		serviceBus,
	)
	channelService := services.NewChannelService(
		repos.Channels,
		repos.Servers,
		permService,
		redisCache, // cache
		serviceBus,
	)
	messageService := services.NewMessageService(
		repos.Messages,
		repos.Channels,
		repos.Servers,
		repos.Roles,
		repos.Users,
		quotaService,
		nil,        // rate limiter
		nil,        // e2ee service
		redisCache, // cache
		serviceBus,
		permService,
	)
	searchService := services.NewSearchService(
		repos.Search,
		repos.Messages,
		repos.Channels,
		repos.Servers,
		repos.Users,
		redisCache, // cache
	)
	typingService := services.NewTypingService(serviceBus)
	webhookService := services.NewWebhookService(
		repos.Webhooks,
		repos.Channels,
		repos.Servers,
		permService,
		messageService,
		serviceBus,
	)
	pollService := services.NewPollService(repos.Polls)

	// Initialize voice signaling service
	var voiceService *websocket.VoiceSignalingService
	if wsHub != nil {
		voiceService = websocket.NewVoiceSignalingService(wsHub, repos.VoiceStates)
		wsGateway.SetVoiceService(voiceService)
	}

	// Initialize AI service
	var aiService *ai.AIService
	if cfg.AIEncryptionKey != "" {
		aiEncryption, err := ai.NewAESEncryptionService(cfg.AIEncryptionKey)
		if err != nil {
			log.Printf("⚠️  AI encryption service failed to initialize: %v", err)
		} else {
			aiService = ai.NewAIService(repos.AI, aiEncryption)

			// Configure admin defaults from environment
			adminConfig := &ai.AdminAIConfig{
				DefaultProvider:   cfg.AIDefaultProvider,
				AllowUserOverride: cfg.AIAllowUserOverride,
				FeatureDefaults:   make(map[ai.FeatureType]ai.FeatureConfig),
			}
			aiService.SetAdminConfig(adminConfig)
			log.Printf("✅ AI service initialized (user override: %v)", cfg.AIAllowUserOverride)
		}
	} else {
		log.Printf("⚠️  AI service disabled (AI_ENCRYPTION_KEY not set)")
	}

	// Initialize Fiber app with security settings
	app := fiber.New(fiber.Config{
		AppName:               "Hearth",
		DisableStartupMessage: true,
		BodyLimit:             100 * 1024 * 1024, // 100MB
		ReadTimeout:           30 * time.Second,
		WriteTimeout:          30 * time.Second,
		// Security
		EnableTrustedProxyCheck: true,
		ProxyHeader:             "X-Forwarded-For",
		// Custom error handler for handlers.HTTPError
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			var httpErr *handlers.HTTPError
			if errors.As(err, &httpErr) {
				return c.Status(httpErr.Status).JSON(fiber.Map{
					"error":   httpErr.ErrorType,
					"message": httpErr.Message,
					"code":    httpErr.Code,
				})
			}
			// Fallback for other errors
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{
				"error":   "internal_error",
				"message": err.Error(),
			})
		},
	})

	// Security middleware
	app.Use(recover.New())

	// Prometheus HTTP metrics middleware (before other middleware to capture all requests)
	app.Use(httpMetrics.Middleware())

	// Helmet for security headers
	app.Use(helmet.New(helmet.Config{
		XSSProtection:             "1; mode=block",
		ContentTypeNosniff:        "nosniff",
		XFrameOptions:             "SAMEORIGIN",
		ReferrerPolicy:            "strict-origin-when-cross-origin",
		CrossOriginEmbedderPolicy: "require-corp",
		CrossOriginOpenerPolicy:   "same-origin",
		CrossOriginResourcePolicy: "same-origin",
		PermissionPolicy:          "camera=(), microphone=(), geolocation=()",
	}))

	// Rate limiting (can be disabled for testing with RATE_LIMIT_ENABLED=false)
	if cfg.RateLimitEnabled {
		log.Printf("Rate limiting enabled: %d requests per %s", cfg.RateLimitMax, cfg.RateLimitWindow)
		app.Use(limiter.New(limiter.Config{
			Max:               cfg.RateLimitMax,
			Expiration:        cfg.RateLimitWindow,
			LimiterMiddleware: limiter.SlidingWindow{},
			KeyGenerator: func(c *fiber.Ctx) string {
				return c.IP()
			},
			LimitReached: func(c *fiber.Ctx) error {
				return c.Status(429).JSON(fiber.Map{
					"error":   "rate_limited",
					"message": "Too many requests",
				})
			},
		}))
	} else {
		log.Printf("⚠️  Rate limiting DISABLED (not recommended for production)")
	}

	// Logging
	app.Use(logger.New(logger.Config{
		Format:     "[${time}] ${status} - ${latency} ${method} ${path}\n",
		TimeFormat: "2006-01-02 15:04:05",
	}))

	// CORS
	app.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.PublicURL,
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowMethods:     "GET, POST, PUT, PATCH, DELETE, OPTIONS",
		AllowCredentials: true,
		MaxAge:           86400,
	}))

	// Initialize handlers and middleware
	// Thread service
	threadService := services.NewThreadService(
		repos.Threads,
		repos.Channels,
		repos.Servers,
		permService,
		serviceBus,
	)

	h := handlers.NewHandlersWithTyping(authService, userService, serverService, channelService, messageService, roleService, searchService, threadService, typingService, webhookService, wsGateway, voiceService)

	// Wire up Poll handler
	h.SetPollHandler(pollService)

	// Wire up OAuth service if available
	if oauthService != nil {
		h.Auth.SetOAuthService(oauthService)
		log.Printf("✅ OAuth handler wired up")
	}

	// Wire up AI handler if service is available
	if aiService != nil {
		h.SetAIHandler(aiService)

		// Initialize AI Chat service
		chatRepo := ai.NewPostgresChatRepository(db)
		chatService := ai.NewChatService(chatRepo, aiService)
		h.SetAIChatHandler(chatService)
		log.Printf("✅ AI Chat service initialized")
	}

	// Initialize E2EE service and handler
	e2eeService := services.NewE2EEServiceImpl(repos.E2EE)
	h.SetE2EEHandler(e2eeService)
	log.Printf("✅ E2EE service initialized")

	// Initialize component service
	componentRepo := postgres.NewComponentRepository(db)
	componentService := services.NewComponentService(
		componentRepo,
		repos.Messages,
		repos.Channels,
		repos.Servers,
		serviceBus,
	)
	h.SetComponentHandlerWithDeps(componentService, messageService, channelService, permService)
	log.Printf("✅ Component service initialized")

	// Initialize component rate limiter if rate limiting is enabled
	if cfg.RateLimitEnabled {
		memoryCache := middleware.NewSimpleInMemoryCache()
		memoryLimiter := ratelimit.NewLimiter(memoryCache)
		componentRateLimiter := ratelimit.NewComponentRateLimiter(memoryLimiter)
		componentService.SetRateLimiter(componentRateLimiter)
		log.Printf("✅ Component rate limiter wired")
	} else {
		log.Printf("⚠️  Component rate limiting DISABLED")
	}

	// Initialize AutoMod service and handler
	automodService := services.NewAutoModService(repos.AutoMod)
	h.SetAutoModHandler(automodService, serverService)
	log.Printf("✅ AutoMod service initialized")

	// Initialize Template service and handler
	templateService := services.NewTemplateService(repos.Templates, repos.Channels, repos.Roles, repos.Servers)
	h.SetTemplateHandler(templateService, serverService)
	log.Printf("✅ Template service initialized")

	// Initialize Event service and handler
	eventService := services.NewEventService(
		repos.Events,
		repos.Channels,
		repos.Servers,
		permService,
		serviceBus,
	)
	h.SetEventHandler(eventService, serverService, permService)
	log.Printf("✅ Event service initialized")

	// Initialize Sticker service and handler
	stickerService := services.NewStickerService(repos.Stickers, nil)
	h.SetStickerHandler(stickerService, serverService, permService)
	log.Printf("✅ Sticker service initialized")

	// Initialize Soundboard service and handler
	soundboardService := services.NewSoundboardService(nil)
	h.SetSoundboardHandler(soundboardService, serverService, permService)
	log.Printf("✅ Soundboard service initialized")

	// Initialize Discovery service and handler
	discoveryRepo := postgres.NewDiscoveryRepository(db)
	discoveryService := services.NewDiscoveryService(
		discoveryRepo,
		repos.Servers,
		repos.Invites,
		permService,
		serviceBus,
	)
	h.SetDiscoveryHandler(discoveryService, serverService)
	log.Printf("✅ Discovery service initialized")

	// Initialize Enhanced Server Discovery (DiscoverableServer) service with Redis caching
	discoverableServerRepo := postgres.NewDiscoverableServerRepository(db)
	discoverableServerService := services.NewDiscoverableServerService(
		discoverableServerRepo,
		repos.Servers,
		repos.Servers, // ServerRepository implements MemberRepo (GetMember, AddMember)
	)
	if redisCache != nil {
		cachedDiscoverableService := services.NewCachedDiscoverableServerService(discoverableServerService, redisCache)
		h.SetDiscoverableServerHandler(cachedDiscoverableService.DiscoverableServerService, serverService)
		log.Printf("✅ Enhanced Server Discovery service initialized (with Redis caching)")
	} else {
		h.SetDiscoverableServerHandler(discoverableServerService, serverService)
		log.Printf("✅ Enhanced Server Discovery service initialized (without caching)")
	}

	// Initialize App Directory service and handler
	appDirectoryRepo := postgres.NewAppDirectoryRepository(db)
	appDirectoryService := services.NewAppDirectoryService(appDirectoryRepo)
	h.SetAppDirectoryHandler(appDirectoryService)
	log.Printf("✅ App Directory service initialized")

	// Initialize Forum Tags service and handler
	forumTagService := services.NewForumTagService(repos.ForumTags, repos.Threads, repos.Channels, repos.Servers, permService)
	h.SetForumTagsHandler(forumTagService, threadService)
	log.Printf("✅ Forum tags service initialized")

	// Initialize Server Audio Settings service and handler
	serverAudioSettingsService := services.NewServerAudioSettingsService(repos.ServerAudioSettings, serviceBus)
	h.SetServerAudioSettingsHandler(serverAudioSettingsService)
	log.Printf("✅ Server audio settings service initialized")

	// Initialize Slash Command service and handler
	slashCmdRepo := postgres.NewSlashCommandRepository(db)
	interactionTokenRepo := postgres.NewInteractionTokenRepository(db)
	slashCmdService := services.NewSlashCommandService(
		slashCmdRepo,
		nil, // webhook commander
		permService,
		redisCache,
	)
	h.SetSlashCommandHandler(slashCmdService, permService)
	log.Printf("✅ Slash command service initialized")

	// Initialize Interaction service and handler
	interactionService := services.NewInteractionService(
		interactionTokenRepo,
		slashCmdService,
		nil, // webhook commander
		redisCache,
	)
	h.SetInteractionHandler(interactionService)
	log.Printf("✅ Interaction service initialized")

	// Initialize Premium & Billing services
	billingService := services.NewBillingService(repos.Premium, cfg.StripeSecretKey, cfg.IsProduction)
	premiumService := services.NewPremiumService(repos.Premium, repos.Users, repos.Servers, billingService)
	h.SetPremiumHandler(premiumService, billingService)
	log.Printf("✅ Premium & Billing services initialized")


	m := middleware.NewMiddleware(cfg.SecretKey)

	// Wire up API rate limiter for per-endpoint rate limiting
	if cfg.RateLimitEnabled {
		if redisCache != nil {
			// Redis available: use Redis-backed distributed rate limiter
			redisConfig := ratelimit.DefaultRedisLimiterConfig()
			redisConfig.KeyPrefix = "hearth:ratelimit:"
			redisConfig.DefaultWindow = cfg.RateLimitWindow
			redisConfig.DefaultMax = cfg.RateLimitMax
			redisConfig.FallbackOnError = true

			// Create memory fallback cache
			memoryCache := middleware.NewSimpleInMemoryCache()
			memoryLimiter := ratelimit.NewLimiter(memoryCache)

			// Create Redis limiter with memory fallback
			redisLimiter := ratelimit.NewRedisLimiter(redisCache.Client(), redisConfig, memoryLimiter)

			// Wrap with our adapter to implement api/middleware.RateLimiter interface
			adapter := middleware.NewHybridRateLimiterAdapter(redisLimiter, memoryLimiter)
			m.SetRateLimiter(adapter)
			log.Printf("✅ API rate limiter wired (Redis-backed, fallback=memory)")
		} else {
			// Redis not available: use in-memory rate limiter
			memoryCache := middleware.NewSimpleInMemoryCache()
			memoryLimiter := ratelimit.NewLimiter(memoryCache)
			adapter := middleware.NewMemoryRateLimiterAdapter(memoryLimiter)
			m.SetRateLimiter(adapter)
			log.Printf("✅ API rate limiter wired (memory-only, single-instance)")
		}
	} else {
		log.Printf("⚠️  API rate limiting DISABLED")
	}

	// Prometheus metrics endpoint (before API routes, no auth required)
	app.Get("/metrics", adaptor.HTTPHandler(promhttp.Handler()))
	log.Printf("📊 Prometheus metrics endpoint: /metrics")

	// Setup routes
	api.SetupRoutes(app, h, m)

	// Set component service on channel handler for message components support
	if h.Channels != nil {
		h.Channels.SetComponentService(componentService)
	}

	// Graceful shutdown signal handler with connection draining
	shutdownComplete := make(chan struct{})
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigCh
		log.Printf("Received %v signal, initiating graceful shutdown...", sig)

		// Create a context for the drain operation with overall timeout
		drainCtx, drainCancel := context.WithTimeout(context.Background(), cfg.DrainTimeout+5*time.Second)
		defer drainCancel()

		// Step 1: Start draining WebSocket connections
		log.Println("📤 Step 1/3: Draining WebSocket connections...")
		if wsGateway != nil {
			if err := wsGateway.Shutdown(drainCtx); err != nil {
				log.Printf("⚠️  Gateway drain error: %v", err)
			}
		}

		// Step 2: Stop accepting new HTTP requests
		log.Println("🛑 Step 2/3: Stopping HTTP server...")
		if err := app.ShutdownWithContext(drainCtx); err != nil {
			log.Printf("⚠️  HTTP shutdown error: %v", err)
		}

		// Step 3: Cancel the main context to stop background goroutines
		log.Println("🔄 Step 3/3: Stopping background services...")
		cancel()

		close(shutdownComplete)
	}()

	// Start server
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	go func() {
		log.Printf("Listening on %s", addr)
		if err := app.Listen(addr); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for shutdown to complete
	<-shutdownComplete
	log.Println("✅ Graceful shutdown complete")
}

