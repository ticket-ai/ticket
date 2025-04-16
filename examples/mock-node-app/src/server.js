const express = require('express');
const morgan = require('morgan');
const axios = require('axios');
const path = require('path');

// Import Guardian with minimal configuration
const guardianPath = path.resolve(__dirname, '../../../dist/guardian.js');
console.log('Guardian path:', guardianPath);
const Guardian = require(guardianPath);
const guardian = new Guardian({ 
  serviceName: 'mock-ai-app', 
  debug: true 
});

// Express setup
const app = express();
const PORT = 3001;
const MOCK_LLM_URL = 'http://localhost:3456';

// Middleware
app.use(express.json());
app.use(morgan('dev'));

// API Chat Endpoint (what client-test.js calls)
app.post('/api/chat', async (req, res) => {
  console.log('[Guardian] Routing HTTP AI API call through Guardian');
  
  try {
    // Make direct HTTP request to mock LLM server
    const llmURL = `${MOCK_LLM_URL}/v1/chat/completions`;
    console.log(`Forwarding request to: ${llmURL}`);
    
    const response = await axios({
      method: 'post',
      url: llmURL,
      data: req.body,
      headers: {
        'Content-Type': 'application/json'
      }
    });
    
    console.log('Successfully received response from LLM');
    res.json(response.data);
  } catch (error) {
    console.error('Error calling mock LLM:');
    console.error(`Status: ${error.response?.status || 'unknown'}`);
    console.error(`Message: ${error.message}`);
    
    // Return error to client
    res.status(error.response?.status || 500).json({
      error: 'Failed to get response from LLM service',
      details: error.message
    });
  }
});

// Basic admin endpoints
app.get('/admin/flagged-users', (req, res) => res.json({ flaggedUsers: [] }));
app.get('/admin/blocked-ips', (req, res) => res.json({ blockedIPs: [] }));

// Start server
app.listen(PORT, () => {
  console.log(`Server running on http://localhost:${PORT}`);
  console.log(`Using Mock LLM at ${MOCK_LLM_URL}`);
});