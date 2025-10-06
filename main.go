package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"GoRateLimiter/limiter"

	"github.com/go-redis/redis/v8"
)

// Global Limiter Instance
var rateLimiter *limiter.Limiter

func main() {
	ctx := context.Background()

	// 1. Initialize Redis Client
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	// Ping Redis to ensure the connection is working
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Could not connect to Redis: %v", err)
	}
	fmt.Println("Successfully connected to Redis.")

	// 2. Initialize the Hybrid Rate Limiter
	rateLimiter = limiter.NewLimiter(rdb, ctx)

	// 3. Define Handlers and Middleware
	// All incoming requests will go through the rateLimitMiddleware first.
	http.Handle("/api/data", rateLimitMiddleware(http.HandlerFunc(dataHandler)))
	http.Handle("/status", http.HandlerFunc(statusHandler))

	// 4. Start the Server
	fmt.Println("Server starting on :8080. Try hitting http://localhost:8080/api/data")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// rateLimitMiddleware is the critical function that wraps our core handlers.
func rateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Get the identifier (e.g., IP address)
		// For a simple example, we use the remote IP. In production, you'd use a user ID or API key.
		identifier := r.RemoteAddr

		// 2. Execute the hybrid rate limiter check
		if rateLimiter.Allow(identifier) {
			// Request is ALLOWED: Pass control to the next handler (dataHandler)
			next.ServeHTTP(w, r)
			return
		}

		// 3. Request is DENIED: Respond with HTTP 429
		w.WriteHeader(http.StatusTooManyRequests) // 429 Too Many Requests
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"error": "Rate limit exceeded. Try again later."}`)
	})
}

// dataHandler is the core business logic handler, only reached if the request is allowed.
func dataHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"message": "Welcome! Your request was processed."}`)
}

// statusHandler is for simple server health checks.
func statusHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Server is healthy and rate limiting is active.")
}
