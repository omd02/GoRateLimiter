package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"GoRateLimiter/pkg/adaptive" // Assuming pkg/adaptive is where the new limiter is
	"GoRateLimiter/pkg/health"   // Assuming pkg/health is where the new simulated source is
)

// Global Limiter Instance
var adaptiveLimiter *adaptive.AdaptiveLimiter

func main() {

	const BaseRPS = 100.0                   // Max theoretical requests per second (RPS)
	const MonitorInterval = 5 * time.Second // How often the monitor checks health

	// The Adaptive Limiter is in-memory and does not need Redis.
	log.Println("Initializing Adaptive Rate Limiter...")
	adaptiveLimiter = adaptive.NewAdaptiveLimiter(BaseRPS)

	// We use the Simulated Health Source for testing.
	simulatedSource := health.NewSimulatedSource()
	monitor := adaptive.NewMonitor(adaptiveLimiter, simulatedSource, MonitorInterval)

	// Start the background monitoring routine.
	go monitor.StartMonitoring()
	log.Println("Adaptive Monitor started in background.")

	// All incoming requests will go through the rateLimitMiddleware now using the dynamic limit.
	http.Handle("/api/data", rateLimitMiddleware(http.HandlerFunc(dataHandler)))
	http.Handle("/status", http.HandlerFunc(statusHandler))

	fmt.Println("Server starting on :8080. Check console for dynamic rate limit changes.")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// rateLimitMiddleware is the critical function that wraps our core handlers.
func rateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if adaptiveLimiter.Allow() {
			// Request is ALLOWED: Pass control to the next handler
			next.ServeHTTP(w, r)
			return
		}

		// The `Retry-After` header is a good practice for adaptive limiting.
		w.Header().Set("Retry-After", "5")        // Suggest client wait 5 seconds
		w.WriteHeader(http.StatusTooManyRequests) // 429 Too Many Requests
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"error": "Rate limit exceeded. System load is high."}`)
	})
}

// dataHandler is the core business logic handler, only reached if the request is allowed.
func dataHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"message": "Welcome! Request processed under the dynamic rate limit."}`)
}

// statusHandler is for simple server health checks.
func statusHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Server is healthy and adaptive rate limiting is active.")
}
