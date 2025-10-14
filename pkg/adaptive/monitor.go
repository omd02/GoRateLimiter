package adaptive

import (
	"GoRateLimiter/pkg/health" // **<-- CHANGE 'your_module' TO YOUR ACTUAL MODULE NAME**
	"log"
	"time"
)

// Monitor manages the background routine that adjusts the rate limiter.
type Monitor struct {
	Limiter  *AdaptiveLimiter
	Source   health.HealthSource
	Interval time.Duration
}

// NewMonitor creates a new instance of the Adaptive Monitor.
func NewMonitor(limiter *AdaptiveLimiter, source health.HealthSource, interval time.Duration) *Monitor {
	return &Monitor{
		Limiter:  limiter,
		Source:   source,
		Interval: interval,
	}
}

// StartMonitoring runs the check-and-adjust loop in a goroutine.
func (m *Monitor) StartMonitoring() {
	ticker := time.NewTicker(m.Interval)
	log.Println("Adaptive Rate Monitor started.")

	for range ticker.C {
		// 1. Fetch the data using the Adapter interface
		healthData, err := m.Source.FetchMetrics()
		if err != nil {
			log.Printf("Error fetching health metrics: %v. Sticking to current rate.", err)
			continue
		}

		// 2. Calculate the new adaptive factor
		newFactor := calculateFactor(healthData)

		// 3. Update the Limiter
		m.Limiter.UpdateFactor(newFactor)
	}
}

// =========================================================================
// The Core Adaptive Logic: This determines the throttling factor (F)
// =========================================================================

// calculateFactor determines the throttling factor (0.0 to 1.0) based on health.
func calculateFactor(data health.HealthData) float64 {
	// Define SLO/SLA targets
	const TargetCPU = 0.70       // We want to keep CPU below 70%
	const TargetLatency = 500.0  // We want to keep P95 latency below 500ms
	const TargetErrorRate = 0.01 // We want to keep Error Rate below 1% (0.01)

	// Calculate a factor for each metric: Factor = Target / Current

	// 1. CPU Factor
	cpuFactor := TargetCPU / data.CPUUtilization

	// 2. Latency Factor
	latencyFactor := TargetLatency / data.P95LatencyMs

	// 3. Error Factor
	errorFactor := TargetErrorRate / data.ErrorRate

	// Find the minimum factor (the most stressed metric dictates the throttle)
	factor := min(cpuFactor, latencyFactor, errorFactor)

	// Apply bounds:
	// Cap the maximum factor at 1.0 (no throttling)
	if factor > 1.0 {
		return 1.0
	}
	// Set a floor (e.g., 0.1) to prevent the rate from dropping to absolute zero
	if factor < 0.1 {
		return 0.1
	}

	return factor
}

func min(a, b, c float64) float64 {
	m := a
	if b < m {
		m = b
	}
	if c < m {
		m = c
	}
	return m
}
