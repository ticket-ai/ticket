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
const PORT = process.env.PORT || 3000;
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

// Chat completions endpoint
app.post('/v1/chat/completions', async (req, res) => {
  try {
    const { messages, model = 'mock-gpt-3.5-turbo', temperature = 0.7 } = req.body;
    
    if (!messages || !Array.isArray(messages)) {
      return res.status(400).json({ error: 'Messages are required and must be an array' });
    }
    
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
    
    const data = await response.json();
    res.json(data);
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
    
    const data = await response.json();
    res.json(data);
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