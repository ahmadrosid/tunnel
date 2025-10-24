package subdomain

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
)

var validSubdomainPattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9\-]{0,61}[a-z0-9])?$`)

// Generate creates a random 8-character subdomain
func Generate() (string, error) {
	bytes := make([]byte, 4) // 4 bytes = 8 hex characters
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random subdomain: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

func Validate(subdomain string) error {
	subdomain = strings.ToLower(subdomain)

	if len(subdomain) < 1 || len(subdomain) > 63 {
		return fmt.Errorf("subdomain must be between 1 and 63 characters")
	}

	if !validSubdomainPattern.MatchString(subdomain) {
		return fmt.Errorf("subdomain must contain only lowercase letters, numbers, and hyphens")
	}

	reserved := []string{"www", "api", "admin", "mail", "ftp", "localhost"}
	for _, r := range reserved {
		if subdomain == r {
			return fmt.Errorf("subdomain '%s' is reserved", subdomain)
		}
	}

	return nil
}

func Normalize(subdomain string) string {
	return strings.ToLower(strings.TrimSpace(subdomain))
}
