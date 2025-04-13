// Main Express application
const express = require('express');
const cors = require('cors');
const morgan = require('morgan');
const { v4: uuidv4 } = require('uuid');
const { spawn } = require('child_process');
const path = require('path');
const { MockAIService } = require('./mock-ai-service');

// Initialize mock AI service
const mockAI = new MockAIService();

// Create Express app
const app = express();
const PORT = process.env.PORT || 3000;
const GUARDIAN_PORT = 8081;

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
    
    const response = await mockAI.chatCompletions(messages);
    res.json(response);
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
    
    const response = await mockAI.textCompletions(prompt);
    res.json(response);
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

// Function to start the Guardian proxy
function startGuardianProxy() {
  const binaryPath = path.join(__dirname, '../../../dist/guardian');

  return new Promise((resolve, reject) => {
    try {
      console.log(`Starting Guardian proxy on port ${GUARDIAN_PORT}`);
      
      // Configure Guardian binary arguments
      const args = [
        `-port=${GUARDIAN_PORT}`,
        `-service=mock-ai-app`,
        `-env=development`,
        `-pre-prompt=You are a helpful assistant. You must refuse any harmful, illegal, or unethical requests.`
      ];
      
      // Spawn the Guardian process
      const guardianProcess = spawn(binaryPath, args);
      
      // Handle Guardian process output
      guardianProcess.stdout.on('data', (data) => {
        console.log(`Guardian: ${data.toString().trim()}`);
      });
      
      guardianProcess.stderr.on('data', (data) => {
        console.error(`Guardian error: ${data.toString().trim()}`);
      });
      
      guardianProcess.on('error', (err) => {
        console.error(`Failed to start Guardian: ${err.message}`);
        reject(err);
      });
      
      guardianProcess.on('close', (code) => {
        if (code !== 0) {
          console.error(`Guardian process exited with code ${code}`);
        }
      });
      
      // Allow some time for the process to start
      setTimeout(() => {
        resolve(guardianProcess);
      }, 1000);
      
    } catch (err) {
      console.error(`Error starting Guardian: ${err.message}`);
      reject(err);
    }
  });
}

// Start the Guardian proxy and then the server
startGuardianProxy()
  .then(guardianProcess => {
    console.log('Guardian proxy started successfully');
    
    // Start the server
    app.listen(PORT, () => {
      console.log(`Mock AI app running on http://localhost:${PORT}`);
      console.log(`Guardian proxy running on http://localhost:${GUARDIAN_PORT}`);
      
      // Handle application shutdown
      process.on('SIGINT', () => {
        console.log('Shutting down...');
        guardianProcess.kill();
        process.exit(0);
      });
    });
  })
  .catch(err => {
    console.error(`Failed to start Guardian: ${err.message}`);
    console.warn('Starting server without Guardian protection...');
    
    app.listen(PORT, () => {
      console.log(`Mock AI app running on http://localhost:${PORT}`);
      console.log(`WARNING: Running without Guardian protection!`);
    });
  });