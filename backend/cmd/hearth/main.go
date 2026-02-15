package main

import (
	"context"
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
		Workers:        cfg.BcryptPoolWorkers,  // 0 = NumCPU (auto)
		QueueSize:      cfg.BcryptPoolQueue,    // 0 = Workers * 10 (auto)
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
	quotaService := services.NewQuotaService(cfg.Quotas, nil, nil, nil)
	userService := services.NewUserService(repos.Users, nil, serviceBus)
	authService := services.NewAuthService(repos.Users, jwtService)
	roleService := services.NewRoleService(
		repos.Roles,
		repos.Servers,
		nil, // cache
		serviceBus,
	)
	serverService := services.NewServerService(
		repos.Servers,
		repos.Channels,
		repos.Roles,
		quotaService,
		nil, // cache
		serviceBus,
	)
	channelService := services.NewChannelService(
		repos.Channels,
		repos.Servers,
		nil, // cache
		serviceBus,
	)
	messageService := services.NewMessageService(
		repos.Messages,
		repos.Channels,
		repos.Servers,
		quotaService,
		nil, // rate limiter
		nil, // e2ee service
		nil, // cache
		serviceBus,
	)
	searchService := services.NewSearchService(
		nil, // search repo - TODO: add full-text search
		repos.Messages,
		repos.Channels,
		repos.Servers,
		repos.Users,
		nil, // cache
	)
	typingService := services.NewTypingService(serviceBus)

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
	})

	// Security middleware
	app.Use(recover.New())

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
	// Thread service - TODO: Initialize with proper repository when available
	var threadService *services.ThreadService = nil

	h := handlers.NewHandlersWithTyping(authService, userService, serverService, channelService, messageService, roleService, searchService, threadService, typingService, wsGateway)
	m := middleware.NewMiddleware(cfg.SecretKey)

	// Prometheus metrics endpoint (before API routes, no auth required)
	app.Get("/metrics", adaptor.HTTPHandler(promhttp.Handler()))
	log.Printf("📊 Prometheus metrics endpoint: /metrics")

	// Setup routes
	api.SetupRoutes(app, h, m)

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

	// Keep references to avoid unused variable errors during development
	_ = repos
	_ = quotaService
	_ = userService
	_ = authService
	_ = serverService
	_ = channelService
	_ = messageService
	_ = wsHub
	_ = redisCache
}
