const express = require('express');
const { MockAIService } = require('./mock-ai-service');
const app = express();
const PORT = 3456;

// Create mock AI service
const mockAI = new MockAIService();

// Simple middleware
app.use(express.json());

// VERY minimal request logger
app.use((req, res, next) => {
  console.log(`[Mock LLM] ${req.method} ${req.path}`);
  next();
});

// ONLY the essential endpoint
app.post('/v1/chat/completions', async (req, res) => {
  console.log(`[Mock LLM] Processing chat completion`);
  
  try {
    const { messages } = req.body;
    if (!messages) {
      return res.status(400).json({ error: 'Missing messages in request body' });
    }
    
    const response = await mockAI.chatCompletions(messages);
    res.json(response);
  } catch (error) {
    console.error(`[Mock LLM] Error: ${error.message}`);
    res.status(500).json({ error: error.message });
  }
});

// Catch all other requests
app.all('*', (req, res) => {
  console.log(`[Mock LLM] Unhandled: ${req.method} ${req.path}`);
  res.status(404).send('404 Not Found');
});

// Start the server
app.listen(PORT, () => {
  console.log(`Mock LLM Server running on http://localhost:${PORT}`);
});