package health

import (
	"context"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

// --- PromQL Query Constants ---
const (
	// Example query for the 5-minute average CPU utilization across all cores (0-1 range)
	CPU_QUERY = "1 - avg(rate(node_cpu_seconds_total{mode=\"idle\"}[5m]))"

	// Example query for the P95 latency of HTTP requests (in seconds)
	P95_LATENCY_QUERY = "histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))"

	// Example query for the 5xx error rate (5xx errors / total requests)
	ERROR_RATE_QUERY = "sum(rate(http_requests_total{status_code=~\"5..\"}[5m])) / sum(rate(http_requests_total[5m]))"
)

// PrometheusSource implements the HealthSource interface
type PrometheusSource struct {
	Client v1.API // The Prometheus V1 API client
}

// NewPrometheusSource initializes the Prometheus client connection.
func NewPrometheusSource(promURL string) (*PrometheusSource, error) {
	client, err := api.NewClient(api.Config{
		Address: promURL,
	})
	if err != nil {
		return nil, fmt.Errorf("error creating Prometheus client: %w", err)
	}

	return &PrometheusSource{
		Client: v1.NewAPI(client),
	}, nil
}

// FetchMetrics executes PromQL queries and converts the results into HealthData.
func (p *PrometheusSource) FetchMetrics() (HealthData, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	now := time.Now()
	data := HealthData{}

	// Helper function to execute a query and retrieve the single float value
	queryAndExtract := func(query string) (float64, error) {
		result, warnings, err := p.Client.Query(ctx, query, now)

		if err != nil {
			return 0, fmt.Errorf("prometheus query error for %s: %w", query, err)
		}
		if len(warnings) > 0 {
			// Log warnings, but proceed
		}

		// Expecting a vector result with exactly one element (instant vector)
		if v, ok := result.(model.Vector); ok && len(v) > 0 {
			// Convert Prom value to standard float64
			return float64(v[0].Value), nil
		}

		// Return 0 if no data is found, which should trigger a conservative factor
		return 0, nil
	}

	// 1. Fetch CPU Utilization
	cpu, err := queryAndExtract(CPU_QUERY)
	if err != nil {
		// Log or handle specific errors; for now, we'll return a conservative factor
		// In production, you might return the last known good value
		return data, err
	}
	data.CPUUtilization = cpu * 100.0 // Convert 0-1 range to 0-100%

	// 2. Fetch P95 Latency (comes in seconds, we need milliseconds)
	latencySec, err := queryAndExtract(P95_LATENCY_QUERY)
	if err != nil {
		return data, err
	}
	data.P95LatencyMs = latencySec * 1000.0 // Convert seconds to milliseconds

	// 3. Fetch Error Rate (comes in 0-1 range, keep it as a ratio)
	errorRate, err := queryAndExtract(ERROR_RATE_QUERY)
	if err != nil {
		return data, err
	}
	data.ErrorRate = errorRate * 100.0 // Convert to percentage

	return data, nil
}
