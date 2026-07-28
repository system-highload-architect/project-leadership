package main

import (
	"encoding/json"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{
			"version": "v2",
			"message": "Hello from service v2 (new version!)",
		})
	})
	log.Println("Service v2 starting on :8082")
	log.Fatal(http.ListenAndServe(":8082", nil))
}
