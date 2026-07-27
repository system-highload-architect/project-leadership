package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"hexagonal-architecture/internal/adapters/driven/console"
	"hexagonal-architecture/internal/adapters/driven/inmemory"
	driving "hexagonal-architecture/internal/adapters/driving/http"
	"hexagonal-architecture/internal/core/usecases"
)

func main() {
	// === Сборка зависимостей (DI) ===
	// Driven Adapters
	robotRepo := inmemory.NewRobotRepository()
	notifier := console.NewNotifier()

	// Use Cases (ядро)
	getRobotUC := usecases.NewGetRobotUseCase(robotRepo)
	updateStatusUC := usecases.NewUpdateStatusUseCase(robotRepo, notifier)

	// Driving Adapter
	robotHandler := driving.NewRobotHandler(getRobotUC, updateStatusUC)

	// HTTP роутер
	mux := http.NewServeMux()
	mux.HandleFunc("GET /robots/{id}", robotHandler.GetRobot)
	mux.HandleFunc("PUT /robots/{id}/status", robotHandler.UpdateStatus)

	server := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// Graceful shutdown
	go func() {
		log.Println("Starting server on :8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown error: %v", err)
	}
	log.Println("Server stopped")
}
