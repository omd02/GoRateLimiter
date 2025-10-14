package health

// HealthData represents the metrics needed by the Adaptive Throttler.
type HealthData struct {
	CPUUtilization float64 // e.g., 0.85 (85%)
	P95LatencyMs   float64 // e.g., 620.5 (ms)
	ErrorRate      float64 // e.g., 0.03 (3%)
}

// HealthSource is the interface for any component providing health data.
// This is the Adapter Pattern interface.
type HealthSource interface {
	// FetchMetrics retrieves the current health data from the source.
	FetchMetrics() (HealthData, error)
}
