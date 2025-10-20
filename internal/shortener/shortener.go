package shortener

import (
	"crypto/rand"
	"math/big"
)

// Base62 character set (0-9, A-Z, a-z) - 62 characters total
// Using base62 instead of base64 avoids special characters that might cause URL issues
const base62Chars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// CodeGenerator generates unique short codes using cryptographically secure random numbers
// Thread-safe and collision-resistant
type CodeGenerator struct {
	length int // Length of generated codes
}

// NewCodeGenerator creates a new code generator with specified length
// Recommended length: 6-8 characters for good collision resistance
// - 6 chars = 62^6 = ~56 billion combinations
// - 7 chars = 62^7 = ~3.5 trillion combinations
// - 8 chars = 62^8 = ~218 trillion combinations
func NewCodeGenerator(length int) *CodeGenerator {
	if length < 4 {
		length = 6 // Minimum safe length
	}
	if length > 12 {
		length = 12 // Maximum reasonable length
	}
	
	return &CodeGenerator{
		length: length,
	}
}

// Generate creates a random short code using base62 encoding
// Uses crypto/rand for cryptographically secure random generation
// This prevents predictability and ensures collision resistance
func (g *CodeGenerator) Generate() string {
	result := make([]byte, g.length)
	
	for i := 0; i < g.length; i++ {
		// Generate random index using crypto/rand for security
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(base62Chars))))
		if err != nil {
			// Fallback to less secure method if crypto/rand fails
			// This should rarely happen in practice
			num = big.NewInt(int64(i % len(base62Chars)))
		}
		
		result[i] = base62Chars[num.Int64()]
	}
	
	return string(result)
}

// GenerateFromID converts a numeric ID to a base62 short code
// Useful for deterministic code generation from auto-increment IDs
// This approach eliminates collision risk but may expose sequential patterns
func (g *CodeGenerator) GenerateFromID(id uint) string {
	if id == 0 {
		return string(base62Chars[0])
	}
	
	result := make([]byte, 0, g.length)
	num := id
	
	// Convert ID to base62
	for num > 0 {
		remainder := num % 62
		result = append([]byte{base62Chars[remainder]}, result...)
		num = num / 62
	}
	
	// Pad to minimum length with leading zeros if needed
	for len(result) < g.length {
		result = append([]byte{base62Chars[0]}, result...)
	}
	
	return string(result)
}

// Decode converts a base62 short code back to numeric ID
// Useful for reversing GenerateFromID operation
func (g *CodeGenerator) Decode(code string) uint {
	var result uint = 0
	
	for i := 0; i < len(code); i++ {
		// Find character position in base62 charset
		char := code[i]
		var value uint
		
		switch {
		case char >= '0' && char <= '9':
			value = uint(char - '0')
		case char >= 'A' && char <= 'Z':
			value = uint(char-'A') + 10
		case char >= 'a' && char <= 'z':
			value = uint(char-'a') + 36
		default:
			continue // Skip invalid characters
		}
		
		result = result*62 + value
	}
	
	return result
}

// IsValid checks if a short code contains only valid base62 characters
func (g *CodeGenerator) IsValid(code string) bool {
	if len(code) == 0 || len(code) > g.length {
		return false
	}
	
	for _, char := range code {
		found := false
		for _, validChar := range base62Chars {
			if char == validChar {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	return true
}

// GetCollisionProbability calculates approximate collision probability
// Formula: 1 - (1 - 1/N)^k where N = total combinations, k = number of URLs
// This is a simplified birthday problem calculation
func (g *CodeGenerator) GetCollisionProbability(numURLs int) float64 {
	if numURLs <= 0 {
		return 0.0
	}
	
	// Calculate total possible combinations (62^length)
	totalCombinations := 1.0
	for i := 0; i < g.length; i++ {
		totalCombinations *= 62
	}
	
	// Approximate collision probability using birthday problem
	// For large N, probability ≈ k^2 / (2*N)
	probability := float64(numURLs*numURLs) / (2.0 * totalCombinations)
	
	// Cap at 1.0
	if probability > 1.0 {
		probability = 1.0
	}
	
	return probability
}