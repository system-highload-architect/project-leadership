package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	delivery "layered-architecture/internal/delivery/http"
	"layered-architecture/internal/repository/inmemory"
	"layered-architecture/internal/service"
)

func main() {
	// Инициализация слоёв (снизу вверх)
	repo := inmemory.NewRobotRepository()
	svc := service.NewRobotService(repo)
	handler := delivery.NewRobotHandler(svc)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /robots/{id}", handler.GetRobot)
	mux.HandleFunc("PUT /robots/{id}/status", handler.UpdateStatus)

	server := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		log.Println("Starting layered-architecture server on :8080")
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
