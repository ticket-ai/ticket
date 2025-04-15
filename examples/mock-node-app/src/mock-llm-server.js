const express = require('express');
const cors = require('cors');
const { MockAIService } = require('./mock-ai-service');

const app = express();
const PORT = process.env.MOCK_LLM_PORT || 3456;

// Create instance of the mock AI service
const mockAI = new MockAIService();

// Middleware
app.use(express.json());
app.use(cors());

// Add request logging middleware
app.use((req, res, next) => {
  console.log(`[Mock LLM] ${req.method} ${req.path} received with body:`, JSON.stringify(req.body));
  next();
});

// Health check endpoint
app.get('/health', (req, res) => {
  res.json({ status: 'ok' });
});

// Simple test endpoint for debugging
app.get('/test', (req, res) => {
  res.json({ message: 'Mock LLM server is working' });
});

// Chat completions endpoint
app.post('/v1/chat/completions', async (req, res) => {
  try {
    console.log(`[Mock LLM] Processing chat completion with messages:`, JSON.stringify(req.body.messages));
    const { messages } = req.body;
    
    if (!messages || !Array.isArray(messages)) {
      console.log('[Mock LLM] Invalid messages data:', req.body);
      return res.status(400).json({ error: 'Invalid messages format' });
    }
    
    const response = await mockAI.chatCompletions(messages);
    console.log(`[Mock LLM] Chat completion response:`, JSON.stringify(response));
    res.json(response);
  } catch (error) {
    console.error('[Mock LLM] Error in chat completions:', error);
    res.status(500).json({ error: 'Server error', details: error.message });
  }
});

// Text completions endpoint
app.post('/v1/completions', async (req, res) => {
  try {
    console.log(`[Mock LLM] Processing text completion with prompt:`, req.body.prompt);
    const { prompt } = req.body;
    
    if (!prompt) {
      console.log('[Mock LLM] Invalid prompt data:', req.body);
      return res.status(400).json({ error: 'Invalid prompt format' });
    }
    
    const response = await mockAI.textCompletions(prompt);
    console.log(`[Mock LLM] Text completion response:`, JSON.stringify(response));
    res.json(response);
  } catch (error) {
    console.error('[Mock LLM] Error in text completions:', error);
    res.status(500).json({ error: 'Server error', details: error.message });
  }
});

// Catch-all route for debugging missing endpoints
app.use((req, res) => {
  console.log(`[Mock LLM] ⚠️ Unhandled route: ${req.method} ${req.url}`);
  res.status(404).send(`Endpoint not found: ${req.method} ${req.url}`);
});

// Start the server
app.listen(PORT, () => {
  console.log(`Mock LLM API server running on http://localhost:${PORT}`);
  console.log(`Available endpoints:`);
  console.log(`- GET /health - Health check`);
  console.log(`- GET /test - Test endpoint`);
  console.log(`- POST /v1/chat/completions - Chat completions API`);
  console.log(`- POST /v1/completions - Text completions API`);
});