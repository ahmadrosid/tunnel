package subdomain

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
)

// validSubdomainPattern ensures subdomain contains only alphanumeric and hyphens
var validSubdomainPattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9\-]{0,61}[a-z0-9])?$`)

// Generate creates a random 8-character subdomain
func Generate() (string, error) {
	bytes := make([]byte, 4) // 4 bytes = 8 hex characters
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random subdomain: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// Validate checks if a custom subdomain is valid
func Validate(subdomain string) error {
	// Convert to lowercase for validation
	subdomain = strings.ToLower(subdomain)

	// Check length
	if len(subdomain) < 1 || len(subdomain) > 63 {
		return fmt.Errorf("subdomain must be between 1 and 63 characters")
	}

	// Check format
	if !validSubdomainPattern.MatchString(subdomain) {
		return fmt.Errorf("subdomain must contain only lowercase letters, numbers, and hyphens")
	}

	// Reserved subdomains
	reserved := []string{"www", "api", "admin", "mail", "ftp", "localhost"}
	for _, r := range reserved {
		if subdomain == r {
			return fmt.Errorf("subdomain '%s' is reserved", subdomain)
		}
	}

	return nil
}

// Normalize converts a subdomain to lowercase and trims whitespace
func Normalize(subdomain string) string {
	return strings.ToLower(strings.TrimSpace(subdomain))
}
