package baza

import (
	"database/sql"
	"fmt"
	"gs/config"

	"github.com/go-redis/redis/v8"
)

func ConnectionDb() (*sql.DB, error) {
	conf := config.Load()
	conDb := fmt.Sprintf("host=%s port=%d user=%s dbname=%s password=%s sslmode=disable",
		conf.DB_HOST, conf.DB_PORT, conf.DB_USER, conf.DB_NAME, conf.DB_PASSWORD)
	db, err := sql.Open("postgres", conDb)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}

func ConnectDB() *redis.Client {
	cfg := config.Load()

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RD_HOST,
		Password: cfg.RD_PASSWORD,
		DB:       cfg.RD_NAME,
	})

	return rdb
}
