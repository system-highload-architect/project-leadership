package main

import (
	"log"
	"net/http"
	"os"

	"serverless/internal/handler"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/", handler.Handle)

	log.Printf("Function listening on :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
