
package integration_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"url-shortener/internal/cache"
	"url-shortener/internal/config"
	"url-shortener/internal/domain"
	"url-shortener/internal/handler"
	postgresRepo "url-shortener/internal/repository/postgres"
	"url-shortener/internal/service"
	"url-shortener/pkg/logger"
)

type URLShortenerIntegrationTestSuite struct {
	suite.Suite
	router   *gin.Engine
	db       *gorm.DB
	cache    cache.Cache
	config   *config.Config
	logger   *logger.Logger
	cleanup  func()
}

func (suite *URLShortenerIntegrationTestSuite) SetupSuite() {
	suite.logger = logger.NewLogger()
	
	// Setup test configuration
	suite.config = &config.Config{
		Environment:        "test",
		ServerPort:         "8081",
		BaseURL:            "http://localhost:8081",
		ShortCodeLength:    6,
		RateLimitPerMinute: 1000, // High limit for tests
		CacheTTL:           time.Hour,
		URLExpirationDays:  7,
	}
	
	// Setup test database
	dsn := "host=localhost user=test password=test dbname=urlshortener_test port=5432 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		suite.T().Fatal("Failed to connect to test database:", err)
	}
	suite.db = db
	
	// Run migrations
	err = db.AutoMigrate(&domain.URL{})
	if err != nil {
		suite.T().Fatal("Failed to run migrations:", err)
	}
	
	// Setup Redis cache (using mock or test instance)
	suite.cache, err = cache.NewRedisCache("localhost:6379", "", 1)
	if err != nil {
		suite.T().Log("Redis not available, continuing without cache")
		suite.cache = nil
	}
	
	// Setup application layers
	repo := postgresRepo.NewURLRepository(db)
	urlService := service.NewURLService(repo, suite.cache, suite.config, suite.logger)
	urlHandler := handler.NewURLHandler(urlService, suite.logger)
	
	// Setup router
	suite.router = gin.New()
	suite.router.Use(gin.Recovery())
	suite.router.Use(handler.LoggerMiddleware(suite.logger))
	
	// Register routes
	suite.router.POST("/api/v1/shorten", urlHandler.ShortenURL)
	suite.router.GET("/:shortCode", urlHandler.RedirectURL)
	suite.router.GET("/api/v1/urls/:shortCode", urlHandler.GetURLInfo)
	suite.router.GET("/api/v1/urls/:shortCode/stats", urlHandler.GetStats)
	suite.router.DELETE("/api/v1/urls/:shortCode", urlHandler.DeleteURL)
	suite.router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})
	
	// Setup cleanup function
	suite.cleanup = func() {
		// Clean test database
		db.Exec("DELETE FROM urls")
		if suite.cache != nil {
			suite.cache.Close()
		}
	}
}

func (suite *URLShortenerIntegrationTestSuite) TearDownSuite() {
	if suite.cleanup != nil {
		suite.cleanup()
	}
}

func (suite *URLShortenerIntegrationTestSuite) SetupTest() {
	// Clean data before each test
	suite.db.Exec("DELETE FROM urls")
	if suite.cache != nil {
		// Flush test Redis database if cache is available
		// This would depend on your cache implementation
		// For now, we'll just skip since Redis might not be available in tests
	}
}

func TestURLShortenerIntegrationTestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	suite.Run(t, new(URLShortenerIntegrationTestSuite))
}

