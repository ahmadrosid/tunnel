package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds the server configuration
type Config struct {
	SSHPort           int
	Domain            string
	HTTPPort          int
	HTTPSPort         int
	HostKeyPath       string
	CertCacheDir      string
	LetsEncryptEmail  string
	RequestTimeout    time.Duration
	EnableHTTPS       bool
}

// Load reads configuration from environment variables with defaults
func Load() *Config {
	return &Config{
		SSHPort:          getEnvAsInt("SSH_PORT", 2222),
		Domain:           getEnv("DOMAIN", "unggahin.com"),
		HTTPPort:         getEnvAsInt("HTTP_PORT", 80),
		HTTPSPort:        getEnvAsInt("HTTPS_PORT", 443),
		HostKeyPath:      getEnv("HOST_KEY_PATH", "./ssh_host_key"),
		CertCacheDir:     getEnv("CERT_CACHE_DIR", "./certs"),
		LetsEncryptEmail: getEnv("LETSENCRYPT_EMAIL", ""),
		RequestTimeout:   getEnvAsDuration("REQUEST_TIMEOUT", 30*time.Second),
		EnableHTTPS:      getEnvAsBool("ENABLE_HTTPS", true),
	}
}

// getEnv reads an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt reads an environment variable as integer or returns a default value
func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvAsBool reads an environment variable as boolean or returns a default value
func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

// getEnvAsDuration reads an environment variable as duration or returns a default value
func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
