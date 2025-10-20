package validator

import (
	"net/url"
	"regexp"
	"strings"
)

var (
	// urlRegex is a comprehensive URL validation regex
	urlRegex = regexp.MustCompile(`^(https?|ftp)://[^\s/$.?#].[^\s]*$`)
	
	// shortCodeRegex validates short code format (alphanumeric, hyphens, underscores)
	shortCodeRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	
	// allowedSchemes lists permitted URL schemes
	allowedSchemes = map[string]bool{
		"http":  true,
		"https": true,
		"ftp":   true,
	}
)

// ValidateURL checks if a string is a valid URL
func ValidateURL(rawURL string) error {
	if rawURL == "" {
		return &ValidationError{Field: "url", Message: "URL cannot be empty"}
	}

	// Basic regex check
	if !urlRegex.MatchString(rawURL) {
		return &ValidationError{Field: "url", Message: "Invalid URL format"}
	}

	// Parse URL
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return &ValidationError{Field: "url", Message: "Invalid URL structure"}
	}

	// Validate scheme
	if !allowedSchemes[strings.ToLower(parsed.Scheme)] {
		return &ValidationError{Field: "url", Message: "Unsupported URL scheme"}
	}

	// Validate host
	if parsed.Host == "" {
		return &ValidationError{Field: "url", Message: "URL must contain a host"}
	}

	// Validate length (reasonable maximum)
	if len(rawURL) > 2048 {
		return &ValidationError{Field: "url", Message: "URL too long (max 2048 characters)"}
	}

	return nil
}

// ValidateShortCode checks if a short code has valid format
func ValidateShortCode(code string) bool {
	if len(code) < 2 || len(code) > 50 {
		return false
	}
	return shortCodeRegex.MatchString(code)
}

// NormalizeURL standardizes URL format
func NormalizeURL(rawURL string) string {
	// Ensure scheme
	if !strings.Contains(rawURL, "://") {
		rawURL = "https://" + rawURL
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL // Return original if parsing fails
	}

	// Force lowercase scheme and host
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	parsed.Host = strings.ToLower(parsed.Host)

	// Remove trailing slash from path
	parsed.Path = strings.TrimSuffix(parsed.Path, "/")

	return parsed.String()
}

// IsSafeURL checks if URL points to potentially dangerous protocols
func IsSafeURL(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}

	// Block dangerous schemes
	dangerousSchemes := map[string]bool{
		"javascript": true,
		"data":       true,
		"vbscript":   true,
	}

	return !dangerousSchemes[strings.ToLower(parsed.Scheme)]
}

// ValidationError represents a validation failure
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}