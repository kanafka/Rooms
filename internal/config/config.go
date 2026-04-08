package config

import (
	"os"

	"github.com/google/uuid"
)

type Config struct {
	DatabaseURL string
	JWTSecret   string
	Port        string
	AdminUUID   uuid.UUID
	UserUUID    uuid.UUID
}

func Load() *Config {
	adminUUID := uuid.MustParse(getEnv("ADMIN_UUID", "00000000-0000-0000-0000-000000000001"))
	userUUID := uuid.MustParse(getEnv("USER_UUID", "00000000-0000-0000-0000-000000000002"))

	return &Config{
		DatabaseURL: getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/booking?sslmode=disable"),
		JWTSecret:   getEnv("JWT_SECRET", "secret"),
		Port:        getEnv("PORT", "8080"),
		AdminUUID:   adminUUID,
		UserUUID:    userUUID,
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
