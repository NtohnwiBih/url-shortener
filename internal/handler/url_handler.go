package handler

import (
	"errors"
	"net/http"
	
	"github.com/gin-gonic/gin"
	
	"url-shortener/internal/domain"
	"url-shortener/internal/service"
	"url-shortener/pkg/logger"
)

// URLHandler handles HTTP requests for URL shortening operations
type URLHandler struct {
	service service.URLService
	logger  *logger.Logger
}

// NewURLHandler creates a new URL handler with dependencies
func NewURLHandler(service service.URLService, logger *logger.Logger) *URLHandler {
	return &URLHandler{
		service: service,
		logger:  logger,
	}
}

// ShortenURL handles POST /api/v1/shorten
// Creates a new shortened URL
func (h *URLHandler) ShortenURL(c *gin.Context) {
	var req domain.CreateURLRequest
	
	// Bind and validate request body
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid request body", "error", err)
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body: " + err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}
	
	// Get client IP for tracking
	clientIP := c.ClientIP()
	
	// Call service layer
	response, err := h.service.ShortenURL(c.Request.Context(), &req, clientIP)
	if err != nil {
		h.handleError(c, err)
		return
	}
	
	// Return success response
	c.JSON(http.StatusCreated, response)
}

// RedirectURL handles GET /:shortCode
// Redirects to the original URL
func (h *URLHandler) RedirectURL(c *gin.Context) {
	shortCode := c.Param("shortCode")
	
	// Validate short code format
	if shortCode == "" {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{
			Error:   "invalid_short_code",
			Message: "Short code is required",
			Code:    http.StatusBadRequest,
		})
		return
	}
	
	// Get original URL from service
	originalURL, err := h.service.GetOriginalURL(c.Request.Context(), shortCode)
	if err != nil {
		h.handleError(c, err)
		return
	}
	
	// Perform 301 permanent redirect for SEO benefits
	// Use 302 temporary redirect if you want to always track clicks
	c.Redirect(http.StatusMovedPermanently, originalURL)
}

// GetURLInfo handles GET /api/v1/urls/:shortCode
// Returns detailed information about a shortened URL
func (h *URLHandler) GetURLInfo(c *gin.Context) {
	shortCode := c.Param("shortCode")
	
	if shortCode == "" {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{
			Error:   "invalid_short_code",
			Message: "Short code is required",
			Code:    http.StatusBadRequest,
		})
		return
	}
	
	// Get URL info from service
	url, err := h.service.GetURLInfo(c.Request.Context(), shortCode)
	if err != nil {
		h.handleError(c, err)
		return
	}
	
	c.JSON(http.StatusOK, url)
}

// DeleteURL handles DELETE /api/v1/urls/:shortCode
// Removes a shortened URL
func (h *URLHandler) DeleteURL(c *gin.Context) {
	shortCode := c.Param("shortCode")
	
	if shortCode == "" {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{
			Error:   "invalid_short_code",
			Message: "Short code is required",
			Code:    http.StatusBadRequest,
		})
		return
	}
	
	// Optional: Add authentication check here
	// if !h.isAuthorized(c) { ... }
	
	if err := h.service.DeleteURL(c.Request.Context(), shortCode); err != nil {
		h.handleError(c, err)
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message": "URL deleted successfully",
		"code":    shortCode,
	})
}

// GetStats handles GET /api/v1/urls/:shortCode/stats
// Returns statistics for a shortened URL
func (h *URLHandler) GetStats(c *gin.Context) {
	shortCode := c.Param("shortCode")
	
	if shortCode == "" {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{
			Error:   "invalid_short_code",
			Message: "Short code is required",
			Code:    http.StatusBadRequest,
		})
		return
	}
	
	stats, err := h.service.GetStats(c.Request.Context(), shortCode)
	if err != nil {
		h.handleError(c, err)
		return
	}
	
	c.JSON(http.StatusOK, stats)
}

// handleError processes domain errors and returns appropriate HTTP responses
func (h *URLHandler) handleError(c *gin.Context, err error) {
	var appErr *domain.AppError
	
	switch {
	case errors.As(err, &appErr):
		// Log internal errors but don't expose details to users
		if appErr.Internal {
			h.logger.Error("Internal server error", "error", appErr.Err)
			c.JSON(appErr.StatusCode, domain.ErrorResponse{
				Error:   "internal_error",
				Message: "An internal error occurred",
				Code:    appErr.StatusCode,
			})
		} else {
			c.JSON(appErr.StatusCode, domain.ErrorResponse{
				Error:   "client_error",
				Message: appErr.Message,
				Code:    appErr.StatusCode,
			})
		}
	
	case errors.Is(err, domain.ErrURLNotFound):
		c.JSON(http.StatusNotFound, domain.ErrorResponse{
			Error:   "not_found",
			Message: "The requested URL was not found",
			Code:    http.StatusNotFound,
		})
	
	case errors.Is(err, domain.ErrURLExpired):
		c.JSON(http.StatusGone, domain.ErrorResponse{
			Error:   "url_expired",
			Message: "This URL has expired and is no longer available",
			Code:    http.StatusGone,
		})
	
	case errors.Is(err, domain.ErrShortCodeTaken):
		c.JSON(http.StatusConflict, domain.ErrorResponse{
			Error:   "short_code_taken",
			Message: "This short code is already in use",
			Code:    http.StatusConflict,
		})
	
	case errors.Is(err, domain.ErrInvalidURL):
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{
			Error:   "invalid_url",
			Message: "The provided URL is invalid",
			Code:    http.StatusBadRequest,
		})
	
	case errors.Is(err, domain.ErrRateLimitExceeded):
		c.JSON(http.StatusTooManyRequests, domain.ErrorResponse{
			Error:   "rate_limit_exceeded",
			Message: "Too many requests, please try again later",
			Code:    http.StatusTooManyRequests,
		})
	
	default:
		h.logger.Error("Unexpected error", "error", err)
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{
			Error:   "internal_error",
			Message: "An unexpected error occurred",
			Code:    http.StatusInternalServerError,
		})
	}
}