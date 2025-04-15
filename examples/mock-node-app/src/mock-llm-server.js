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

// Add detailed request logging middleware
app.use((req, res, next) => {
  console.log(`[Mock LLM] ${req.method} ${req.path} received`);
  if (Object.keys(req.body).length > 0) {
    console.log(`[Mock LLM] Request body:`, JSON.stringify(req.body, null, 2));
  }
  next();
});

// Root endpoint for basic testing
app.get('/', (req, res) => {
  res.send(`
    <html>
      <head>
        <title>Mock LLM API Server</title>
        <style>
          body { font-family: Arial, sans-serif; max-width: 800px; margin: 0 auto; padding: 20px; }
          h1 { color: #333; }
          pre { background: #f5f5f5; padding: 10px; border-radius: 5px; }
          .endpoint { margin-bottom: 20px; }
        </style>
      </head>
      <body>
        <h1>Mock LLM API Server</h1>
        <p>This is a mock server that simulates AI API endpoints.</p>
        
        <div class="endpoint">
          <h2>Available Endpoints:</h2>
          <ul>
            <li><strong>GET /health</strong> - Health check endpoint</li>
            <li><strong>POST /v1/chat/completions</strong> - Chat completions API</li>
            <li><strong>POST /v1/completions</strong> - Text completions API</li>
          </ul>
        </div>
        
        <div class="endpoint">
          <h2>Example Chat Completion Request:</h2>
          <pre>
POST /v1/chat/completions
Content-Type: application/json

{
  "messages": [
    {"role": "system", "content": "You are a helpful assistant."},
    {"role": "user", "content": "Hello, how are you?"}
  ],
  "model": "mock-gpt-3.5-turbo"
}
          </pre>
        </div>
      </body>
    </html>
  `);
});

// Health check endpoint
app.get('/health', (req, res) => {
  res.json({ status: 'ok', timestamp: new Date().toISOString() });
});

// Implement express API error handler
function handleApiError(res, error, defaultMessage = 'Server error') {
  console.error('[Mock LLM] Error:', error);
  const statusCode = error.statusCode || 500;
  const errorMessage = error.message || defaultMessage;
  res.status(statusCode).json({ error: errorMessage });
}

// Simple chat completion fallback handler that always succeeds
function generateFallbackChatResponse(messages) {
  const userMessage = messages?.find(m => m.role === 'user')?.content || 'No message provided';
  console.log(`[Mock LLM] Generating fallback response for message: "${userMessage.substring(0, 50)}..."`);
  
  return {
    id: `chatcmpl-${Date.now()}`,
    object: 'chat.completion',
    created: Math.floor(Date.now() / 1000),
    model: 'mock-gpt-3.5-turbo',
    choices: [
      {
        index: 0,
        message: {
          role: 'assistant',
          content: "I'm an AI assistant here to help you with your questions."
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
}

// Simple text completion fallback handler that always succeeds
function generateFallbackTextResponse(prompt) {
  return {
    id: `cmpl-${Date.now()}`,
    object: 'text.completion',
    created: Math.floor(Date.now() / 1000),
    model: 'mock-text-davinci-003',
    choices: [
      {
        text: "This is a response from the mock LLM API.",
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
}

// Chat completions endpoint - case sensitive route with fallback
app.post('/v1/chat/completions', async (req, res) => {
  console.log('[Mock LLM] Processing chat completion request');
  try {
    const { messages } = req.body;
    
    if (!messages || !Array.isArray(messages)) {
      return res.status(400).json({ error: 'Messages are required and must be an array' });
    }
    
    try {
      const response = await mockAI.chatCompletions(messages);
      return res.json(response);
    } catch (serviceError) {
      console.error('[Mock LLM] Service error:', serviceError);
      // Fallback response instead of failing
      const fallbackResponse = generateFallbackChatResponse(messages);
      return res.json(fallbackResponse);
    }
  } catch (error) {
    handleApiError(res, error, 'Error processing chat completion');
  }
});

// Text completions endpoint - case sensitive route with fallback
app.post('/v1/completions', async (req, res) => {
  console.log('[Mock LLM] Processing text completion request');
  try {
    const { prompt } = req.body;
    
    if (!prompt) {
      return res.status(400).json({ error: 'Prompt is required' });
    }
    
    try {
      const response = await mockAI.textCompletions(prompt);
      return res.json(response);
    } catch (serviceError) {
      console.error('[Mock LLM] Service error:', serviceError);
      // Fallback response instead of failing
      const fallbackResponse = generateFallbackTextResponse(prompt);
      return res.json(fallbackResponse);
    }
  } catch (error) {
    handleApiError(res, error, 'Error processing text completion');
  }
});

// Additional routes for case insensitivity
app.post('/V1/chat/completions', (req, res) => {
  console.log('[Mock LLM] Redirecting from uppercase route to lowercase route');
  app._router.handle(req, res, () => {
    req.url = '/v1/chat/completions';
    app._router.handle(req, res);
  });
});

app.post('/V1/completions', (req, res) => {
  console.log('[Mock LLM] Redirecting from uppercase route to lowercase route');
  app._router.handle(req, res, () => {
    req.url = '/v1/completions';
    app._router.handle(req, res);
  });
});

// Catch-all route handler for debugging
app.use((req, res) => {
  console.log(`[Mock LLM] ⚠️ Unhandled route: ${req.method} ${req.path}`);
  res.status(404).send(`
    <html>
      <head>
        <title>404 - Endpoint Not Found</title>
        <style>
          body { font-family: Arial, sans-serif; max-width: 800px; margin: 0 auto; padding: 20px; }
          h1 { color: #d32f2f; }
          pre { background: #f5f5f5; padding: 10px; border-radius: 5px; }
        </style>
      </head>
      <body>
        <h1>404 - Endpoint Not Found</h1>
        <p>The requested endpoint <code>${req.method} ${req.path}</code> was not found.</p>
        
        <h3>Available Endpoints:</h3>
        <ul>
          <li><code>GET /</code> - Home page</li>
          <li><code>GET /health</code> - Health check</li>
          <li><code>POST /v1/chat/completions</code> - Chat completions API</li>
          <li><code>POST /v1/completions</code> - Text completions API</li>
        </ul>
      </body>
    </html>
  `);
});

// Start the server
app.listen(PORT, () => {
  console.log(`\n--------------------------------------------------------`);
  console.log(`Mock LLM API Server running on http://localhost:${PORT}`);
  console.log(`--------------------------------------------------------`);
  console.log(`Available endpoints:`);
  console.log(`- GET / - Home page with documentation`);
  console.log(`- GET /health - Health check endpoint`);
  console.log(`- POST /v1/chat/completions - Chat completions API`);
  console.log(`- POST /v1/completions - Text completions API`);
  console.log(`--------------------------------------------------------\n`);
});