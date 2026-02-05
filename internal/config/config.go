package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port           string
	AllowedOrigins []string
	RedisURL       string
	AuthToken      string
	RateLimit      int
	MaxMessageSize int64
	MessageTTL     int // hours
	MaxMessages    int // per room
}

func Load() *Config {
	cfg := &Config{
		Port:           getEnv("PORT", "8080"),
		RedisURL:       os.Getenv("REDIS_URL"),
		AuthToken:      os.Getenv("AUTH_TOKEN"),
		RateLimit:      getEnvInt("RATE_LIMIT", 60),
		MaxMessageSize: int64(getEnvInt("MAX_MESSAGE_SIZE", 4096)),
		MessageTTL:     getEnvInt("MESSAGE_TTL_HOURS", 24),
		MaxMessages:    getEnvInt("MAX_MESSAGES_PER_ROOM", 100),
	}

	originsStr := getEnv("ALLOWED_ORIGINS", "*")
	if originsStr == "*" {
		cfg.AllowedOrigins = []string{"*"}
	} else {
		cfg.AllowedOrigins = strings.Split(originsStr, ",")
		for i, origin := range cfg.AllowedOrigins {
			cfg.AllowedOrigins[i] = strings.TrimSpace(origin)
		}
	}

	return cfg
}

func (c *Config) IsOriginAllowed(origin string) bool {
	if len(c.AllowedOrigins) == 1 && c.AllowedOrigins[0] == "*" {
		return true
	}
	for _, allowed := range c.AllowedOrigins {
		if allowed == origin {
			return true
		}
	}
	return false
}

func (c *Config) AuthEnabled() bool {
	return c.AuthToken != ""
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}
