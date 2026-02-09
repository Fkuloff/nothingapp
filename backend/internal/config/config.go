// internal/config/config.go
package config

import (
	"log"
	"os"

	"messenger/internal/storage"

	"github.com/joho/godotenv"
)

type Config struct {
	Storage   *storage.StorageConfig
	DBURL     string
	JWTSecret string
}

func LoadConfig() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatal("DB_URL is not set")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET is not set")
	}

	if len(jwtSecret) < 32 {
		log.Fatal("JWT_SECRET must be at least 32 characters long")
	}

	return &Config{
		DBURL:     dbURL,
		JWTSecret: jwtSecret,
		Storage:   storage.LoadStorageConfig(),
	}
}
