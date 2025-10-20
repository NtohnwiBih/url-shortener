// cmd/server/main.go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"url-shortener/internal/cache"
	"url-shortener/internal/config"
	"url-shortener/internal/handler"
	postgresRepo "url-shortener/internal/repository/postgres"
	"url-shortener/internal/service"
	customLogger "url-shortener/pkg/logger"
)

// gormWriter wraps our custom logger to implement gorm's logger.Writer interface
type gormWriter struct {
	logger *customLogger.Logger
}

// Printf implements the logger.Writer interface
func (w *gormWriter) Printf(format string, args ...interface{}) {
	w.logger.Info(fmt.Sprintf(format, args...))
}

func main() {
	// Simple health check for Docker - just make HTTP request to existing server
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		resp, err := http.Get("http://localhost:8081/health")
		if err != nil || resp.StatusCode != http.StatusOK {
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Load environment variables from .env file (development only)
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using environment variables")
	}

	// Initialize structured logger
	appLogger := customLogger.NewLogger()
	appLogger.Info("Starting URL Shortener Service")

	// Load application configuration 
	cfg, err := config.LoadConfig()
	if err != nil {
		appLogger.Fatal("Failed to load configuration", "error", err)
	}

	// Initialize database connection
	db, err := initDatabase(cfg, appLogger)
	if err != nil {
		appLogger.Fatal("Failed to initialize database", "error", err)
	}

	// Initialize Redis cache
	redisCache, err := cache.NewRedisCache(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	if err != nil {
		appLogger.Warn("Failed to initialize Redis cache, continuing without cache", "error", err)
		redisCache = nil // Continue without cache
	}

	// Initialize repository layer
	urlRepo := postgresRepo.NewURLRepository(db)

	// Initialize service layer with dependency injection
	urlService := service.NewURLService(urlRepo, redisCache, cfg, appLogger)

	// Initialize HTTP handler
	urlHandler := handler.NewURLHandler(urlService, appLogger)

	// Setup HTTP router with middleware
	router := setupRouter(urlHandler, cfg, appLogger)

	// Create HTTP server with timeouts
	srv := &http.Server{
		Addr:           fmt.Sprintf(":%s", cfg.ServerPort),
		Handler:        router,
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   15 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}

	// Start server in a goroutine for graceful shutdown
	go func() {
		appLogger.Info("Server starting", "port", cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			appLogger.Fatal("Failed to start server", "error", err)
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	appLogger.Info("Shutting down server...")

	// Graceful shutdown with 30 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		appLogger.Error("Server forced to shutdown", "error", err)
	}

	// Close Redis connection
	if redisCache != nil {
		if err := redisCache.Close(); err != nil {
			appLogger.Error("Error closing Redis connection", "error", err)
		}
	}

	appLogger.Info("Server exited successfully")
}

// initDatabase initializes the PostgreSQL database connection with connection pooling
func initDatabase(cfg *config.Config, log *customLogger.Logger) (*gorm.DB, error) {
	writer := &gormWriter{logger: log}
	
	gormLogger := logger.New(
		writer, // Use our custom writer
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: true,
			Colorful:                  false,
		},
	)

	// Connect to PostgreSQL with retry logic
	var db *gorm.DB
	var err error
	
	maxRetries := 5
	for i := 0; i < maxRetries; i++ {
		dsn := fmt.Sprintf(
			"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=UTC",
			cfg.DBHost, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBPort, cfg.DBSSLMode,
		)

		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger:                 gormLogger,
			SkipDefaultTransaction: true,
			PrepareStmt:            true,
		})
		
		if err == nil {
			break
		}
		
		log.Warn("Failed to connect to database, retrying...", "attempt", i+1, "error", err)
		time.Sleep(5 * time.Second)
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database after %d attempts: %w", maxRetries, err)
	}

	// Get underlying SQL DB for connection pool configuration
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	// Configure connection pool for optimal performance
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// Verify database connection
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Info("Database connection established successfully")
	return db, nil
}

// setupRouter configures the Gin router with middleware and routes
func setupRouter(urlHandler *handler.URLHandler, cfg *config.Config, log *customLogger.Logger) *gin.Engine {
	// Set Gin mode based on environment
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Apply global middleware
	router.Use(gin.Recovery()) // Panic recovery
	router.Use(handler.LoggerMiddleware(log))
	router.Use(handler.CORSMiddleware(cfg))
	router.Use(handler.SecurityHeadersMiddleware())
	router.Use(handler.RateLimitMiddleware(cfg.RateLimitPerMinute))

	// Health check endpoint (no authentication required)
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "url-shortener",
			"version": "1.0.0",
		})
	})

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// URL shortening endpoints
		v1.POST("/shorten", urlHandler.ShortenURL)        // Create short URL
		v1.GET("/urls/:shortCode", urlHandler.GetURLInfo) // Get URL details
		v1.DELETE("/urls/:shortCode", urlHandler.DeleteURL) // Delete URL (optional auth)
		v1.GET("/urls/:shortCode/stats", urlHandler.GetStats) // Get click statistics
	}

	// Short URL redirection (public endpoint)
	router.GET("/:shortCode", urlHandler.RedirectURL)

	// 404 handler
	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "endpoint not found",
		})
	})

	return router
}