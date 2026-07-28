package main

import (
	"log"
	"net/http"

	"microkernel/internal/core"
	"microkernel/internal/plugins/calculator"
	"microkernel/internal/plugins/greeting"
)

func main() {
	// Создаём ядро
	kernel := core.NewKernel()

	// Регистрируем плагины
	kernel.RegisterPlugin(greeting.NewPlugin())
	kernel.RegisterPlugin(calculator.NewPlugin())

	// Запускаем HTTP-сервер ядра
	mux := http.NewServeMux()
	mux.HandleFunc("GET /plugin/{name}", kernel.HandlePlugin)

	log.Println("Microkernel server starting on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
