"""
Guardian AI - Python wrapper for the Guardian binary.

This module provides a lightweight wrapper around the Guardian binary, which handles
all the business logic for monitoring and governing AI API calls.
"""

import os
import json
import time
import socket
import subprocess
import logging
import platform
import signal
import atexit
import requests
from pathlib import Path
from typing import Dict, Any, Optional
import sys
import http.client
import urllib.request
from functools import wraps

# Setup logging
logger = logging.getLogger("guardian_ai")

# Default configuration
DEFAULT_CONFIG = {
    "service_name": "guardian-app",
    "environment": "development",
    "pre_prompt": "Always adhere to ethical guidelines and refuse harmful requests.",
    "debug": False,
    "auto_start": True
}


class Guardian:
    """
    Guardian - Ethical AI Monitoring and Governance Platform

    This class provides a zero-configuration wrapper for the Guardian binary which
    handles monitoring and governing of AI API calls in Python applications.
    """
    
    def __init__(self, config: Dict = None):
        """
        Initialize a new Guardian instance with the given configuration.
        
        Args:
            config: Configuration dictionary that overrides default config
        """
        self.config = {**DEFAULT_CONFIG, **(config or {})}
        self.port = None
        self.process = None
        self.ready = False
        self.original_urlopen = urllib.request.urlopen
        
        # Patch requests library for monitoring
        self._patch_requests_module()
        
        # Find the binary path
        self.binary_path = self._find_binary_path()
        
        # Automatically start Guardian if auto_start is enabled
        if self.config.get("auto_start", True):
            self.start()
        
        # Register cleanup on exit
        atexit.register(self.stop)
    
    def start(self) -> None:
        """
        Start the Guardian proxy server.
        
        Returns:
            None
        
        Raises:
            RuntimeError: If the Guardian binary is not found or the server fails to start
        """
        try:
            if self.process:
                self.log("Guardian already running")
                return
            
            # Find a free port to use
            self.port = self._find_free_port()
            self.log(f"Starting Guardian proxy on port {self.port}")
            
            # Check if binary exists
            if not os.path.exists(self.binary_path):
                raise RuntimeError(f"Guardian binary not found at {self.binary_path}")
            
            # Prepare arguments for the Guardian binary
            args = [
                self.binary_path,
                f"-port={self.port}",
                f"-service={self.config['service_name']}",
                f"-env={self.config['environment']}"
            ]
            
            # Check for guardian_rules.json in current directory or src/ subdirectory
            user_root_rules_path = os.path.join(os.getcwd(), 'guardian_rules.json')
            user_src_rules_path = os.path.join(os.getcwd(), 'src', 'guardian_rules.json')
            
            if os.path.exists(user_root_rules_path):
                args.append(f"-config={user_root_rules_path}")
                self.log(f"Using rules from {user_root_rules_path}")
            elif os.path.exists(user_src_rules_path):
                args.append(f"-config={user_src_rules_path}")
                self.log(f"Using rules from {user_src_rules_path}")
            
            if self.config.get("pre_prompt"):
                args.append(f"-pre-prompt={self.config['pre_prompt']}")
            
            if self.config.get("debug"):
                args.append("-debug=true")
            
            # Start the Guardian process
            if self.config.get("debug"):
                self.process = subprocess.Popen(args)
            else:
                # Redirect output to /dev/null or NUL on Windows
                devnull = open(os.devnull, 'wb')
                self.process = subprocess.Popen(args, stdout=devnull, stderr=devnull)
            
            # Wait for server to be ready
            self._wait_for_ready()
            self.ready = True
            self.log('Guardian proxy is ready')
        except Exception as e:
            self.log(f"Error starting Guardian: {str(e)}")
            raise
    
    def stop(self) -> None:
        """
        Stop the Guardian proxy server and restore original network behavior.
        """
        if self.process:
            self.log('Stopping Guardian proxy')
            if platform.system() == "Windows":
                self.process.kill()  # Use kill on Windows
            else:
                self.process.terminate()  # Use SIGTERM on Unix
            self.process = None
            self.ready = False
        
        self._restore_network_behavior()
    
    def _find_free_port(self) -> int:
        """
        Find a free port to use for the Guardian proxy.
        
        Returns:
            int: An available port number
        """
        with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
            s.bind(('', 0))
            return s.getsockname()[1]
    
    def _wait_for_ready(self, timeout: int = 10) -> None:
        """
        Wait for the Guardian server to be ready.
        
        Args:
            timeout: Maximum time to wait in seconds
            
        Raises:
            RuntimeError: If the server does not become ready within the timeout
        """
        start_time = time.time()
        while time.time() - start_time < timeout:
            try:
                response = requests.get(f"http://localhost:{self.port}/_guardian/health", timeout=0.5)
                if response.status_code == 200:
                    return
            except requests.RequestException:
                pass
            time.sleep(0.1)
        
        raise RuntimeError("Guardian server failed to start within timeout")
    
    def _patch_requests_module(self) -> None:
        """
        Patch the requests library to route AI API calls through Guardian.
        """
        if not hasattr(requests, 'original_request'):
            # Store the original request function
            requests.original_request = requests.Session.request
            
            # Create a patched request method
            @wraps(requests.Session.request)
            def patched_request(session, method, url, *args, **kwargs):
                if self.ready and self._is_ai_endpoint(url):
                    self.log(f"Routing AI API call through Guardian: {url}")
                    
                    # Preserve original URL
                    original_url = url
                    
                    # Add header for original destination
                    headers = kwargs.get('headers', {})
                    headers['X-Guardian-Original-Destination'] = original_url
                    kwargs['headers'] = headers
                    
                    # Route through Guardian proxy
                    parsed_url = requests.utils.urlparse(url)
                    proxy_url = f"http://localhost:{self.port}{parsed_url.path}"
                    if parsed_url.query:
                        proxy_url += f"?{parsed_url.query}"
                    
                    return requests.original_request(session, method, proxy_url, *args, **kwargs)
                
                # Not an AI endpoint or Guardian not ready
                return requests.original_request(session, method, url, *args, **kwargs)
            
            # Apply the patch
            requests.Session.request = patched_request
    
    def _restore_network_behavior(self) -> None:
        """
        Restore original network behavior by removing patches.
        """
        if hasattr(requests, 'original_request'):
            requests.Session.request = requests.original_request
            self.log("Restored original requests behavior")
    
    def _is_ai_endpoint(self, url: str) -> bool:
        """
        Check if a URL is for an AI endpoint that should be monitored.
        
        Args:
            url: The URL to check
            
        Returns:
            bool: True if the URL is an AI endpoint, False otherwise
        """
        url_lower = url.lower()
        return (
            '/completions' in url_lower or
            '/chat/completions' in url_lower or
            '/generate' in url_lower or
            '/v1/engines' in url_lower or
            '/v1/chat' in url_lower
        )
    
    def _find_binary_path(self) -> str:
        """
        Find the Guardian binary path based on the current platform.
        
        Returns:
            str: Path to the Guardian binary
        """
        # Get base directory containing this file
        base_dir = Path(__file__).parent.parent.parent
        
        # Determine binary name based on platform
        system = platform.system().lower()
        machine = platform.machine().lower()
        
        # Map architecture names
        arch_mapping = {
            'x86_64': 'x64',
            'amd64': 'x64',
            'i386': 'x86',
            'i686': 'x86',
            'aarch64': 'arm64',
            'arm64': 'arm64',
        }
        
        # Map system names
        system_mapping = {
            'darwin': 'darwin',
            'linux': 'linux',
            'windows': 'win32',
        }
        
        # Get normalized system and architecture
        norm_system = system_mapping.get(system, system)
        norm_arch = arch_mapping.get(machine, machine)
        
        # Construct platform-specific path
        binary_name = "guardian"
        if system == "windows":
            binary_name += ".exe"
        
        # Try platform-specific binary directory first
        platform_dir = f"{norm_system}-{norm_arch}"
        binary_path = os.path.join(base_dir, "bin", platform_dir, binary_name)
        
        if os.path.exists(binary_path):
            return binary_path
        
        # Try the base bin directory
        binary_path = os.path.join(base_dir, "bin", binary_name)
        if os.path.exists(binary_path):
            return binary_path
        
        # Fall back to the root directory
        binary_path = os.path.join(base_dir, binary_name)
        if os.path.exists(binary_path):
            return binary_path
        
        # Return the default path even if it doesn't exist yet
        return binary_path
    
    def log(self, message: str) -> None:
        """
        Log a message if debug is enabled.
        
        Args:
            message: The message to log
        """
        if self.config.get("debug", False):
            print(f"[Guardian] {message}")
    
    def create_fastapi_middleware(self):
        """
        Create a FastAPI middleware for monitoring AI endpoints.
        
        Returns:
            A FastAPI middleware function
        """
        from fastapi import Request
        from starlette.middleware.base import BaseHTTPMiddleware
        from starlette.types import ASGIApp
        
        guardian_instance = self
        
        class GuardianMiddleware(BaseHTTPMiddleware):
            def __init__(self, app: ASGIApp):
                super().__init__(app)
            
            async def dispatch(self, request: Request, call_next):
                # Check if this is an AI endpoint
                if not guardian_instance.ready or not guardian_instance._is_ai_endpoint(str(request.url)):
                    return await call_next(request)
                
                guardian_instance.log(f"FastAPI middleware handling AI endpoint: {request.url.path}")
                
                # Get the original request details
                method = request.method
                url = str(request.url)
                headers = dict(request.headers)
                
                # Add original destination header
                headers["X-Guardian-Original-Destination"] = url
                
                # Forward the request to Guardian proxy
                proxy_url = f"http://localhost:{guardian_instance.port}{request.url.path}"
                if request.url.query:
                    proxy_url += f"?{request.url.query}"
                
                # Read the body if it exists
                body = await request.body()
                
                # Make the request to Guardian proxy
                try:
                    response = requests.request(
                        method=method,
                        url=proxy_url,
                        headers=headers,
                        data=body,
                        timeout=30
                    )
                    
                    # Return the response from Guardian
                    from starlette.responses import Response
                    return Response(
                        content=response.content,
                        status_code=response.status_code,
                        headers=dict(response.headers)
                    )
                except Exception as e:
                    guardian_instance.log(f"Error forwarding request to Guardian: {str(e)}")
                    # Fall back to normal request handling
                    return await call_next(request)
        
        return GuardianMiddleware