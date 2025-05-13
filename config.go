package main

import (
	"log"
	"os"

	"github.com/go-redis/redis"
	"github.com/joho/godotenv"
)

// NewRedisClient initializes a Redis client
func NewRedisClient(addr, password string) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       0,
	})
}

// Config holds application configuration
type Config struct {
	GISUsername string
	GISPassword string
	GISClientIP string
	RedisAddr   string
	RedisPass   string
	Port        string
}

// LoadConfig loads environment variables
func LoadConfig() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: No .env file found")
	}

	return &Config{
		GISUsername: os.Getenv("GIS_USERNAME"),
		GISPassword: os.Getenv("GIS_PASSWORD"),
		GISClientIP: os.Getenv("GIS_CLIENT_IP"),
		RedisAddr:   os.Getenv("REDIS_ADDR"),
		RedisPass:   os.Getenv("REDIS_PASS"),
		Port:        os.Getenv("PORT"),
	}
}
