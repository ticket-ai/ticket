#!/usr/bin/env node

/**
 * download-binary.js
 * Downloads the appropriate Guardian binary for the current platform
 */

const https = require('https');
const fs = require('fs');
const path = require('path');
const os = require('os');
const { execSync } = require('child_process');

// Define configuration
const packageJson = require('../package.json');
const GUARDIAN_VERSION = packageJson.version || '0.1.0';
const GITHUB_REPO = 'rohanadwankar/guardian'; // Update this with your actual GitHub username/repo
const BASE_URL = `https://github.com/${GITHUB_REPO}/releases/download`;
const RETRY_DELAY = 1000;
const MAX_RETRIES = 3;

// Determine platform and architecture
const platform = os.platform();
const arch = os.arch();

// Map Node.js platform/arch to Go naming conventions
function getPlatformName() {
  const platforms = {
    'win32': 'windows',
    'darwin': 'darwin',
    'linux': 'linux'
  };
  return platforms[platform] || platform;
}

function getArchName() {
  const architectures = {
    'x64': 'amd64',
    'arm64': 'arm64',
    'ia32': '386'
  };
  return architectures[arch] || arch;
}

// Build binary filename
const platformName = getPlatformName();
const archName = getArchName();
const extension = platform === 'win32' ? '.exe' : '';
const binaryName = `guardian${extension}`;

// Set up directories
const binaryDir = path.join(__dirname, '..', `${platform}-${arch}`);
const binaryPath = path.join(binaryDir, binaryName);

// Ensure binary directory exists
if (!fs.existsSync(binaryDir)) {
  try {
    fs.mkdirSync(binaryDir, { recursive: true });
  } catch (err) {
    console.error(`Error creating directory ${binaryDir}: ${err.message}`);
    process.exit(1);
  }
}

// Determine download URL
const archiveExtension = platform === 'win32' ? 'zip' : 'tar.gz';
const remoteFilename = `guardian-${platformName}-${archName}-v${GUARDIAN_VERSION}.${archiveExtension}`;
const downloadUrl = `${BASE_URL}/v${GUARDIAN_VERSION}/${remoteFilename}`;

console.log(`Guardian: Installing binary for ${platformName}-${archName}`);
console.log(`Guardian: Downloading from ${downloadUrl}`);

// Download and extract the binary
function downloadFile(url, destPath, retries = 0) {
  return new Promise((resolve, reject) => {
    const file = fs.createWriteStream(destPath, { mode: 0o755 });
    
    const handleError = (err) => {
      fs.unlink(destPath, () => {}); // Ignore unlink errors
      if (retries < MAX_RETRIES) {
        console.log(`Guardian: Retrying download (${retries + 1}/${MAX_RETRIES})...`);
        setTimeout(() => {
          downloadFile(url, destPath, retries + 1)
            .then(resolve)
            .catch(reject);
        }, RETRY_DELAY);
      } else {
        reject(err);
      }
    };

    const request = https.get(url, (response) => {
      if (response.statusCode === 302 || response.statusCode === 301) {
        // Handle redirect
        downloadFile(response.headers.location, destPath, retries)
          .then(resolve)
          .catch(reject);
        return;
      }
      
      if (response.statusCode !== 200) {
        return reject(new Error(`Failed to download, status code: ${response.statusCode}`));
      }
      
      response.pipe(file);
      
      file.on('finish', () => {
        file.close();
        resolve();
      });
    });
    
    request.on('error', handleError);
    
    file.on('error', handleError);
  });
}

// Create a temporary file to download the archive
const tempDir = os.tmpdir();
const tempFilePath = path.join(tempDir, remoteFilename);

// Download and extract
async function downloadAndExtract() {
  try {
    await downloadFile(downloadUrl, tempFilePath);
    console.log('Guardian: Binary archive downloaded successfully');
    
    // Extract the binary based on platform
    if (platform === 'win32') {
      // For Windows, extract zip file
      console.log('Guardian: Extracting zip archive');
      try {
        // Using Node.js unzip libraries would require additional dependencies
        // For simplicity in the npm package, use the built-in Windows commands
        const extractCmd = `powershell -command "Expand-Archive -Path '${tempFilePath}' -DestinationPath '${binaryDir}' -Force"`;
        execSync(extractCmd);
      } catch (err) {
        console.error(`Guardian: Error extracting zip: ${err.message}`);
        console.log('Guardian: Falling back to manual extraction');
        
        // Fallback approach for Windows
        const AdmZip = require('adm-zip');
        try {
          const zip = new AdmZip(tempFilePath);
          zip.extractAllTo(binaryDir, true);
        } catch (zipErr) {
          console.error(`Guardian: Fallback extraction failed: ${zipErr.message}`);
          throw zipErr;
        }
      }
    } else {
      // For Unix-like systems, extract tar.gz
      console.log('Guardian: Extracting tar.gz archive');
      try {
        execSync(`tar -xzf "${tempFilePath}" -C "${binaryDir}"`);
      } catch (err) {
        console.error(`Guardian: Error extracting tar.gz: ${err.message}`);
        throw err;
      }
    }
    
    console.log(`Guardian: Making binary executable: ${binaryPath}`);
    try {
      // Set executable permissions (does nothing on Windows but no harm)
      fs.chmodSync(binaryPath, 0o755);
    } catch (err) {
      console.error(`Guardian: Error setting executable permissions: ${err.message}`);
      // Non-fatal, continue
    }
    
    // Clean up temp file
    fs.unlink(tempFilePath, () => {});
    
    console.log('Guardian: Binary installation complete');
  } catch (err) {
    console.error(`Guardian: Binary installation failed: ${err.message}`);
    console.error('You may need to install the Guardian binary manually.');
    
    // Print manual installation instructions
    console.error('\nManual installation instructions:');
    console.error(`1. Download the appropriate binary for your platform from: ${BASE_URL}/v${GUARDIAN_VERSION}`);
    console.error(`2. Extract the archive`);
    console.error(`3. Place the binary in: ${binaryDir}`);
    console.error(`4. Ensure the binary has executable permissions (chmod +x on Unix-like systems)`);
    
    // Exit with error code
    process.exit(1);
  }
}

// Start the download and extraction
downloadAndExtract();