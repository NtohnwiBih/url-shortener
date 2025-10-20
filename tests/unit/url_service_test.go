package unit

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"url-shortener/internal/config"
	"url-shortener/internal/domain"
	"url-shortener/internal/service"
	"url-shortener/pkg/logger"
)

// MockURLRepository is a mock implementation of URLRepository
type MockURLRepository struct {
	mock.Mock
}

func (m *MockURLRepository) Create(ctx context.Context, url *domain.URL) error {
	args := m.Called(ctx, url)
	return args.Error(0)
}

func (m *MockURLRepository) FindByShortCode(ctx context.Context, shortCode string) (*domain.URL, error) {
	args := m.Called(ctx, shortCode)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.URL), args.Error(1)
}

func (m *MockURLRepository) FindByOriginalURL(ctx context.Context, originalURL string) (*domain.URL, error) {
	args := m.Called(ctx, originalURL)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.URL), args.Error(1)
}

func (m *MockURLRepository) Update(ctx context.Context, url *domain.URL) error {
	args := m.Called(ctx, url)
	return args.Error(0)
}

func (m *MockURLRepository) Delete(ctx context.Context, shortCode string) error {
	args := m.Called(ctx, shortCode)
	return args.Error(0)
}

func (m *MockURLRepository) IncrementClickCount(ctx context.Context, shortCode string) error {
	args := m.Called(ctx, shortCode)
	return args.Error(0)
}

func (m *MockURLRepository) GetStats(ctx context.Context, shortCode string) (*domain.URLStats, error) {
	args := m.Called(ctx, shortCode)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.URLStats), args.Error(1)
}

func (m *MockURLRepository) DeleteExpired(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockURLRepository) ExistsByShortCode(ctx context.Context, shortCode string) (bool, error) {
	args := m.Called(ctx, shortCode)
	return args.Bool(0), args.Error(1)
}

// MockCache is a mock implementation of Cache
type MockCache struct {
	mock.Mock
}

func (m *MockCache) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	args := m.Called(ctx, key, value, ttl)
	return args.Error(0)
}

func (m *MockCache) Get(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	return args.String(0), args.Error(1)
}

func (m *MockCache) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockCache) Exists(ctx context.Context, key string) (bool, error) {
	args := m.Called(ctx, key)
	return args.Bool(0), args.Error(1)
}

func (m *MockCache) Close() error {
	args := m.Called()
	return args.Error(0)
}

type URLServiceTestSuite struct {
	repo     *MockURLRepository
	cache    *MockCache
	cfg      *config.Config
	logger   *logger.Logger
	service  service.URLService
}

func setupURLServiceTest(t *testing.T) *URLServiceTestSuite {
	repo := new(MockURLRepository)
	cache := new(MockCache)
	
	cfg := &config.Config{
		BaseURL:              "https://short.url",
		ShortCodeLength:      6,
		CacheTTL:             time.Hour,
		URLExpirationDays:    30,
		EnableAuthentication: false,
	}
	
	logger := logger.NewLogger()
	service := service.NewURLService(repo, cache, cfg, logger)
	
	return &URLServiceTestSuite{
		repo:    repo,
		cache:   cache,
		cfg:     cfg,
		logger:  logger,
		service: service,
	}
}

func TestShortenURL_Success(t *testing.T) {
	suite := setupURLServiceTest(t)
	ctx := context.Background()
	
	req := &domain.CreateURLRequest{
		URL: "https://example.com/very/long/url",
	}
	
	// Mock repository calls
	suite.repo.On("FindByOriginalURL", ctx, "https://example.com/very/long/url").
		Return((*domain.URL)(nil), domain.ErrURLNotFound)
	suite.repo.On("ExistsByShortCode", ctx, mock.AnythingOfType("string")).
		Return(false, nil)
	suite.repo.On("Create", ctx, mock.AnythingOfType("*domain.URL")).
		Return(nil)
	suite.cache.On("Set", ctx, mock.AnythingOfType("string"), "https://example.com/very/long/url", time.Hour).
		Return(nil)
	
	resp, err := suite.service.ShortenURL(ctx, req, "192.168.1.1")
	
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "https://example.com/very/long/url", resp.OriginalURL)
	assert.Contains(t, resp.ShortURL, "https://short.url/")
	
	suite.repo.AssertExpectations(t)
	suite.cache.AssertExpectations(t)
}

