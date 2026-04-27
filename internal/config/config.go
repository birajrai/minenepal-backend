package config

import (
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
)

type Config struct {
	Port                string
	Host                string
	CacheTTL            time.Duration
	BannerCacheTTL      time.Duration
	CacheDir            string
	ServerQueryTimeout time.Duration
	VotifierTimeout    time.Duration
	LogLevel           zerolog.Level
	LogPretty          bool
}

func Load() *Config {
	c := &Config{}

	godotenv.Load()

	c.Port = getEnv("PORT", "10000")
	c.Host = getEnv("HOST", "0.0.0.0")

	c.CacheDir = getEnv("CACHE_DIR", "./cache")
	c.ServerQueryTimeout = parseDuration(getEnv("SERVER_QUERY_TIMEOUT", "5s"))
	c.VotifierTimeout = parseDuration(getEnv("VOTIFIER_TIMEOUT", "5s"))

	disableCache := getEnv("DISABLE_CACHE", "false") == "true"
	if disableCache {
		c.CacheTTL = 0
		c.BannerCacheTTL = 0
	} else {
		c.CacheTTL = parseDuration(getEnv("CACHE_TTL", "15s"))
		c.BannerCacheTTL = parseDuration(getEnv("BANNER_CACHE_TTL", "24h"))
	}

	logLevel := getEnv("LOG_LEVEL", "info")
	c.LogLevel = parseLogLevel(logLevel)

	logPrettyStr := getEnv("LOG_PRETTY", "true")
	c.LogPretty = logPrettyStr == "true" || logPrettyStr == "1"

	return c
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 15 * time.Second
	}
	return d
}

func parseLogLevel(s string) zerolog.Level {
	switch s {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	default:
		return zerolog.InfoLevel
	}
}