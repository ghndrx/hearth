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
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"hearth/internal/api"
	"hearth/internal/api/handlers"
	"hearth/internal/api/middleware"
	"hearth/internal/auth"
	"hearth/internal/config"
	"hearth/internal/database/postgres"
	"hearth/internal/events"
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

	// Initialize auth services
	jwtService := auth.NewJWTService(
		cfg.SecretKey,
		15*time.Minute, // Access token expiry
		7*24*time.Hour, // Refresh token expiry
	)

	// Initialize WebSocket hub
	wsHub := websocket.NewHub()
	go wsHub.Run(context.Background())

	// Initialize WebSocket gateway
	wsGateway := websocket.NewGateway(wsHub, jwtService, nil)

	// Initialize event bridge (connects domain events to WebSocket)
	_ = websocket.NewEventBridge(wsHub, eventBus)

	// Initialize services
	quotaService := services.NewQuotaService(cfg.Quotas, nil, nil, nil)
	userService := services.NewUserService(repos.Users, nil, serviceBus)
	authService := services.NewAuthService(repos.Users)
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

	// Rate limiting
	app.Use(limiter.New(limiter.Config{
		Max:               100,
		Expiration:        60 * time.Second,
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
	h := handlers.NewHandlers(authService, userService, serverService, channelService, messageService, roleService, wsGateway)
	m := middleware.NewMiddleware(cfg.SecretKey)

	// Setup routes
	api.SetupRoutes(app, h, m)

	// Graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("Received shutdown signal...")
		cancel()
	}()

	// Start server
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	go func() {
		log.Printf("Listening on %s", addr)
		if err := app.Listen(addr); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("Shutting down gracefully...")
	app.Shutdown()

	// Keep references to avoid unused variable errors during development
	_ = repos
	_ = quotaService
	_ = userService
	_ = authService
	_ = serverService
	_ = channelService
	_ = messageService
}
