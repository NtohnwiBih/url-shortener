package service

import (
	"context"
	"fmt"
	"time"
	
	"url-shortener/internal/cache"
	"url-shortener/internal/config"
	"url-shortener/internal/domain"
	"url-shortener/internal/repository"
	"url-shortener/internal/shortener"
	"url-shortener/pkg/logger"
	"url-shortener/pkg/validator"
)

// urlService implements the URLService interface
type urlService struct {
	repo      repository.URLRepository
	cache     cache.Cache
	cfg       *config.Config
	logger    *logger.Logger
	generator *shortener.CodeGenerator
}

// NewURLService creates a new URL service with dependencies injected
func NewURLService(
	repo repository.URLRepository,
	cache cache.Cache,
	cfg *config.Config,
	logger *logger.Logger,
) URLService {
	return &urlService{
		repo:      repo,
		cache:     cache,
		cfg:       cfg,
		logger:    logger,
		generator: shortener.NewCodeGenerator(cfg.ShortCodeLength),
	}
}

// ShortenURL creates a new shortened URL with validation and deduplication
func (s *urlService) ShortenURL(ctx context.Context, req *domain.CreateURLRequest, clientIP string) (*domain.CreateURLResponse, error) {
	// Step 1: Validate the original URL
	if err := validator.ValidateURL(req.URL); err != nil {
		s.logger.Warn("Invalid URL provided", "url", req.URL, "error", err)
		return nil, domain.NewValidationError("Invalid URL format")
	}
	
	// Step 2: Normalize URL (add https:// if missing, remove trailing slash)
	normalizedURL := validator.NormalizeURL(req.URL)
	
	// Step 3: Check if URL already exists (optional deduplication)
	// This prevents creating multiple short codes for the same URL
	existingURL, err := s.repo.FindByOriginalURL(ctx, normalizedURL)
	if err == nil && existingURL != nil && !existingURL.IsExpired() {
		s.logger.Info("URL already shortened, returning existing", "short_code", existingURL.ShortCode)
		return s.buildResponse(existingURL), nil
	}
	
	// Step 4: Generate or validate custom short code
	var shortCode string
	if req.CustomAlias != "" {
		// Validate custom alias format
		if !validator.ValidateShortCode(req.CustomAlias) {
			return nil, domain.NewValidationError("Custom alias contains invalid characters")
		}
		
		// Check if custom alias is already taken
		exists, err := s.repo.ExistsByShortCode(ctx, req.CustomAlias)
		if err != nil {
			s.logger.Error("Failed to check short code existence", "error", err)
			return nil, domain.NewInternalError(err)
		}
		if exists {
			return nil, domain.ErrShortCodeTaken
		}
		
		shortCode = req.CustomAlias
	} else {
		// Generate unique short code with collision handling
		shortCode, err = s.generateUniqueShortCode(ctx)
		if err != nil {
			s.logger.Error("Failed to generate short code", "error", err)
			return nil, domain.NewInternalError(err)
		}
	}
	
	// Step 5: Calculate expiration date if specified
	var expiresAt *time.Time
	if req.ExpiryDays > 0 {
		expiry := time.Now().AddDate(0, 0, req.ExpiryDays)
		expiresAt = &expiry
	} else if s.cfg.URLExpirationDays > 0 {
		// Use default expiration from config
		expiry := time.Now().AddDate(0, 0, s.cfg.URLExpirationDays)
		expiresAt = &expiry
	}
	
	// Step 6: Create URL entity
	url := &domain.URL{
		ShortCode:   shortCode,
		OriginalURL: normalizedURL,
		ExpiresAt:   expiresAt,
		CreatorIP:   clientIP,
		IsActive:    true,
		CustomAlias: req.CustomAlias != "",
		ClickCount:  0,
	}
	
	// Step 7: Save to database
	if err := s.repo.Create(ctx, url); err != nil {
		s.logger.Error("Failed to create URL", "error", err, "short_code", shortCode)
		return nil, err
	}
	
	// Step 8: Cache the URL for fast retrieval
	if s.cache != nil {
		if err := s.cache.Set(ctx, shortCode, normalizedURL, s.cfg.CacheTTL); err != nil {
			// Log cache error but don't fail the request
			s.logger.Warn("Failed to cache URL", "error", err, "short_code", shortCode)
		}
	}
	
	s.logger.Info("URL shortened successfully", 
		"short_code", shortCode, 
		"original_url", normalizedURL,
		"custom", req.CustomAlias != "",
	)
	
	return s.buildResponse(url), nil
}

