package config

import (
	"errors"
	"io/fs"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config is loaded entirely from environment variables (12-factor).
type Config struct {
	Port                string
	MongoURI            string
	MongoDB             string
	MongoConnectTimeout time.Duration
	JWTSecret           string
	JWTExpiresIn        time.Duration
	CORSOrigins         []string
	SeedEnabled         bool
	CacheEnabled        bool
	ValkeyAddr          string
	ValkeyPassword      string
	ValkeyDB            int
	CacheStatsTTL       time.Duration
	CacheUserTTL        time.Duration
}

// Load reads configuration from the environment. A .env file in the working directory is loaded
// first if present; variables already set in the real environment always take precedence.
func Load() Config {
	if err := godotenv.Load(); err == nil {
		log.Println("loaded configuration from .env")
	} else if !errors.Is(err, fs.ErrNotExist) {
		log.Fatalf(".env: %v", err)
	}

	cfg := Config{
		Port:                getEnv("PORT", "8080"),
		MongoURI:            getEnv("MONGO_URI", "mongodb://localhost:27017"),
		MongoDB:             getEnv("MONGO_DB", "projecthub"),
		MongoConnectTimeout: getDuration("MONGO_CONNECT_TIMEOUT", 30*time.Second),
		JWTSecret:           os.Getenv("JWT_SECRET"),
		JWTExpiresIn:        getDuration("JWT_EXPIRES_IN", 24*time.Hour),
		CORSOrigins:         strings.Split(getEnv("CORS_ORIGINS", "http://localhost:5173"), ","),
		SeedEnabled:         getEnv("SEED_ENABLED", "true") == "true",
		CacheEnabled:        getEnv("CACHE_ENABLED", "true") == "true",
		ValkeyAddr:          getEnv("VALKEY_ADDR", "localhost:6379"),
		ValkeyPassword:      os.Getenv("VALKEY_PASSWORD"),
		ValkeyDB:            getInt("VALKEY_DB", 0),
		CacheStatsTTL:       getDuration("CACHE_STATS_TTL", 2*time.Minute),
		CacheUserTTL:        getDuration("CACHE_USER_TTL", time.Minute),
	}
	if len(cfg.JWTSecret) < 32 {
		log.Fatal("JWT_SECRET must be set and at least 32 characters long")
	}
	return cfg
}

func getEnv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func getInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		log.Fatalf("%s: invalid number %q", key, v)
	}
	return n
}

func getDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		log.Fatalf("%s: invalid duration %q (examples: 30s, 15m, 24h)", key, v)
	}
	return d
}
