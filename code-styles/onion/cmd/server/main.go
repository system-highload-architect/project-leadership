package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"onion-architecture/internal/application/robot"
	delivery "onion-architecture/internal/delivery/http"
	"onion-architecture/internal/domain/services"
	"onion-architecture/internal/infrastructure/console"
	"onion-architecture/internal/infrastructure/inmemory"
)

func main() {
	// === Сборка зависимостей (слои от внешних к внутренним) ===

	// Infrastructure (адаптеры)
	repo := inmemory.NewRobotRepository()
	notifier := console.NewNotifier()

	// Domain Services
	assigner := &services.DefaultAssigner{}

	// Application (Use Cases)
	getRobotUC := robot.NewGetRobotUseCase(repo)
	assignTaskUC := robot.NewAssignTaskUseCase(repo, assigner, notifier)

	// Delivery (HTTP)
	handler := delivery.NewRobotHandler(getRobotUC, assignTaskUC)

	// HTTP роутер
	mux := http.NewServeMux()
	mux.HandleFunc("GET /robots/{id}", handler.GetRobot)
	mux.HandleFunc("POST /robots/{id}/tasks", handler.AssignTask)

	server := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		log.Println("Starting Onion Architecture server on :8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	log.Println("shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("server shutdown error: %v", err)
	}
	log.Println("server stopped")
}
