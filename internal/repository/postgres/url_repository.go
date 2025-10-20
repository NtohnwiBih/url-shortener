package postgres

import (
	"context"
	"errors"
	"time"
	
	"gorm.io/gorm"
	
	"url-shortener/internal/domain"
	"url-shortener/internal/repository"
)

// urlRepository implements the URLRepository interface for PostgreSQL
type urlRepository struct {
	db *gorm.DB
}

// NewURLRepository creates a new PostgreSQL URL repository
func NewURLRepository(db *gorm.DB) repository.URLRepository {
	return &urlRepository{db: db}
}

// Create inserts a new URL record into the database
// Uses GORM's Create method with proper error handling
func (r *urlRepository) Create(ctx context.Context, url *domain.URL) error {
	result := r.db.WithContext(ctx).Create(url)
	if result.Error != nil {
		// Check for unique constraint violation (duplicate short code)
		if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
			return domain.ErrShortCodeTaken
		}
		return domain.NewInternalError(result.Error)
	}
	return nil
}

// FindByShortCode retrieves a URL by its short code
// Returns ErrURLNotFound if the code doesn't exist
func (r *urlRepository) FindByShortCode(ctx context.Context, shortCode string) (*domain.URL, error) {
	var url domain.URL
	
	// Use First to get a single record, with index hint for performance
	result := r.db.WithContext(ctx).
		Where("short_code = ? AND is_active = ?", shortCode, true).
		First(&url)
	
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, domain.ErrURLNotFound
		}
		return nil, domain.NewInternalError(result.Error)
	}
	
	return &url, nil
}

// FindByOriginalURL checks if an original URL already exists
// This helps prevent duplicate URLs and can be used for deduplication
func (r *urlRepository) FindByOriginalURL(ctx context.Context, originalURL string) (*domain.URL, error) {
	var url domain.URL
	
	result := r.db.WithContext(ctx).
		Where("original_url = ? AND is_active = ?", originalURL, true).
		First(&url)
	
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, domain.ErrURLNotFound
		}
		return nil, domain.NewInternalError(result.Error)
	}
	
	return &url, nil
}

// Update modifies an existing URL record
func (r *urlRepository) Update(ctx context.Context, url *domain.URL) error {
	result := r.db.WithContext(ctx).Save(url)
	if result.Error != nil {
		return domain.NewInternalError(result.Error)
	}
	
	if result.RowsAffected == 0 {
		return domain.ErrURLNotFound
	}
	
	return nil
}

// Delete soft-deletes a URL by setting is_active to false
// This preserves data for analytics while preventing access
func (r *urlRepository) Delete(ctx context.Context, shortCode string) error {
	result := r.db.WithContext(ctx).
		Model(&domain.URL{}).
		Where("short_code = ?", shortCode).
		Update("is_active", false)
	
	if result.Error != nil {
		return domain.NewInternalError(result.Error)
	}
	
	if result.RowsAffected == 0 {
		return domain.ErrURLNotFound
	}
	
	return nil
}

// IncrementClickCount atomically increments the click counter
// Uses SQL UPDATE to ensure thread-safety without SELECT-then-UPDATE race condition
func (r *urlRepository) IncrementClickCount(ctx context.Context, shortCode string) error {
	now := time.Now()
	
	// Use raw SQL for atomic increment to prevent race conditions
	result := r.db.WithContext(ctx).
		Model(&domain.URL{}).
		Where("short_code = ? AND is_active = ?", shortCode, true).
		Updates(map[string]interface{}{
			"click_count":    gorm.Expr("click_count + ?", 1),
			"last_access_at": now,
		})
	
	if result.Error != nil {
		return domain.NewInternalError(result.Error)
	}
	
	if result.RowsAffected == 0 {
		return domain.ErrURLNotFound
	}
	
	return nil
}

// GetStats retrieves comprehensive statistics for a URL
func (r *urlRepository) GetStats(ctx context.Context, shortCode string) (*domain.URLStats, error) {
	var url domain.URL
	
	result := r.db.WithContext(ctx).
		Where("short_code = ?", shortCode).
		First(&url)
	
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, domain.ErrURLNotFound
		}
		return nil, domain.NewInternalError(result.Error)
	}
	
	stats := &domain.URLStats{
		ShortCode:    url.ShortCode,
		OriginalURL:  url.OriginalURL,
		TotalClicks:  url.ClickCount,
		CreatedAt:    url.CreatedAt,
		LastAccessAt: url.LastAccessAt,
		ExpiresAt:    url.ExpiresAt,
		IsActive:     url.IsActive,
	}
	
	// Calculate days remaining if URL has expiration
	if url.ExpiresAt != nil {
		remaining := int(time.Until(*url.ExpiresAt).Hours() / 24)
		if remaining >= 0 {
			stats.DaysRemaining = &remaining
		}
	}
	
	return stats, nil
}

// DeleteExpired removes all URLs that have passed their expiration date
// This should be called periodically by a cleanup job
func (r *urlRepository) DeleteExpired(ctx context.Context) (int64, error) {
	result := r.db.WithContext(ctx).
		Model(&domain.URL{}).
		Where("expires_at IS NOT NULL AND expires_at < ?", time.Now()).
		Update("is_active", false)
	
	if result.Error != nil {
		return 0, domain.NewInternalError(result.Error)
	}
	
	return result.RowsAffected, nil
}

// ExistsByShortCode checks if a short code exists without loading the full record
// More efficient than FindByShortCode when you only need existence check
func (r *urlRepository) ExistsByShortCode(ctx context.Context, shortCode string) (bool, error) {
	var count int64
	
	result := r.db.WithContext(ctx).
		Model(&domain.URL{}).
		Where("short_code = ? AND is_active = ?", shortCode, true).
		Count(&count)
	
	if result.Error != nil {
		return false, domain.NewInternalError(result.Error)
	}
	
	return count > 0, nil
}