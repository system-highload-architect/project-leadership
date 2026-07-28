package main

import (
	"encoding/json"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{
			"version": "v1",
			"message": "Hello from service v1",
		})
	})
	log.Println("Service v1 starting on :8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}
