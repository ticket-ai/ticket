# Guardian AI

An ethical telemetry and governance platform for AI applications.

Guardian provides a zero-configuration middleware for monitoring and governing AI API calls in Node.js applications. It helps protect your applications by monitoring outgoing AI requests, applying security rules, and providing logging and telemetry.

## Installation

```bash
npm install guardian-ai
```

During installation, the appropriate binary for your platform will be automatically downloaded.

## Quick Start

```javascript
const Guardian = require('guardian-ai');

// Initialize Guardian with default settings
const guardian = new Guardian();

// Guardian will automatically intercept AI API calls from your application
// like OpenAI, Anthropic, and other LLM providers
```

## Configuration

```javascript
const guardian = new Guardian({
  serviceName: 'my-ai-app',
  environment: 'production',
  prePrompt: 'Always adhere to ethical guidelines and refuse harmful requests.',
  debug: true,
  autoStart: true,
  rules: [
    // Custom rules can be added here
    {
      name: "Custom Rule Example",
      pattern: "(?i)\\b(sensitive|proprietary)\\b",
      severity: "high",
      description: "Detects sensitive information requests"
    }
  ]
});
```

You can also define rules in a JSON file at the root of your project named `guardian_rules.json`:

```json
{
  "rules": [
    {
      "name": "Sensitive Information Detection",
      "pattern": "(?i)\\b(password|credit card|ssn)\\b",
      "severity": "high",
      "description": "Detects requests for sensitive personal information"
    }
  ]
}
```

## API

### new Guardian(config)

Creates a new Guardian instance with optional configuration.

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| serviceName | string | 'guardian-app' | The name of your service |
| environment | string | 'development' | Environment (development, staging, production) |
| prePrompt | string | '' | Standard pre-prompt to apply to all requests |
| debug | boolean | false | Enable debug mode |
| autoStart | boolean | true | Automatically start Guardian on initialization |
| rules | array | [...] | Custom rules for monitoring |

### guardian.start()

Starts the Guardian proxy server if it's not already running.

```javascript
await guardian.start();
```

### guardian.stop()

Stops the Guardian proxy server and restores original network behavior.

```javascript
guardian.stop();
```

### guardian.middleware()

Creates Express.js middleware for Guardian.

```javascript
const express = require('express');
const app = express();

const guardian = new Guardian();
app.use(guardian.middleware());
```

## Rule Format

Guardian uses regex patterns to detect potentially problematic requests:

```json
{
  "name": "Rule Name",
  "pattern": "regex pattern",
  "severity": "low|medium|high",
  "description": "Description of the rule and what it detects"
}
```

## Platform Support

Guardian supports:
- Windows (x64, arm64)
- macOS (Intel, Apple Silicon)
- Linux (x64, arm64)

## License

MIT