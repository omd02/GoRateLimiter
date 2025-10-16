package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"GoRateLimiter/pkg/adaptive"
	"GoRateLimiter/pkg/health" // Ensure this is the correct module path
)

// Global Limiter Instance
var adaptiveLimiter *adaptive.AdaptiveLimiter

// NOTE: Replace this with the actual URL of your Prometheus server
const PROMETHEUS_URL = "http://localhost:9090"

func main() {
	// --- 1. CONFIGURATION ---
	const BaseRPS = 100.0
	const MonitorInterval = 5 * time.Second

	log.Println("Initializing Adaptive Rate Limiter...")
	adaptiveLimiter = adaptive.NewAdaptiveLimiter(BaseRPS)

	// --- 2. START THE ADAPTIVE MONITOR (Using REAL Prometheus Data) ---

	// ⚠️ 1. Initialize the Prometheus Client
	realSource, err := health.NewPrometheusSource(PROMETHEUS_URL)
	if err != nil {
		log.Fatalf("Fatal: Could not initialize Prometheus Source: %v", err)
	}

	// ⚠️ 2. Start the Monitor with the REAL source
	monitor := adaptive.NewMonitor(adaptiveLimiter, realSource, MonitorInterval)

	go monitor.StartMonitoring()
	log.Printf("Adaptive Monitor started, fetching metrics from: %s", PROMETHEUS_URL)

	// --- 3. START THE SERVER ---
	http.Handle("/api/data", rateLimitMiddleware(http.HandlerFunc(dataHandler)))
	http.Handle("/status", http.HandlerFunc(statusHandler))

	fmt.Println("Server starting on :8080. The rate limit is now dynamically adjusting based on real Prometheus metrics.")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// ... rest of rateLimitMiddleware and handler functions ...
// / rateLimitMiddleware is the critical function that wraps our core handlers.
func rateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Get the identifier (e.g., IP address)
		// ⚠️ FIX: Commented out for now to resolve 'declared and not used' error.
		// TODO: Re-enable and use 'identifier' once adaptiveLimiter is updated for per-client limits.
		// identifier := r.RemoteAddr

		// 2. Execute the global adaptive rate limiter check
		if adaptiveLimiter.Allow() {
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
	fmt.Fprintf(w, "Server is healthy and adaptive rate limiting is active.")
}
