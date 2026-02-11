package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"hearth/internal/api"
	"hearth/internal/api/handlers"
	"hearth/internal/api/middleware"
	"hearth/internal/config"
	"hearth/internal/database/postgres"
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

	// Initialize services
	quotaService := services.NewQuotaService(cfg.Quotas, nil, nil, nil)

	// Initialize WebSocket hub
	wsHub := websocket.NewHub()
	go wsHub.Run(context.Background())

	// Initialize Fiber app with security settings
	app := fiber.New(fiber.Config{
		AppName:               "Hearth",
		DisableStartupMessage: true,
		BodyLimit:             100 * 1024 * 1024, // 100MB
		ReadTimeout:           30,
		WriteTimeout:          30,
		// Security
		EnableTrustedProxyCheck: true,
		ProxyHeader:             "X-Forwarded-For",
	})

	// Security middleware
	app.Use(recover.New())

	// Helmet for security headers
	app.Use(helmet.New(helmet.Config{
		XSSProtection:         "1; mode=block",
		ContentTypeNosniff:    "nosniff",
		XFrameOptions:         "SAMEORIGIN",
		ReferrerPolicy:        "strict-origin-when-cross-origin",
		CrossOriginEmbedderPolicy: "require-corp",
		CrossOriginOpenerPolicy:   "same-origin",
		CrossOriginResourcePolicy: "same-origin",
		PermissionPolicy:          "camera=(), microphone=(), geolocation=()",
	}))

	// Rate limiting
	app.Use(limiter.New(limiter.Config{
		Max:               100,
		Expiration:        60,
		LimiterMiddleware: limiter.SlidingWindow{},
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(429).JSON(fiber.Map{
				"error": "rate_limited",
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
	h := handlers.NewHandlers(nil, nil, nil, wsHub)
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

	_ = repos
	_ = quotaService
}
