package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	delivery "clean-architecture/internal/delivery/http"
	"clean-architecture/internal/repository/inmemory"
	"clean-architecture/internal/usecase/robot"
)

func main() {
	// Инициализация репозитория (in-memory для примера)
	robotRepo := inmemory.NewRobotRepository()

	// Инициализация Use Cases
	getRobotUC := robot.NewGetRobotUseCase(robotRepo)
	updateStatusUC := robot.NewUpdateStatusUseCase(robotRepo)

	// Инициализация HTTP-хендлера
	robotHandler := delivery.NewRobotHandler(getRobotUC, updateStatusUC)

	// Настройка маршрутов
	mux := http.NewServeMux()
	mux.HandleFunc("GET /robots/{id}", robotHandler.GetRobot)
	mux.HandleFunc("PUT /robots/{id}/status", robotHandler.UpdateStatus)

	// Запуск сервера
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

	// Ожидание сигнала завершения
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
