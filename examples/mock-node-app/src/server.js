// Main Express application
const express = require('express');
const cors = require('cors');
const morgan = require('morgan');
const path = require('path');
const { v4: uuidv4 } = require('uuid');
const fetch = (...args) => import('node-fetch').then(({default: fetch}) => fetch(...args)); // Import fetch for Node.js

// Use absolute path to the guardian.js file - going up 3 levels from src directory
const guardianPath = path.resolve(__dirname, '../../../dist/guardian.js');
console.log('Guardian path:', guardianPath);
const Guardian = require(guardianPath);

// Initialize Guardian with minimal configuration
// This single line is all that's needed to enable monitoring!
const guardian = new Guardian({ 
  serviceName: 'mock-ai-app', 
  debug: true 
});

// Create Express app
const app = express();
const PORT = process.env.PORT || 3001; // Changed from 3000 to 3001 to avoid conflict with Grafana
const MOCK_LLM_URL = process.env.MOCK_LLM_URL || 'http://localhost:3456';

// Middleware
app.use(express.json());
app.use(cors());
app.use(morgan('dev'));

// Add request IDs for tracing
app.use((req, res, next) => {
  req.id = uuidv4();
  next();
});

// Health check endpoint
app.get('/health', (req, res) => {
  res.json({ status: 'ok', timestamp: new Date().toISOString() });
});

// Diagnostic endpoint to check connection to the mock LLM server
app.get('/check-llm', async (req, res) => {
  try {
    const health = await fetch(`${MOCK_LLM_URL}/health`);
    if (health.ok) {
      const data = await health.json();
      res.json({ 
        status: 'ok', 
        llmStatus: data,
        message: 'Mock LLM server is available'
      });
    } else {
      res.status(health.status).json({ 
        status: 'error',
        message: `Mock LLM server returned status ${health.status}`,
        details: await health.text()
      });
    }
  } catch (error) {
    console.error('Error checking LLM server:', error);
    res.status(500).json({ 
      status: 'error', 
      message: 'Failed to connect to mock LLM server',
      error: error.message
    });
  }
});

// Chat completions endpoint
app.post('/v1/chat/completions', async (req, res) => {
  try {
    const { messages, model = 'mock-gpt-3.5-turbo', temperature = 0.7 } = req.body;
    
    if (!messages || !Array.isArray(messages)) {
      return res.status(400).json({ error: 'Messages are required and must be an array' });
    }
    
    // Log request to help with debugging
    console.log(`Sending chat completion request to ${MOCK_LLM_URL}/v1/chat/completions`);
    
    // Create a fallback response in case the LLM server fails
    const fallbackResponse = {
      id: `chatcmpl-${Date.now()}`,
      object: 'chat.completion',
      created: Math.floor(Date.now() / 1000),
      model: model,
      choices: [
        {
          index: 0,
          message: {
            role: 'assistant',
            content: "I'm an AI assistant here to help you. (Fallback response)"
          },
          finish_reason: 'stop'
        }
      ],
      usage: {
        prompt_tokens: 10,
        completion_tokens: 10,
        total_tokens: 20
      }
    };
    
    try {
      // Make an actual HTTP request to the mock LLM server
      const response = await fetch(`${MOCK_LLM_URL}/v1/chat/completions`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'User-Id': req.headers['user-id'] || 'unknown',
          'X-Forwarded-For': req.headers['x-forwarded-for'] || req.ip
        },
        body: JSON.stringify({ messages, model, temperature })
      });
      
      // Check if response is ok before trying to parse JSON
      if (!response.ok) {
        const errorText = await response.text();
        console.log(`Error response from mock LLM: ${response.status} - ${errorText}`);
        
        // Instead of returning an error, use the fallback response
        console.log('Using fallback response due to LLM server error');
        return res.json(fallbackResponse);
      }
      
      // Only try to parse JSON if the response is ok
      const data = await response.json();
      res.json(data);
    } catch (fetchError) {
      console.error('Network error with LLM server:', fetchError);
      
      // Use fallback response for any network errors
      console.log('Using fallback response due to network error');
      return res.json(fallbackResponse);
    }
  } catch (error) {
    console.error('Error in chat completions:', error);
    res.status(500).json({ error: 'An error occurred processing your request' });
  }
});

// Text completions endpoint
app.post('/v1/completions', async (req, res) => {
  try {
    const { prompt, model = 'mock-text-davinci-003', temperature = 0.7 } = req.body;
    
    if (!prompt) {
      return res.status(400).json({ error: 'Prompt is required' });
    }
    
    // Log request to help with debugging
    console.log(`Sending text completion request to ${MOCK_LLM_URL}/v1/completions`);
    
    // Create a fallback response in case the LLM server fails
    const fallbackResponse = {
      id: `cmpl-${Date.now()}`,
      object: 'text.completion',
      created: Math.floor(Date.now() / 1000),
      model: model,
      choices: [
        {
          text: "This is a fallback response from the mock API.",
          index: 0,
          finish_reason: 'stop'
        }
      ],
      usage: {
        prompt_tokens: 10,
        completion_tokens: 10,
        total_tokens: 20
      }
    };
    
    try {
      // Make an actual HTTP request to the mock LLM server
      const response = await fetch(`${MOCK_LLM_URL}/v1/completions`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'User-Id': req.headers['user-id'] || 'unknown',
          'X-Forwarded-For': req.headers['x-forwarded-for'] || req.ip
        },
        body: JSON.stringify({ prompt, model, temperature })
      });
      
      // Check if response is ok before trying to parse JSON
      if (!response.ok) {
        const errorText = await response.text();
        console.log(`Error response from mock LLM: ${response.status} - ${errorText}`);
        
        // Instead of returning an error, use the fallback response
        console.log('Using fallback response due to LLM server error');
        return res.json(fallbackResponse);
      }
      
      // Only try to parse JSON if the response is ok
      const data = await response.json();
      res.json(data);
    } catch (fetchError) {
      console.error('Network error with LLM server:', fetchError);
      
      // Use fallback response for any network errors
      console.log('Using fallback response due to network error');
      return res.json(fallbackResponse);
    }
  } catch (error) {
    console.error('Error in text completions:', error);
    res.status(500).json({ error: 'An error occurred processing your request' });
  }
});

// Admin endpoint to get all flagged users
app.get('/admin/flagged-users', (req, res) => {
  // In the real implementation, this would query the Guardian API
  // For now, we'll just return an empty array
  res.json({ flaggedUsers: [] });
});

// Admin endpoint to get all blocked IPs
app.get('/admin/blocked-ips', (req, res) => {
  // In the real implementation, this would query the Guardian API
  // For now, we'll just return an empty array
  res.json({ blockedIPs: [] });
});

// Start the server
app.listen(PORT, () => {
  console.log(`Mock AI app running on http://localhost:${PORT}`);
  console.log(`Using Mock LLM API at ${MOCK_LLM_URL}`);
});