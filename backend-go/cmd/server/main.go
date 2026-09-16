package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"quadis-backend-go/internal/api"
	"quadis-backend-go/internal/config"
	"quadis-backend-go/internal/repository"
	"quadis-backend-go/internal/repository/memory"
	"quadis-backend-go/internal/repository/postgres"
	"quadis-backend-go/internal/service/ai"
	"quadis-backend-go/internal/service/auth"
	"quadis-backend-go/internal/service/booking"
	"quadis-backend-go/internal/service/ota"
	"quadis-backend-go/internal/service/payment"
)

func main() {
	cfg := config.Load()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. Data Store selection (PostgreSQL vs In-Memory)
	var repo repository.Repository
	if cfg.DatabaseURL != "" {
		pgStore, err := postgres.NewPostgresStore(ctx, cfg.DatabaseURL)
		if err != nil {
			log.Fatalf("❌ Database connection failed: %v", err)
		}
		if err := pgStore.Migrate(ctx); err != nil {
			log.Fatalf("❌ Database schema migration failed: %v", err)
		}
		repo = pgStore
		log.Println("🐘 Connected to PostgreSQL and migrated schema successfully")
	} else {
		memStore := memory.NewMemoryStore()
		repo = memStore
		log.Println("🧠 Running in-memory store (seeded 10 properties, 21 room categories, 205 keys)")
	}

	// 2. Services
	sessionService := auth.NewSessionService(cfg)
	bookingService := booking.NewBookingService(repo)
	razorpayService := payment.NewRazorpayService(cfg)
	otaService := ota.NewOTAService(repo, cfg)
	aiService := ai.NewAIService(repo, cfg)

	// 3. Background Workers
	bookingService.StartCleanupWorker(ctx, 60*time.Second, 15)
	otaService.StartChannelSyncWorker(ctx, 30*time.Second, 5)

	// 4. HTTP Router
	router := api.NewRouter(api.RouterDeps{
		Config:          cfg,
		Repo:            repo,
		SessionService:  sessionService,
		BookingService:  bookingService,
		RazorpayService: razorpayService,
		OTAService:      otaService,
		AIService:       aiService,
	})

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// 5. Graceful shutdown handler
	shutdownDone := make(chan os.Signal, 1)
	signal.Notify(shutdownDone, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("🚀 Quadis Hotels API Server running strictly on Golang at http://localhost:%d", cfg.Port)
		log.Printf("📡 Health Check endpoint: http://localhost:%d/api/health", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ HTTP server listen error: %v", err)
		}
	}()

	<-shutdownDone
	log.Println("🛑 Shutdown signal received, terminating gracefully...")

	cancel() // Stop background workers

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("⚠️ Server forced to shutdown: %v", err)
	}

	log.Println("✅ Quadis Go server exited cleanly")
}
