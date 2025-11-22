package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort string
	DBUser  string
	DBPass  string
	DBHost  string
	DBPort  string
	DBName  string
}

func Load() *Config {
	_ = godotenv.Load()

	return &Config{
		AppPort: getEnv("APP_PORT", "8080"),
		DBUser:  getEnv("DB_USER", "root"),
		DBPass:  getEnv("DB_PASS", "rian"),
		DBHost:  getEnv("DB_HOST", "127.0.0.1"),
		DBPort:  getEnv("DB_PORT", "3306"),
		DBName:  getEnv("DB_NAME", "tiks-id"),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
