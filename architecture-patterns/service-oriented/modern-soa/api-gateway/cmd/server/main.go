package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

func main() {
	orderURL, _ := url.Parse("http://localhost:8081")
	customerURL, _ := url.Parse("http://localhost:8082")

	orderProxy := httputil.NewSingleHostReverseProxy(orderURL)
	customerProxy := httputil.NewSingleHostReverseProxy(customerURL)

	mux := http.NewServeMux()
	mux.HandleFunc("/orders/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[Gateway] Routing to Order Service: %s %s", r.Method, r.URL.Path)
		orderProxy.ServeHTTP(w, r)
	})
	mux.HandleFunc("/customers/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[Gateway] Routing to Customer Service: %s %s", r.Method, r.URL.Path)
		customerProxy.ServeHTTP(w, r)
	})

	log.Println("API Gateway starting on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