func (suite *URLShortenerIntegrationTestSuite) TestShortenAndRedirect() {
	// Test shortening a URL
	shortenReq := map[string]interface{}{
		"url": "https://example.com/very/long/path/to/resource",
	}
	
	shortenBody, _ := json.Marshal(shortenReq)
	req := httptest.NewRequest("POST", "/api/v1/shorten", strings.NewReader(string(shortenBody)))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	
	assert.Equal(suite.T(), http.StatusCreated, w.Code)
	
	var shortenResp domain.CreateURLResponse
	err := json.Unmarshal(w.Body.Bytes(), &shortenResp)
	assert.NoError(suite.T(), err)
	
	assert.NotEmpty(suite.T(), shortenResp.ShortCode)
	assert.Equal(suite.T(), "https://example.com/very/long/path/to/resource", shortenResp.OriginalURL)
	assert.Contains(suite.T(), shortenResp.ShortURL, suite.config.BaseURL)
	
	// Test redirecting to original URL
	redirectReq := httptest.NewRequest("GET", fmt.Sprintf("/%s", shortenResp.ShortCode), nil)
	redirectW := httptest.NewRecorder()
	
	suite.router.ServeHTTP(redirectW, redirectReq)
	
	assert.Equal(suite.T(), http.StatusMovedPermanently, redirectW.Code)
	assert.Equal(suite.T(), "https://example.com/very/long/path/to/resource", redirectW.Header().Get("Location"))
}

func (suite *URLShortenerIntegrationTestSuite) TestGetURLInfo() {
	// First create a short URL
	shortenReq := map[string]interface{}{
		"url": "https://example.com/info-test",
	}
	
	shortenBody, _ := json.Marshal(shortenReq)
	req := httptest.NewRequest("POST", "/api/v1/shorten", strings.NewReader(string(shortenBody)))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	
	var shortenResp domain.CreateURLResponse
	json.Unmarshal(w.Body.Bytes(), &shortenResp)
	
	// Test getting URL info
	infoReq := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/urls/%s", shortenResp.ShortCode), nil)
	infoW := httptest.NewRecorder()
	
	suite.router.ServeHTTP(infoW, infoReq)
	
	assert.Equal(suite.T(), http.StatusOK, infoW.Code)
	
	var urlInfo domain.URL
	err := json.Unmarshal(infoW.Body.Bytes(), &urlInfo)
	assert.NoError(suite.T(), err)
	
	assert.Equal(suite.T(), shortenResp.ShortCode, urlInfo.ShortCode)
	assert.Equal(suite.T(), "https://example.com/info-test", urlInfo.OriginalURL)
	assert.True(suite.T(), urlInfo.IsActive)
}

func (suite *URLShortenerIntegrationTestSuite) TestCustomAlias() {
	customAlias := "my-custom-link"
	
	shortenReq := map[string]interface{}{
		"url":          "https://example.com/custom",
		"custom_alias": customAlias,
	}
	
	shortenBody, _ := json.Marshal(shortenReq)
	req := httptest.NewRequest("POST", "/api/v1/shorten", strings.NewReader(string(shortenBody)))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	
	assert.Equal(suite.T(), http.StatusCreated, w.Code)
	
	var shortenResp domain.CreateURLResponse
	json.Unmarshal(w.Body.Bytes(), &shortenResp)
	
	assert.Equal(suite.T(), customAlias, shortenResp.ShortCode)
	
	// Test that duplicate custom alias fails
	duplicateReq := httptest.NewRequest("POST", "/api/v1/shorten", strings.NewReader(string(shortenBody)))
	duplicateReq.Header.Set("Content-Type", "application/json")
	
	duplicateW := httptest.NewRecorder()
	suite.router.ServeHTTP(duplicateW, duplicateReq)
	
	assert.Equal(suite.T(), http.StatusConflict, duplicateW.Code)
}

func (suite *URLShortenerIntegrationTestSuite) TestInvalidURL() {
	shortenReq := map[string]interface{}{
		"url": "not-a-valid-url",
	}
	
	shortenBody, _ := json.Marshal(shortenReq)
	req := httptest.NewRequest("POST", "/api/v1/shorten", strings.NewReader(string(shortenBody)))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	
	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
}

func (suite *URLShortenerIntegrationTestSuite) TestHealthCheck() {
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	
	suite.router.ServeHTTP(w, req)
	
	assert.Equal(suite.T(), http.StatusOK, w.Code)
	
	var healthResp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &healthResp)
	assert.NoError(suite.T(), err)
	
	assert.Equal(suite.T(), "healthy", healthResp["status"])
}