func TestShortenURL_DuplicateURL(t *testing.T) {
	suite := setupURLServiceTest(t)
	ctx := context.Background()
	
	req := &domain.CreateURLRequest{
		URL: "https://example.com/duplicate",
	}
	
	existingURL := &domain.URL{
		ShortCode:   "abc123",
		OriginalURL: "https://example.com/duplicate",
		CreatedAt:   time.Now(),
		IsActive:    true,
	}
	
	// Mock repository to return existing URL
	suite.repo.On("FindByOriginalURL", ctx, "https://example.com/duplicate").
		Return(existingURL, nil)
	
	resp, err := suite.service.ShortenURL(ctx, req, "192.168.1.1")
	
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "abc123", resp.ShortCode)
	
	suite.repo.AssertExpectations(t)
}

func TestShortenURL_CustomAlias(t *testing.T) {
	suite := setupURLServiceTest(t)
	ctx := context.Background()
	
	req := &domain.CreateURLRequest{
		URL:         "https://example.com/custom",
		CustomAlias: "myalias",
	}
	
	suite.repo.On("FindByOriginalURL", ctx, "https://example.com/custom").
		Return((*domain.URL)(nil), domain.ErrURLNotFound)
	suite.repo.On("ExistsByShortCode", ctx, "myalias").
		Return(false, nil)
	suite.repo.On("Create", ctx, mock.AnythingOfType("*domain.URL")).
		Return(nil)
	suite.cache.On("Set", ctx, "myalias", "https://example.com/custom", time.Hour).
		Return(nil)
	
	resp, err := suite.service.ShortenURL(ctx, req, "192.168.1.1")
	
	assert.NoError(t, err)
	assert.Equal(t, "myalias", resp.ShortCode)
	
	suite.repo.AssertExpectations(t)
	suite.cache.AssertExpectations(t)
}

func TestGetOriginalURL_CacheHit(t *testing.T) {
	suite := setupURLServiceTest(t)
	ctx := context.Background()
	
	suite.cache.On("Get", ctx, "abc123").
		Return("https://example.com/cached", nil)
	suite.repo.On("IncrementClickCount", mock.Anything, "abc123").
		Return(nil)
	
	originalURL, err := suite.service.GetOriginalURL(ctx, "abc123")
	
	assert.NoError(t, err)
	assert.Equal(t, "https://example.com/cached", originalURL)
	
	suite.cache.AssertExpectations(t)
	suite.repo.AssertNotCalled(t, "FindByShortCode")
}

func TestGetOriginalURL_CacheMiss(t *testing.T) {
	suite := setupURLServiceTest(t)
	ctx := context.Background()
	
	url := &domain.URL{
		ShortCode:   "abc123",
		OriginalURL: "https://example.com/notcached",
		IsActive:    true,
		ExpiresAt:   nil, // Never expires
	}
	
	suite.cache.On("Get", ctx, "abc123").
		Return("", nil) // Cache miss
	suite.repo.On("FindByShortCode", ctx, "abc123").
		Return(url, nil)
	suite.repo.On("IncrementClickCount", ctx, "abc123").
		Return(nil)
	suite.cache.On("Set", ctx, "abc123", "https://example.com/notcached", time.Hour).
		Return(nil)
	
	originalURL, err := suite.service.GetOriginalURL(ctx, "abc123")
	
	assert.NoError(t, err)
	assert.Equal(t, "https://example.com/notcached", originalURL)
	
	suite.repo.AssertExpectations(t)
	suite.cache.AssertExpectations(t)
}

func TestGetOriginalURL_Expired(t *testing.T) {
	suite := setupURLServiceTest(t)
	ctx := context.Background()
	
	expiry := time.Now().Add(-24 * time.Hour) // Expired yesterday
	url := &domain.URL{
		ShortCode:   "expired",
		OriginalURL: "https://example.com/expired",
		IsActive:    true,
		ExpiresAt:   &expiry,
	}
	
	suite.cache.On("Get", ctx, "expired").Return("", nil)
	suite.repo.On("FindByShortCode", ctx, "expired").Return(url, nil)
	
	_, err := suite.service.GetOriginalURL(ctx, "expired")
	
	assert.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrURLExpired))
}