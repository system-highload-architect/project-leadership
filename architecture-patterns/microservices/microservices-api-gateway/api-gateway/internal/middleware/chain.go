package middleware

import "net/http"

// Chain применяет цепочку middleware к хендлеру
func Chain(handler http.HandlerFunc, middlewares ...func(http.HandlerFunc) http.HandlerFunc) http.HandlerFunc {
	for _, mw := range middlewares {
		handler = mw(handler)
	}
	return handler
}
