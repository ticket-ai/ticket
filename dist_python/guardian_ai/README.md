# Guardian AI for Python

> Ethical AI monitoring and governance platform for Python applications

## Overview

Guardian AI is a Python package that provides zero-configuration monitoring and governance for AI API calls in Python applications. This package is the Python equivalent of the JavaScript Guardian package, with similar functionality and API.

## Features

- **Automatic monitoring** of AI chat/completions endpoints
- **Zero configuration** setup for most use cases
- **Integration with common Python frameworks** like FastAPI
- **Cross-platform support** with bundled platform-specific binaries
- **Security rules** to detect and prevent misuse of AI systems

## Installation

```bash
pip install guardian-ai
```

## Basic Usage

### Simple Usage

```python
from guardian_ai import Guardian

# Initialize Guardian with default configuration
guardian = Guardian()

# Now all AI API calls using requests will be monitored
import requests

response = requests.post(
    "https://api.openai.com/v1/chat/completions",
    json={
        "model": "gpt-3.5-turbo",
        "messages": [
            {"role": "system", "content": "You are a helpful assistant."},
            {"role": "user", "content": "Tell me about Guardian AI."}
        ]
    },
    headers={"Authorization": f"Bearer {api_key}"}
)
```

### Configuration Options

```python
from guardian_ai import Guardian

# Initialize with custom configuration
guardian = Guardian({
    "service_name": "my-ai-app",
    "environment": "production",
    "debug": True,
    "pre_prompt": "Always adhere to ethical guidelines and refuse harmful requests."
})
```

### Using with FastAPI

```python
from fastapi import FastAPI
from guardian_ai import Guardian

app = FastAPI()
guardian = Guardian({
    "service_name": "my-fastapi-app",
    "debug": True
})

# Add the guardian middleware
app.add_middleware(guardian.create_fastapi_middleware())

@app.post("/api/chat")
async def chat(request: dict):
    # Your code to call AI service
    # Guardian will automatically monitor and protect this endpoint
    pass
```

## Command Line Usage

Guardian AI can also be used directly from the command line:

```bash
# Start Guardian AI proxy server
guardian-ai --port 8080 --service "my-service" --debug

# Use with a custom rules file
guardian-ai --rules ./guardian_rules.json
```

## Security Rules

Guardian includes default security rules, but you can define your own in a `guardian_rules.json` file:

```json
{
  "rules": [
    {
      "name": "Prompt Injection",
      "pattern": "(?i)ignore (previous|prior) instructions",
      "severity": "medium",
      "description": "Attempt to make the model ignore previous instructions."
    },
    {
      "name": "Role Play Bypass",
      "pattern": "(?i)\\b(pretend|imagine|role-play|simulation).+?(ignore|forget|disregard).+?(instruction|prompt|rule)",
      "severity": "medium",
      "description": "Attempt to use role-playing to bypass rules."
    }
  ]
}
```

## License

This package is licensed under the same terms as the Guardian project.