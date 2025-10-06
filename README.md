![GitHub license](https://img.shields.io/badge/license-MIT-blue.svg)
# GoRateLimiter: Highly Efficient Hybrid API Rate Limiter 🛡️

GoRateLimiter is a high-performance, distributed API rate limiting solution built in Go and backed by Redis. It utilizes a sophisticated Hybrid Rate Limiting approach, combining the Token Bucket algorithm for instantaneous burst control with the Sliding Window Counter (SWC) for accurate long-term throughput management.

🌟 Features

    Hybrid Approach: Implements a dual-check mechanism (Token Bucket + SWC) for robust defense against both sudden traffic spikes (DoS) and boundary-based abuse.

    High Efficiency (O(1)): All critical checks run in constant time, utilizing simple Redis counters and timestamps for minimal latency.

    Concurrency Safe: Built for distributed environments, leveraging the atomic operations of Redis to ensure thread safety across multiple application instances.

    Go Middleware: Easy integration into any standard Go HTTP server.

    Minimal Memory Footprint: Avoids the memory overhead of the traditional Sliding Window Log by using only two integer counters per user/window.

📦 Getting Started

Prerequisites

You need the following installed on your machine:

    Go (1.18+): For running the application.

    Redis (6.0+): The required persistent store for all rate limit counters.

Installation & Setup

    Clone the repository:
    Bash

git clone https://github.com/YourUsername/GoRateLimiter.git
cd GoRateLimiter

Start Redis:
Ensure your Redis server is running, typically on localhost:6379. (If using Docker: docker run --name rate-redis -p 6379:6379 -d redis).

Run the Go application:
Bash

    go run main.go

    The server will start on port 8080.

⚙️ How It Works (The Hybrid Logic)

The core logic is implemented in the limiter/limiter.go file, where the Allow(identifier) method performs two mandatory, sequential checks:

1. Token Bucket Check (Burst Control)

    Goal: Limits the maximum number of requests a user can send in a very short period.

    Mechanism: When a request arrives, the system calculates how many tokens have refilled since the last check. If the current token count is less than 1, the request is DENIED instantly, preventing an overwhelming burst.

2. Sliding Window Counter (SWC) Check (Sustained Rate Control)

    Goal: Ensures the user's request rate is evenly distributed over a long period (e.g., 60 minutes) and prevents the fixed-window "boundary effect."

    Mechanism: The system fetches the counter for the Current Window and the Previous Window, then calculates a precise O(1) estimate:
    Estimated Count=(Prev. Count×Overlap Fraction)+Curr. Count

    If the Estimated Count exceeds the long-term limit, the request is DENIED.

A request is only ALLOWED if it PASSES BOTH the Token Bucket and the SWC check.

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
