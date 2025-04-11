package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/spf13/cast"
)

type Config struct {
	DB_USER     string
	DB_PASSWORD string
	DB_NAME     string
	DB_HOST     string
	DB_PORT     int
	RD_HOST     string
	RD_PASSWORD string
	RD_NAME     int
}

func Load() Config {
	if err := godotenv.Load(".env"); err != nil {
		log.Print("No .env file found?")
	}

	config := Config{}
	config.DB_USER = cast.ToString(Coalesce("DB_USER", "macbookpro"))
	config.DB_HOST = cast.ToString(Coalesce("DB_HOST", "localhost"))
	config.DB_NAME = cast.ToString(Coalesce("DB_NAME", "google_docs"))
	config.DB_PASSWORD = cast.ToString(Coalesce("DB_PASSWORD", "1111"))
	config.DB_PORT = cast.ToInt(Coalesce("DB_PORT", 5432))
	config.RD_HOST = cast.ToString(Coalesce("RD_HOST", "localhost:6379"))
	config.RD_PASSWORD = cast.ToString(Coalesce("RD_PASSWORD", ""))
	config.RD_NAME = cast.ToInt(Coalesce("RD_NAME", 0))

	return config
}

func Coalesce(key string, defaultValue interface{}) interface{} {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return defaultValue
}
