package main

import (
	"context"
	"log"
	"os"
	"sync"
	"time"

	"github.com/iamgak/grpc_gis/services"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	// Setup structured logging
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	// Load configuration
	cfg := LoadConfig()

	// Initialize Redis client
	redisClient := NewRedisClient(cfg.RedisAddr, cfg.RedisPass)

	// Initialize GIS service

	gisService := services.NewMadinaGisService(cfg.GISUsername, cfg.GISPassword, cfg.GISClientIP, cfg.Port, redisClient)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()

	// init grpc server and start running it in new goroutine

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func(ctx context.Context) {
		grpc := Init(gisService, logger)
		grpc.GrpcServer.Activate(ctx)
		wg.Done()
	}(ctx)

	// Check if running in live reload mode
	if os.Getenv("AIR") == "true" {
		logger.Info("🚀 Running in live reload mode")
	}

	logger.Info("🚀 GIS API Server running on :50051 👿❤️‍🔥")
	wg.Wait()
	logger.Warn("🙯 GIS API Server shutting down on the basis of context🧊")
}
