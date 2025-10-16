![GitHub license](https://img.shields.io/badge/license-MIT-blue.svg)
# Go Rate Limiter 🛡️

**Go Rate Limiter** is a high-performance, self-adjusting traffic control module built in **Go**. Unlike traditional static rate limiters, this module dynamically adjusts the maximum allowed requests per second (RPS) based on the service's **real-time operational health** (CPU, Latency, Errors).

This proactive throttling ensures maximum throughput during normal operation and guarantees **system stability** by actively shedding load before a service or its dependencies become overwhelmed.

## 🌟 Core Features

* **Dynamic Adaptation:** The core throttling factor is updated every few seconds based on live health metrics.
* **Decoupled Health Source (Adapter Pattern):** Uses the `HealthSource` interface, allowing easy swapping between the **Prometheus Source** (Production) and the **Simulated Source** (Testing).
* **High Performance:** Built on the concurrency-safe `golang.org/x/time/rate` package for efficient Token Bucket enforcement.
* **Built for Production:** Integrates directly with **Prometheus**, the industry-standard monitoring solution.
* **Extensible Architecture:** Designed for future integration of **per-client adaptive limits** via API keys or JWTs.

***

## 🚀 Getting Started

### Prerequisites

You need the following installed:

* **Go (1.20+):** For running the application.
* **Prometheus:** A running Prometheus server accessible to your application (required for production mode).

### Installation & Setup

1.  **Clone the Repository:**
    ```bash
    git clone [Your Repository URL Here]
    cd go-adaptive-limiter
    ```
2.  **Resolve Dependencies:**
    ```bash
    go mod tidy
    ```
3.  **Configure Prometheus URL:**
    Open `main.go` and set the correct endpoint for your Prometheus server:
    ```go
    const PROMETHEUS_URL = "http://your-prometheus-server:9090" 
    ```
4.  **Run the Application (Production Mode):**
    ```bash
    go run .
    ```
    The server will start on port `8080`, and the monitor will immediately begin querying Prometheus to set the initial adaptive rate limit.

***

## ⚙️ Core Architecture

The system is cleanly separated into three core components, emphasizing modularity and testability through the **Adapter Pattern**.

| Component | Package | Responsibility |
| :--- | :--- | :--- |
| **Adaptive Limiter** | `pkg/adaptive` | The central throttling mechanism. Enforces the limit calculated by the Monitor using the Token Bucket algorithm. |
| **Health Monitor** | `pkg/adaptive` | The decision engine. Periodically calls the `HealthSource`, runs the risk assessment formula, and applies the resulting **Throttling Factor** to the Limiter. |
| **Health Source** | `pkg/health` | **Adapter** interface for metric retrieval. Implemented by `real.go` (Prometheus) and `simulated.go` (Test). |

### The Adaptive Logic

The Monitor calculates a **Throttling Factor** (a floating point value between 0.1 and 1.0) based on three key health metrics:

1.  **CPU Utilization**
2.  **P95 Request Latency**
3.  **5xx Error Rate**

The formula is designed to quickly reduce the allowed RPS when *any* metric exceeds its healthy threshold. This ensures the applied rate limit is always sufficient to maintain a healthy service state.

$$
\text{New RPS} = \text{BaseRPS} \times \text{Factor}
$$

### Health Source Decoupling

The module achieves production readiness by isolating the data source behind the `HealthSource` interface:

```go
// pkg/health/interface.go
type HealthData struct {
    CPUUtilization float64 // 0-100%
    P95LatencyMs   float64 // Milliseconds
    ErrorRate      float64 // 0-100%
}

type HealthSource interface {
    // FetchMetrics retrieves the latest status metrics from the source.
    FetchMetrics() (HealthData, error)
}

🛠️ Configuration

The rate limit parameters are defined in the limiter/limiter.go file within the NewLimiter constructor.
Parameter	Algorithm	Default Value	Description
BucketCapacity	Token Bucket	10	The maximum size of the instantaneous burst allowed.
RefillRate	Token Bucket	6 * time.Second	The time it takes to replenish one token (10 tokens/minute).
SWCLimit	SWC	100	The maximum number of requests allowed within the full SWCWindow.
SWCWindow	SWC	60 * time.Minute	The duration of the rolling window (e.g., 1 hour).

🧪 Testing

You can test the hybrid functionality by sending requests to the API endpoint:

Endpoint: http://localhost:8080/api/data

    Test Burst (Token Bucket): Hit the endpoint 12 times quickly. The server should respond with 429 Too Many Requests (the Token Bucket is empty) after the 10th request.

    Test Refill: Wait 7 seconds (longer than the 6-second RefillRate) and send 2 more requests. They should be allowed, demonstrating the Token Bucket refilling correctly.

🤝 Contributing

Contributions are welcome! Please feel free to open issues or submit pull requests.

📄 License

This project is licensed under the MIT License - see the LICENSE file for details.
