package health

import (
	"fmt"
	"math/rand"
	"time"
)

// SimulatedSource simulates fetching real-time data with random variance.
type SimulatedSource struct {
	// Empty for now, but could hold configuration later.
}

// NewSimulatedSource creates a new instance.
func NewSimulatedSource() *SimulatedSource {
	// Ensure the random number generator is seeded once
	rand.Seed(time.Now().UnixNano())
	return &SimulatedSource{}
}

// FetchMetrics implements HealthSource by generating synthetic data.
func (s *SimulatedSource) FetchMetrics() (HealthData, error) {
	// Introduce slight random variance around a base load
	cpuBase := 0.75
	latencyBase := 600.0 // Base P95 latency of 600ms
	errorBase := 0.02    // Base error rate of 2%

	// Add random noise (+/- 5 percentage points, +/- 50ms, +/- 0.5%)
	cpu := cpuBase + (rand.Float64()*0.1 - 0.05)
	latency := latencyBase + (rand.Float64()*100 - 50)
	errors := errorBase + (rand.Float64()*0.01 - 0.005)

	// Apply bounds
	if cpu < 0.1 {
		cpu = 0.1
	}
	if latency < 1.0 {
		latency = 1.0
	}
	if errors < 0.001 {
		errors = 0.001
	}

	data := HealthData{
		CPUUtilization: cpu,
		P95LatencyMs:   latency,
		ErrorRate:      errors,
	}

	// Print the fetched data for real-time monitoring
	fmt.Printf("[HEALTH] CPU: %.2f%%, P95 Latency: %.0fms, Errors: %.2f%%\n",
		data.CPUUtilization*100, data.P95LatencyMs, data.ErrorRate*100)

	return data, nil
}
