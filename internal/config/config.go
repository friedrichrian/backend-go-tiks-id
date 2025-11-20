package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort string
}

func Load() *Config {
	_ = godotenv.Load()

	return &Config{
		AppPort: getEnv("APP_PORT", "8080"),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
