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

// Health check endpoint
app.get('/health', (req, res) => {
  res.json({ status: 'ok' });
});

// Chat completions endpoint
app.post('/v1/chat/completions', async (req, res) => {
  try {
    const { messages } = req.body;
    const response = await mockAI.chatCompletions(messages);
    console.log(`Mock LLM server handled chat completion request`);
    res.json(response);
  } catch (error) {
    console.error('Error in mock chat completions:', error);
    res.status(500).json({ error: 'Server error' });
  }
});

// Text completions endpoint
app.post('/v1/completions', async (req, res) => {
  try {
    const { prompt } = req.body;
    const response = await mockAI.textCompletions(prompt);
    console.log(`Mock LLM server handled text completion request`);
    res.json(response);
  } catch (error) {
    console.error('Error in mock text completions:', error);
    res.status(500).json({ error: 'Server error' });
  }
});

// Start the server
app.listen(PORT, () => {
  console.log(`Mock LLM API server running on http://localhost:${PORT}`);
});