// GetOriginalURL retrieves the original URL and tracks the access
// Uses cache-aside pattern for optimal performance
func (s *urlService) GetOriginalURL(ctx context.Context, shortCode string) (string, error) {
	// Step 1: Try to get from cache first (fast path)
	if s.cache != nil {
		cachedURL, err := s.cache.Get(ctx, shortCode)
		if err == nil && cachedURL != "" {
			// Cache hit - increment counter asynchronously to avoid blocking
			go func() {
				if err := s.repo.IncrementClickCount(context.Background(), shortCode); err != nil {
					s.logger.Error("Failed to increment click count", "error", err, "short_code", shortCode)
				}
			}()
			
			s.logger.Debug("Cache hit", "short_code", shortCode)
			return cachedURL, nil
		}
	}
	
	// Step 2: Cache miss or no cache - query database
	url, err := s.repo.FindByShortCode(ctx, shortCode)
	if err != nil {
		s.logger.Warn("Short code not found", "short_code", shortCode)
		return "", err
	}
	
	// Step 3: Check if URL has expired
	if url.IsExpired() {
		s.logger.Info("Attempted to access expired URL", "short_code", shortCode)
		return "", domain.ErrURLExpired
	}
	
	// Step 4: Increment click count
	if err := s.repo.IncrementClickCount(ctx, shortCode); err != nil {
		// Log but don't fail the redirect
		s.logger.Error("Failed to increment click count", "error", err, "short_code", shortCode)
	}
	
	// Step 5: Update cache for future requests
	if s.cache != nil {
		if err := s.cache.Set(ctx, shortCode, url.OriginalURL, s.cfg.CacheTTL); err != nil {
			s.logger.Warn("Failed to update cache", "error", err, "short_code", shortCode)
		}
	}
	
	s.logger.Info("URL accessed", "short_code", shortCode, "clicks", url.ClickCount+1)
	return url.OriginalURL, nil
}

// GetURLInfo returns detailed information about a shortened URL
func (s *urlService) GetURLInfo(ctx context.Context, shortCode string) (*domain.URL, error) {
	url, err := s.repo.FindByShortCode(ctx, shortCode)
	if err != nil {
		return nil, err
	}
	
	return url, nil
}

// DeleteURL removes a shortened URL and invalidates cache
func (s *urlService) DeleteURL(ctx context.Context, shortCode string) error {
	// Delete from database
	if err := s.repo.Delete(ctx, shortCode); err != nil {
		s.logger.Error("Failed to delete URL", "error", err, "short_code", shortCode)
		return err
	}
	
	// Invalidate cache
	if s.cache != nil {
		if err := s.cache.Delete(ctx, shortCode); err != nil {
			s.logger.Warn("Failed to delete from cache", "error", err, "short_code", shortCode)
		}
	}
	
	s.logger.Info("URL deleted", "short_code", shortCode)
	return nil
}

// GetStats returns detailed statistics for a shortened URL
func (s *urlService) GetStats(ctx context.Context, shortCode string) (*domain.URLStats, error) {
	stats, err := s.repo.GetStats(ctx, shortCode)
	if err != nil {
		return nil, err
	}
	
	return stats, nil
}

// generateUniqueShortCode generates a short code and ensures it's unique
// Implements collision handling with retry logic
func (s *urlService) generateUniqueShortCode(ctx context.Context) (string, error) {
	const maxRetries = 5
	
	for i := 0; i < maxRetries; i++ {
		// Generate random short code
		shortCode := s.generator.Generate()
		
		// Check if it already exists
		exists, err := s.repo.ExistsByShortCode(ctx, shortCode)
		if err != nil {
			return "", err
		}
		
		if !exists {
			return shortCode, nil
		}
		
		// Collision detected, log and retry
		s.logger.Warn("Short code collision detected, retrying", 
			"short_code", shortCode, 
			"attempt", i+1,
		)
	}
	
	return "", fmt.Errorf("failed to generate unique short code after %d attempts", maxRetries)
}

// buildResponse constructs the API response with full short URL
func (s *urlService) buildResponse(url *domain.URL) *domain.CreateURLResponse {
	return &domain.CreateURLResponse{
		ShortCode:   url.ShortCode,
		ShortURL:    fmt.Sprintf("%s/%s", s.cfg.BaseURL, url.ShortCode),
		OriginalURL: url.OriginalURL,
		CreatedAt:   url.CreatedAt,
		ExpiresAt:   url.ExpiresAt,
	}
}