/**
 * Guardian Node.js Integration Module
 * 
 * This module provides seamless integration between Node.js applications and the Guardian proxy.
 * It automatically launches the Guardian binary as a child process and configures the application
 * to route AI API calls through the Guardian proxy for monitoring and governance.
 */

const { spawn } = require('child_process');
const path = require('path');
const fs = require('fs');
const http = require('http');
const os = require('os');

// Default configuration
const DEFAULT_CONFIG = {
  port: 8080,
  serviceName: 'nodejs-app',
  environment: 'development',
  prePrompt: '',
  blockThreshold: 0.85,
  enableNLP: true,
  autoStart: true,
  binaryPath: undefined, // Will be auto-detected
  debug: false
};

class Guardian {
  constructor(config = {}) {
    this.config = { ...DEFAULT_CONFIG, ...config };
    this.process = null;
    this.ready = false;
    this.originalFetch = global.fetch;
    
    // Auto-detect binary path if not specified
    if (!this.config.binaryPath) {
      this.config.binaryPath = this._detectBinaryPath();
    }
    
    // Setup proxy if autoStart is enabled
    if (this.config.autoStart) {
      this.start();
    }
    
    // Patch global fetch to route through Guardian
    this._patchFetch();
    
    // Handle process exit
    process.on('exit', () => {
      this.stop();
    });
  }
  
  /**
   * Start the Guardian proxy server
   * @returns {Promise} Resolves when the server is ready
   */
  start() {
    return new Promise((resolve, reject) => {
      if (this.process) {
        this.log('Guardian already running');
        return resolve();
      }
      
      this.log(`Starting Guardian proxy on port ${this.config.port}`);
      
      // Check if binary exists
      if (!fs.existsSync(this.config.binaryPath)) {
        return reject(new Error(`Guardian binary not found at ${this.config.binaryPath}`));
      }
      
      // Prepare arguments for the binary
      const args = [
        `-port=${this.config.port}`,
        `-service=${this.config.serviceName}`,
        `-env=${this.config.environment}`,
        `-threshold=${this.config.blockThreshold}`,
        `-nlp=${this.config.enableNLP}`
      ];
      
      if (this.config.prePrompt) {
        args.push(`-pre-prompt=${this.config.prePrompt}`);
      }
      
      // Launch the Guardian process
      this.process = spawn(this.config.binaryPath, args, {
        stdio: this.config.debug ? 'inherit' : 'pipe'
      });
      
      // Handle process events
      this.process.on('error', (err) => {
        this.log(`Guardian process error: ${err.message}`);
        reject(err);
      });
      
      this.process.on('exit', (code) => {
        this.log(`Guardian process exited with code ${code}`);
        this.process = null;
        this.ready = false;
      });
      
      if (!this.config.debug) {
        // Log stdout/stderr if not in debug mode (debug mode inherits stdio)
        this.process.stdout?.on('data', (data) => {
          this.log(`Guardian: ${data.toString().trim()}`);
        });
        
        this.process.stderr?.on('data', (data) => {
          this.log(`Guardian error: ${data.toString().trim()}`);
        });
      }
      
      // Wait for server to be ready
      this._waitForReady().then(() => {
        this.ready = true;
        this.log('Guardian proxy is ready');
        resolve();
      }).catch(reject);
    });
  }
  
  /**
   * Stop the Guardian proxy server
   */
  stop() {
    if (this.process) {
      this.log('Stopping Guardian proxy');
      this.process.kill();
      this.process = null;
      this.ready = false;
    }
  }
  
  /**
   * Restore original fetch behavior
   */
  restore() {
    if (this.originalFetch) {
      global.fetch = this.originalFetch;
      this.log('Restored original fetch');
    }
  }
  
  /**
   * Wait for the Guardian server to be ready
   * @private
   */
  _waitForReady() {
    return new Promise((resolve, reject) => {
      const maxAttempts = 30;
      let attempts = 0;
      
      const checkHealth = () => {
        attempts++;
        
        http.get(`http://localhost:${this.config.port}/_guardian/health`, (res) => {
          if (res.statusCode === 200) {
            return resolve();
          }
          
          if (attempts >= maxAttempts) {
            return reject(new Error('Guardian server failed to start'));
          }
          
          setTimeout(checkHealth, 100);
        }).on('error', () => {
          if (attempts >= maxAttempts) {
            return reject(new Error('Guardian server failed to start'));
          }
          
          setTimeout(checkHealth, 100);
        });
      };
      
      // Start checking after a small delay
      setTimeout(checkHealth, 100);
    });
  }
  
  /**
   * Patch global fetch to route AI API calls through Guardian
   * @private
   */
  _patchFetch() {
    const self = this;
    const originalFetch = global.fetch;
    
    global.fetch = async function guardianFetch(url, options = {}) {
      // Check if this is an AI API call that should be routed through Guardian
      if (self._isAIEndpoint(url) && self.ready) {
        // Modify the URL to go through the Guardian proxy
        const proxyUrl = new URL(url);
        proxyUrl.host = `localhost:${self.config.port}`;
        
        self.log(`Routing AI API call through Guardian: ${url}`);
        return originalFetch(proxyUrl.toString(), options);
      }
      
      // Not an AI endpoint or Guardian not ready, use original fetch
      return originalFetch(url, options);
    };
  }
  
  /**
   * Detect if a URL is an AI endpoint that should be monitored
   * @private
   */
  _isAIEndpoint(url) {
    const urlStr = url.toString();
    return (
      urlStr.includes('/completions') ||
      urlStr.includes('/chat/completions') ||
      urlStr.includes('/generate') ||
      urlStr.includes('/v1/engines') ||
      urlStr.includes('/v1/chat')
    );
  }
  
  /**
   * Auto-detect the Guardian binary path based on the current platform
   * @private
   */
  _detectBinaryPath() {
    // Start with the module directory
    const baseDir = path.join(__dirname, '../dist');
    
    // Determine platform-specific binary name
    let binaryName = 'guardian';
    if (os.platform() === 'win32') {
      binaryName += '.exe';
    }
    
    // Check for platform/arch specific binary
    const platformArch = `${os.platform()}-${os.arch()}`;
    let binaryPath = path.join(baseDir, platformArch, binaryName);
    
    if (fs.existsSync(binaryPath)) {
      return binaryPath;
    }
    
    // Fall back to default location
    binaryPath = path.join(baseDir, binaryName);
    if (fs.existsSync(binaryPath)) {
      return binaryPath;
    }
    
    // If not found, return the default path anyway and we'll error later
    return binaryPath;
  }
  
  /**
   * Log a message if debug is enabled
   * @private
   */
  log(message) {
    if (this.config.debug) {
      console.log(`[Guardian] ${message}`);
    }
  }
}

module.exports = Guardian;