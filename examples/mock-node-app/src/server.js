const express = require('express');
const axios = require('axios');
const path = require('path');

const guardianPath = path.resolve(__dirname, '../../../dist/guardian.js');
console.log('Guardian path:', guardianPath);
const Guardian = require(guardianPath);
const guardian = new Guardian({ 
  serviceName: 'js-app', 
  debug: true,
  stripeAuthToken: process.env.stripe,
});

const app = express();
const PORT = 3001;
const MOCK_LLM_URL = 'http://localhost:3456';

app.use(express.json());

app.post('/api/chat', async (req, res) => {  
  try {
    const llmURL = `${MOCK_LLM_URL}/v1/chat/completions`;    
    const response = await axios({
      method: 'post',
      url: llmURL,
      data: req.body,
      headers: {
        'Content-Type': 'application/json'
      }
    });
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

app.listen(PORT, () => {
  console.log(`Server running on http://localhost:${PORT}`);
  console.log(`Using Mock LLM at ${MOCK_LLM_URL}`);
});
