package domain

import (
	"time"
)

// URL represents a shortened URL entry in the system
// This is the core domain entity that models our business concept
type URL struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	ShortCode    string    `gorm:"uniqueIndex;not null;size:12" json:"short_code"`
	OriginalURL  string    `gorm:"not null;type:text" json:"original_url"`
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime" json:"updated_at"`
	ExpiresAt    *time.Time `gorm:"index" json:"expires_at,omitempty"` // Nullable for non-expiring URLs
	ClickCount   int64     `gorm:"default:0" json:"click_count"`
	LastAccessAt *time.Time `json:"last_access_at,omitempty"`
	CreatorIP    string    `gorm:"size:45" json:"-"` // IPv6 max length, not exposed in JSON
	IsActive     bool      `gorm:"default:true;index" json:"is_active"`
	CustomAlias  bool      `gorm:"default:false" json:"custom_alias"` // User-defined vs auto-generated
}

// TableName specifies the table name for GORM
func (URL) TableName() string {
	return "urls"
}

// IsExpired checks if the URL has expired
func (u *URL) IsExpired() bool {
	if u.ExpiresAt == nil {
		return false // Never expires
	}
	return time.Now().After(*u.ExpiresAt)
}

// IncrementClickCount safely increments the click counter
// This should be called atomically in the repository layer
func (u *URL) IncrementClickCount() {
	u.ClickCount++
	now := time.Now()
	u.LastAccessAt = &now
}

// URLStats represents aggregated statistics for a shortened URL
type URLStats struct {
	ShortCode     string    `json:"short_code"`
	OriginalURL   string    `json:"original_url"`
	TotalClicks   int64     `json:"total_clicks"`
	CreatedAt     time.Time `json:"created_at"`
	LastAccessAt  *time.Time `json:"last_access_at,omitempty"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty"`
	IsActive      bool      `json:"is_active"`
	DaysRemaining *int      `json:"days_remaining,omitempty"` // Calculated field
}

// CreateURLRequest represents the request payload for creating a short URL
type CreateURLRequest struct {
	URL         string `json:"url" binding:"required"`          // Original URL to shorten
	CustomAlias string `json:"custom_alias,omitempty"`          // Optional custom short code
	ExpiryDays  int    `json:"expiry_days,omitempty"`           // Optional expiration in days
}

// CreateURLResponse represents the response after creating a short URL
type CreateURLResponse struct {
	ShortCode   string    `json:"short_code"`
	ShortURL    string    `json:"short_url"`    // Full shortened URL
	OriginalURL string    `json:"original_url"`
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

// ErrorResponse represents a standard error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Code    int    `json:"code"`
}

// HealthResponse represents health check response
type HealthResponse struct {
	Status    string    `json:"status"`
	Service   string    `json:"service"`
	Version   string    `json:"version"`
	Timestamp time.Time `json:"timestamp"`
}