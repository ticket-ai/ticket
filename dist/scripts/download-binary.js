/**
 * Binary Download Script
 * 
 * This script downloads the appropriate Guardian binary for the user's platform
 * during npm installation. It's executed automatically as part of the post-install
 * process to ensure the correct binary is available for the user's system.
 */

const https = require('https');
const http = require('http');
const os = require('os');
const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

// Configuration
const RELEASE_VERSION = '0.1.0';
const BINARY_NAME = 'guardian';
const BASE_URL = 'https://github.com/rohanadwankar/guardian/releases/download';

// Map of platform and architecture to binary names
const PLATFORM_MAPPING = {
  'darwin-x64': 'guardian-darwin-amd64',
  'darwin-arm64': 'guardian-darwin-arm64',
  'linux-x64': 'guardian-linux-amd64',
  'linux-arm64': 'guardian-linux-arm64',
  'win32-x64': 'guardian-windows-amd64.exe'
};

async function downloadBinary() {
  try {
    console.log('Guardian: Downloading binary...');
    
    // Determine platform and architecture
    const platform = os.platform();
    const arch = os.arch();
    const platformKey = `${platform}-${arch}`;
    
    // Get binary name for this platform
    const binaryName = PLATFORM_MAPPING[platformKey];
    if (!binaryName) {
      console.error(`Guardian: Unsupported platform ${platformKey}`);
      console.error('Guardian: Supported platforms: ' + Object.keys(PLATFORM_MAPPING).join(', '));
      process.exit(1);
    }
    
    // Determine download URL and local path
    const url = `${BASE_URL}/v${RELEASE_VERSION}/${binaryName}`;
    const distDir = path.join(__dirname, '..');
    const outputDir = path.join(distDir, `${platform}-${arch}`);
    const outputPath = path.join(outputDir, platform === 'win32' ? `${BINARY_NAME}.exe` : BINARY_NAME);
    
    // Create directory if it doesn't exist
    if (!fs.existsSync(outputDir)) {
      fs.mkdirSync(outputDir, { recursive: true });
    }
    
    console.log(`Guardian: Downloading from ${url}`);
    console.log(`Guardian: Saving to ${outputPath}`);
    
    // Download the file
    await downloadFile(url, outputPath);
    
    // Make the binary executable on Unix platforms
    if (platform !== 'win32') {
      fs.chmodSync(outputPath, 0o755); // rwxr-xr-x
    }
    
    console.log('Guardian: Binary download complete');
    console.log(`Guardian: Binary saved to ${outputPath}`);
  } catch (error) {
    console.error('Guardian: Failed to download binary:', error.message);
    process.exit(1);
  }
}

function downloadFile(url, outputPath) {
  return new Promise((resolve, reject) => {
    // Choose http or https module based on URL
    const client = url.startsWith('https') ? https : http;
    
    const request = client.get(url, (response) => {
      // Handle redirects
      if (response.statusCode === 301 || response.statusCode === 302) {
        const redirectUrl = response.headers.location;
        return downloadFile(redirectUrl, outputPath).then(resolve).catch(reject);
      }
      
      // Check if the request was successful
      if (response.statusCode !== 200) {
        return reject(new Error(`Failed to download binary (status code: ${response.statusCode})`));
      }
      
      // Create write stream and pipe the response
      const file = fs.createWriteStream(outputPath);
      response.pipe(file);
      
      file.on('finish', () => {
        file.close();
        resolve();
      });
    });
    
    request.on('error', (err) => {
      fs.unlink(outputPath, () => {}); // Delete partial file
      reject(err);
    });
  });
}

// Run the download
downloadBinary().catch(error => {
  console.error('Guardian: Error downloading binary:', error);
  process.exit(1);
});