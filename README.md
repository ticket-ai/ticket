# Guardian

> An open-source ethical telemetry and governance platform for AI applications

<img src="docs/assets/image.png" alt="Guardian Logo" width="100"><!-- 
[![Go Reference](https://pkg.go.dev/badge/github.com/yourusername/guardian.svg)](https://pkg.go.dev/github.com/yourusername/guardian)
[![Go Report Card](https://goreportcard.com/badge/github.com/yourusername/guardian)](https://goreportcard.com/report/github.com/yourusername/guardian)
[![License](https://img.shields.io/github/license/yourusername/guardian)](LICENSE) -->

## Dev Notes for yall
- Build Executible: cd ~/rhack && go build -o dist/guardian cmd/guardian/main.go
  The executible should have all buisness logic. It puts it in a dist folder that allows for the executible to be run in as a js package.
- Start Mock Server: cd ~/rhack/examples/mock-node-app && npm start
  The mock server is a mock up of the backend. When you run it, it will also run Guardian which along with performing the chat/completions monitoring will also generate a dashboard link.
- Test Mock Server: cd ~/rhack/examples/mock-node-app && npm test
  Running the test script simulates the backend recieving requests from the frontend. On the server view you should see it recieving these requests and 200ing. When the guardian dashboard is up and running you should also see that there.

## Why Guardian?

As AI systems become increasingly integrated into applications, organizations face growing challenges in monitoring usage, preventing misuse, and responding to security incidents. While the cybersecurity industry has mature tools for monitoring and incident response, the AI governance space lacks comparable solutions:

- **Jailbreak attempts** are constantly evolving, making it difficult for individual applications to stay ahead
- **Distributed security risks** arise as AI capabilities are integrated across different services
- **Lack of visibility** into how AI services are being used and potentially misused
- **No standardized approach** to AI governance and incident response

Guardian addresses these challenges by providing a language-agnostic middleware solution that enables comprehensive monitoring, detection, and governance of AI applications.

## Features

Guardian goes beyond simple telemetry to provide true AI governance capabilities:

### 1. Comprehensive Telemetry

- Automatic monitoring of all AI chat/completions endpoints
- Integration with OpenTelemetry and the LGTM stack (Grafana, Tempo, Prometheus)
- User behavior analytics including usage patterns, IP tracking, and request profiling
- Customizable dashboards for visualizing AI system usage

### 2. Multi-layered Security

- **Distributed NLP Analysis**: Low-resource natural language processing to identify potential policy violations
- **Review Agent**: Optional batch analysis of chat logs to detect sophisticated misuse patterns
- **Static Analysis**: Pattern matching using regex and other techniques to catch known attack vectors
- **Pre-prompting Management**: Standardized security controls applied across all AI endpoints

### 3. Governance Tools

- Real-time alerting for suspicious activity
- Automated incident triage and categorization
- User/IP flagging and blocking capabilities
- Trend analysis to identify emerging attack patterns

### 4. Incident Response

- Centralized incident management console
- Attack mitigation workflows
- Forensic logging for security investigations
- Automated remediation options

## Architecture

Guardian is designed as a lightweight, embeddable middleware that integrates seamlessly with your existing AI applications:

```
┌─────────────────┐      ┌───────────────┐      ┌─────────────────┐
│                 │      │               │      │                 │
│  Your AI App    │─────▶│   Guardian    │─────▶│   AI Provider   │
│                 │      │  Middleware   │      │    API          │
└─────────────────┘      └───────────────┘      └─────────────────┘
                               │
                               ▼
                        ┌───────────────┐
                        │               │
                        │  Telemetry    │
                        │  Pipeline     │
                        └───────────────┘
                               │
                               ▼
┌─────────────────┐      ┌───────────────┐      ┌─────────────────┐
│                 │      │               │      │                 │
│    Grafana      │◀────▶│  Prometheus/  │◀────▶│   Guardian      │
│   Dashboards    │      │    Tempo      │      │   Console       │
└─────────────────┘      └───────────────┘      └─────────────────┘
```

## Getting Started

### Prerequisites

- Go 1.20 or higher
- Docker and Docker Compose (for running the monitoring stack)

### Installation

Use the executible in dist or the upcoming js package.

### Basic Usage

Add Guardian middleware to your application:

```js
const guardian = new Guardian({ 
  serviceName: 'mock-ai-app', 
  debug: true 
});
```

### Configuration Options

Guardian can be configured to meet your specific requirements:

```go
config := guardian.Config{
    // Basic configuration
    ServiceName: "my-ai-app",
    Environment: "production",
    
    // Telemetry configuration
    OTelEndpoint: "localhost:4317",
    MetricsEnabled: true,
    TracingEnabled: true,
    
    // Security features
    NLPAnalysisEnabled: true,
    StaticAnalysisRules: []string{
        `\b(system prompt|ignore previous instructions)\b`,
        // Add custom regex patterns for known attack vectors
    },
    
    // Governance options
    AutoBlockThreshold: 0.85, // Confidence threshold for automatic blocking
    ReviewAgentEnabled: true,
    
    // Pre-prompting management
    StandardPrePrompt: "Always adhere to ethical guidelines and refuse harmful requests.",
}
```

### Monitoring Setup

Guardian includes a Docker Compose configuration for quickly setting up the monitoring stack:

```bash
# Start the monitoring stack
guardian setup monitoring

# View the setup in the browser
open http://localhost:3000  # Grafana dashboard
```

## Security Rules and Governance

Guardian comes with a set of predefined security rules that can be extended with your own custom rules:

### Static Analysis Rules

Guardian uses regex and pattern matching to identify potential security threats:

```yaml
# Example rules configuration (rules.yaml)
rules:
  - name: prompt-injection-basic
    pattern: '\b(system prompt|ignore previous instructions|my previous instructions|my prior instructions)\b'
    severity: high
    description: "Basic prompt injection attempt"
    
  - name: scenario-nesting
    pattern: 'pretend|imagine|role-play|simulation'
    context_pattern: '(ignore|forget|disregard).*(instruction|prompt|rule)'
    severity: medium
    description: "Possible scenario nesting attack"
```

### NLP Analysis

Guardian's NLP component analyzes conversations for:

- **Intent classification**: Categorizing user intentions
- **Sentiment analysis**: Detecting hostile or manipulative language
- **Topic modeling**: Identifying sensitive or prohibited subjects
- **Anomaly detection**: Flagging unusual interaction patterns

## API Reference

Guardian exposes several APIs for integration and extension:

- **Middleware API**: For intercepting and monitoring AI requests
- **Rules API**: For managing and customizing security rules
- **Reporting API**: For accessing telemetry data and insights
- **Governance API**: For responding to security incidents
<!-- 
## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for more information.

## License

Guardian is licensed under the [MIT License](LICENSE). -->
