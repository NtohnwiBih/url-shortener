package repository

import (
	"context"
	"url-shortener/internal/domain"
)

// URLRepository defines the contract for URL data access
// This interface allows us to swap implementations (PostgreSQL, MySQL, MongoDB, etc.)
// without changing business logic - following Dependency Inversion Principle
type URLRepository interface {
	// Create stores a new shortened URL in the database
	Create(ctx context.Context, url *domain.URL) error
	
	// FindByShortCode retrieves a URL by its short code
	FindByShortCode(ctx context.Context, shortCode string) (*domain.URL, error)
	
	// FindByOriginalURL checks if an original URL already has a short code
	FindByOriginalURL(ctx context.Context, originalURL string) (*domain.URL, error)
	
	// Update modifies an existing URL record
	Update(ctx context.Context, url *domain.URL) error
	
	// Delete removes a URL by its short code
	Delete(ctx context.Context, shortCode string) error
	
	// IncrementClickCount atomically increments the click counter
	// This prevents race conditions with concurrent requests
	IncrementClickCount(ctx context.Context, shortCode string) error
	
	// GetStats retrieves statistics for a short URL
	GetStats(ctx context.Context, shortCode string) (*domain.URLStats, error)
	
	// DeleteExpired removes all expired URLs (cleanup job)
	DeleteExpired(ctx context.Context) (int64, error)
	
	// ExistsByShortCode checks if a short code exists without fetching data
	ExistsByShortCode(ctx context.Context, shortCode string) (bool, error)
}