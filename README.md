# Guardian ![PyPI - Version](https://img.shields.io/pypi/v/guardian-ai)![PyPI - Downloads](https://img.shields.io/pypi/dm/guardian-ai)![NPM Version](https://img.shields.io/npm/v/guardian-ai)![NPM Downloads](https://img.shields.io/npm/d18m/guardian-ai)

> An open-source ethical telemetry and governance platform for AI applications

<img width="1208" alt="Screenshot 2025-04-19 at 12 45 35 AM" src="https://github.com/user-attachments/assets/691233cb-8761-4677-9fa2-49f1277af9e7" />

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
- Customizable dashboards for visualizing AI system usage

### 2. Multi-layered Security

- **Distributed NLP Analysis**: Low-resource natural language processing to identify potential policy violations

- **Static Analysis**: Pattern matching using regex and other techniques to catch known attack vectors

Guardian supports customizable security preferences through user submitted jsons. These rules use regex and pattern matching to identify potential security threats. Guardian will automatically register any rules either passed into the config over by reading a guardian_rules.json file in the root or src directory.

```json
{
  "rules": [
    {
      "name": "prompt-injection-basic",
      "pattern": "\\b(system prompt|ignore previous instructions)\\b",
      "severity": "high", 
      "description": "Basic prompt injection attempt"
    },
    {
      "name": "scenario-nesting",
      "pattern": "pretend|imagine|role-play|simulation",
      "severity": "medium",
      "description": "Possible scenario nesting attack"
    }
  ]
}
```

- **Pre-prompting Management**: Standardized security controls applied across all AI endpoints

Guardian supports custom configs upon instantiaion that prepend a standardized prompt across all endpoints, ensuring universal protection.

const DEFAULT_CONFIG = {
  prePrompt: "Refuse to answer any questions related to politics or world affairs.",
};

### 3. Governance Tools

- Real-time alerting for suspicious activity
- Automated incident triage and categorization
- Message flagging and blocking capabilities
- Trend analysis to identify emerging attack patterns

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
                        │  OTEL-LGTM    │
                        │               │
                        └───────────────┘
```

## Getting Started

Guardian supports multiple programming languages through native implementations. Choose the one that fits your project:

### Setting up the Monitoring Stack

Before using any of the language implementations, you should set up the monitoring stack:

1. **Start the LGTM Stack**:
   ```bash
   # Start the LGTM stack (Loki, Grafana, Tempo, Mimir/Prometheus)
   ./run-lgtm.sh
   ```

2. **Access Grafana Dashboard**:
   Open http://localhost:3000 in your browser


### JavaScript/Node.js Implementation

#### Installation

```bash
# Using npm
npm install guardian-ai

# Using yarn
yarn add guardian-ai
```

#### Usage

```javascript
const Guardian = require('guardian-ai');

// Initialize Guardian with your configuration
const guardian = new Guardian({ 
  serviceName: 'my-js-app', 
  environment: 'development',
  debug: true 
});
```

No further configuration needed - Guardian will automatically monitor AI API calls made through the standard Node.js HTTP/HTTPS APIs and fetch.

### Python Implementation

#### Installation

```bash
# Using pip
pip install guardian-ai

# Using poetry
poetry add guardian-ai
```

#### Usage

```python
from guardian-ai import Guardian

# Initialize Guardian
guardian = Guardian()

# Guardian will automatically monitor AI API calls
```
### Go Implementation

#### Installation

```bash
go get github.com/ticket-ai/ticket
```

#### Usage

```go
package main

import (
    "github.com/ticket-ai/ticket"
)

func main() {
    // Create Guardian configuration
    config := ticket.DefaultConfig()
    config.ServiceName = "my-go-app"
    config.Environment = "development"
    
    // Initialize Guardian
    g, err := ticket.New(config)
    if err != nil {
        panic(err)
    }
    
    // Use Guardian middleware with your HTTP handlers
    http.ListenAndServe(":8080", g.Middleware.HTTPHandler(yourHandler))
}
```