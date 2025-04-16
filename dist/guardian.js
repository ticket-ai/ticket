/**
 * Guardian - Ethical AI Monitoring and Governance Platform
 * 
 * This module provides a zero-configuration middleware for monitoring and 
 * governing AI API calls in Node.js applications.
 */

const { spawn } = require('child_process');
const path = require('path');
const fs = require('fs');
const http = require('http');
const https = require('https');
const net = require('net');
const os = require('os');

// Default configuration
const DEFAULT_CONFIG = {
  serviceName: 'guardian-app',
  environment: 'development',
  prePrompt: "Always adhere to ethical guidelines and refuse harmful requests.",
  debug: false,
  autoStart: true,
};

class Guardian {
  constructor(config = {}) {
    this.config = { ...DEFAULT_CONFIG, ...config };
    this.port = null;
    this.process = null;
    this.ready = false;
    this.originalFetch = global.fetch;
    this.originalHttpRequest = http.request;
    this.originalHttpsRequest = https.request;
    
    // Find the binary path
    this.binaryPath = this._findBinaryPath();
    
    // Automatically start Guardian if autoStart is enabled
    if (this.config.autoStart) {
      this.start();
    }
  }

  /**
   * Start the Guardian proxy server
   * @returns {Promise} Resolves when the server is ready
   */
  start() {
    return new Promise(async (resolve, reject) => {
      try {
        if (this.process) {
          this.log('Guardian already running');
          return resolve();
        }
        
        // Find a free port to use
        this.port = await this._findFreePort();
        this.log(`Starting Guardian proxy on port ${this.port}`);
        
        // Check if binary exists
        if (!fs.existsSync(this.binaryPath)) {
          return reject(new Error(`Guardian binary not found at ${this.binaryPath}`));
        }
        
        // Prepare arguments for the Guardian binary
        const args = [
          `-port=${this.port}`,
          `-service=${this.config.serviceName}`,
          `-env=${this.config.environment}`
        ];
        
        if (this.config.prePrompt) {
          args.push(`-pre-prompt=${this.config.prePrompt}`);
        }
        
        if (this.config.debug) {
          args.push('-debug=true');
        }
        
        // Spawn the Guardian process
        this.process = spawn(this.binaryPath, args, {
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
        
        if (!this.config.debug && this.process.stdout && this.process.stderr) {
          // Log stdout/stderr if not in debug mode
          this.process.stdout.on('data', (data) => {
            this.log(`Guardian: ${data.toString().trim()}`);
          });
          
          this.process.stderr.on('data', (data) => {
            this.log(`Guardian: ${data.toString().trim()}`);
          });
        }
        
        // Patch network APIs to route through Guardian with transparent forwarding
        this._patchNetworkAPIs();
        
        // Wait for server to be ready
        await this._waitForReady();
        this.ready = true;
        this.log('Guardian proxy is ready');
        resolve();
      } catch (err) {
        this.log(`Error starting Guardian: ${err.message}`);
        reject(err);
      }
    });
  }
  
  /**
   * Stop the Guardian proxy server and restore original network behavior
   */
  stop() {
    if (this.process) {
      this.log('Stopping Guardian proxy');
      this.process.kill();
      this.process = null;
      this.ready = false;
    }
    
    this._restoreNetworkAPIs();
  }
  
  /**
   * Find a free port to use for the Guardian proxy
   * @private
   */
  _findFreePort() {
    return new Promise((resolve, reject) => {
      const server = net.createServer();
      server.on('error', reject);
      server.listen(0, () => {
        const port = server.address().port;
        server.close(() => resolve(port));
      });
    });
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
        
        http.get(`http://localhost:${this.port}/_guardian/health`, (res) => {
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
   * Patch network APIs to route AI requests through Guardian
   * @private
   */
  _patchNetworkAPIs() {
    this._patchFetch();
    this._patchHttpRequest();
  }
  
  /**
   * Restore original network APIs
   * @private
   */
  _restoreNetworkAPIs() {
    if (this.originalFetch) {
      global.fetch = this.originalFetch;
    }
    
    if (this.originalHttpRequest) {
      http.request = this.originalHttpRequest;
    }
    
    if (this.originalHttpsRequest) {
      https.request = this.originalHttpsRequest;
    }
    
    this.log('Restored original network APIs');
  }
  
  /**
   * Patch the fetch API to route through Guardian
   * @private
   */
  _patchFetch() {
    if (!global.fetch) return;
    
    const self = this;
    const originalFetch = global.fetch;
    
    global.fetch = async function guardianFetch(url, options = {}) {
      // Check if this is an AI API call that should be routed through Guardian
      if (self._isAIEndpoint(url)) {
        try {
          // Ensure we preserve the original destination for internal forwarding
          const originalUrl = url.toString();
          
          // Create modified headers to include original destination
          const modifiedOptions = { ...options };
          modifiedOptions.headers = { ...(options.headers || {}) };
          
          // Add the original destination header if we're redirecting
          modifiedOptions.headers['X-Guardian-Original-Destination'] = originalUrl;
          
          self.log(`Routing AI API call through Guardian: ${url}`);
          
          // Route to Guardian proxy
          const proxyUrl = new URL('http://localhost:' + self.port);
          proxyUrl.pathname = new URL(url).pathname;
          proxyUrl.search = new URL(url).search;
          
          return originalFetch(proxyUrl.toString(), modifiedOptions);
        } catch (err) {
          self.log(`Error routing through Guardian: ${err.message}`);
          return originalFetch(url, options);
        }
      }
      
      // Not an AI endpoint, use original fetch
      return originalFetch(url, options);
    };
  }
  
  /**
   * Patch the HTTP/HTTPS request APIs to route through Guardian
   * @private
   */
  _patchHttpRequest() {
    const self = this;
    
    // Patch HTTP
    http.request = function guardianHttpRequest(options, callback) {
      if (self._isAIEndpointFromOptions(options)) {
        self.log(`Routing HTTP AI API call through Guardian`);
        
        // Save original options for forwarding
        let originalDestination = '';
        
        // Extract original destination
        if (typeof options === 'string') {
          originalDestination = options;
        } else if (options instanceof URL) {
          originalDestination = options.toString();
        } else {
          const protocol = options.protocol || 'http:';
          const host = options.host || options.hostname || 'localhost';
          const port = options.port ? `:${options.port}` : '';
          const path = options.path || '/';
          originalDestination = `${protocol}//${host}${port}${path}`;
        }
        
        self.log(`Original destination: ${originalDestination}`);
        
        // Modify options to go through Guardian proxy
        if (typeof options === 'string' || options instanceof URL) {
          const originalUrl = options;
          options = new URL(options.toString());
          options.host = 'localhost';
          options.hostname = 'localhost';
          options.port = self.port;
          options.headers = options.headers || {};
          options.headers['X-Guardian-Original-Destination'] = originalUrl.toString();
        } else {
          // Clone options to avoid modifying the original
          const originalOpts = { ...options };
          
          // Add header to the new options
          options = { ...options };
          if (!options.headers) options.headers = {};
          options.headers['X-Guardian-Original-Destination'] = originalDestination;
          
          // Modify to go through Guardian
          options.host = 'localhost';
          options.hostname = 'localhost';
          options.port = self.port;
        }
        
        self.log(`Forwarding through Guardian proxy: localhost:${self.port}`);
      }
      
      return self.originalHttpRequest.call(http, options, callback);
    };
    
    // Similar approach for HTTPS
    https.request = function guardianHttpsRequest(options, callback) {
      if (self._isAIEndpointFromOptions(options)) {
        self.log(`Routing HTTPS AI API call through Guardian`);
        
        // Save original options for forwarding
        let originalDestination = '';
        
        // Extract original destination
        if (typeof options === 'string') {
          originalDestination = options;
        } else if (options instanceof URL) {
          originalDestination = options.toString();
        } else {
          const protocol = 'https:';
          const host = options.host || options.hostname || 'localhost';
          const port = options.port ? `:${options.port}` : '';
          const path = options.path || '/';
          originalDestination = `${protocol}//${host}${port}${path}`;
        }
        
        self.log(`Original destination: ${originalDestination}`);
        
        // For HTTPS, we have to convert to using HTTP to the Guardian proxy
        let newOptions;
        if (typeof options === 'string' || options instanceof URL) {
          const url = new URL(options.toString());
          newOptions = {
            protocol: 'http:',
            host: 'localhost',
            hostname: 'localhost',
            port: self.port,
            path: url.pathname + url.search,
            method: 'POST',
            headers: {
              'Content-Type': 'application/json',
              'X-Guardian-Original-Destination': originalDestination
            }
          };
        } else {
          // Clone headers and add original destination
          const headers = { ...(options.headers || {}) };
          headers['X-Guardian-Original-Destination'] = originalDestination;
          
          newOptions = {
            ...options,
            protocol: 'http:',
            host: 'localhost',
            hostname: 'localhost',
            port: self.port,
            headers
          };
        }
        
        self.log(`Forwarding through Guardian proxy: localhost:${self.port}`);
        return self.originalHttpRequest.call(http, newOptions, callback);
      }
      
      return self.originalHttpsRequest.call(https, options, callback);
    };
  }
  
  /**
   * Check if a URL is for an AI endpoint that should be monitored
   * @private
   */
  _isAIEndpoint(url) {
    const urlStr = url.toString().toLowerCase();
    return (
      urlStr.includes('/completions') ||
      urlStr.includes('/chat/completions') ||
      urlStr.includes('/generate') ||
      urlStr.includes('/v1/engines') ||
      urlStr.includes('/v1/chat')
    );
  }
  
  /**
   * Check if HTTP request options point to an AI endpoint
   * @private
   */
  _isAIEndpointFromOptions(options) {
    if (typeof options === 'string') {
      return this._isAIEndpoint(options);
    }
    
    if (options instanceof URL) {
      return this._isAIEndpoint(options.toString());
    }
    
    // Check path in options object
    if (options.path) {
      const path = options.path.toLowerCase();
      return (
        path.includes('/completions') ||
        path.includes('/chat/completions') ||
        path.includes('/generate') ||
        path.includes('/v1/engines') ||
        path.includes('/v1/chat')
      );
    }
    
    return false;
  }
  
  /**
   * Find the Guardian binary path
   * @private
   */
  _findBinaryPath() {
    // Start with the module directory
    const baseDir = path.dirname(__filename);
    
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
    
    // Fall back to project root
    binaryPath = path.join(baseDir, '..', binaryName);
    if (fs.existsSync(binaryPath)) {
      return binaryPath;
    }
    
    // If not found, return the default path anyway
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
  
  /**
   * Create a middleware function for Express.js applications
   * This is an alternative to the API patching approach
   */
  middleware() {
    const self = this;
    
    return function(req, res, next) {
      if (!self.ready || !self._isAIEndpointFromOptions({ path: req.path })) {
        return next();
      }
      
      self.log(`Express middleware handling AI endpoint: ${req.path}`);
      
      // Store the original destination
      const protocol = req.protocol;
      const host = req.headers.host;
      const path = req.originalUrl || req.url;
      const originalDestination = `${protocol}://${host}${path}`;
      req.headers['X-Guardian-Original-Destination'] = originalDestination;
      
      // Proxy the request to Guardian
      const proxyReq = http.request({
        method: req.method,
        host: 'localhost',
        port: self.port,
        path: req.url,
        headers: req.headers
      }, proxyRes => {
        // Copy status code
        res.status(proxyRes.statusCode);
        
        // Copy headers
        Object.keys(proxyRes.headers).forEach(key => {
          res.set(key, proxyRes.headers[key]);
        });
        
        // Stream response data
        proxyRes.pipe(res);
      });
      
      // Handle errors
      proxyReq.on('error', error => {
        self.log(`Error in Guardian proxy: ${error.message}`);
        next(error);
      });
      
      // Send request body if present
      if (req.body) {
        proxyReq.write(JSON.stringify(req.body));
      }
      
      proxyReq.end();
    };
  }
}

module.exports = Guardian;