package service

import (
	"context"
	"url-shortener/internal/domain"
)

// URLService defines the business logic interface for URL operations
// This layer orchestrates between repositories, cache, and external services
type URLService interface {
	// ShortenURL creates a new shortened URL
	ShortenURL(ctx context.Context, req *domain.CreateURLRequest, clientIP string) (*domain.CreateURLResponse, error)
	
	// GetOriginalURL retrieves and redirects to the original URL
	GetOriginalURL(ctx context.Context, shortCode string) (string, error)
	
	// GetURLInfo returns detailed information about a shortened URL
	GetURLInfo(ctx context.Context, shortCode string) (*domain.URL, error)
	
	// DeleteURL removes a shortened URL
	DeleteURL(ctx context.Context, shortCode string) error
	
	// GetStats returns statistics for a shortened URL
	GetStats(ctx context.Context, shortCode string) (*domain.URLStats, error)
}