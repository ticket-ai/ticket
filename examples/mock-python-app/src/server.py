"""
Mock FastAPI Server that uses Guardian AI to monitor AI API calls.
"""

import os
import sys
import requests
from fastapi import FastAPI, Request
import uvicorn
from guardian_ai import Guardian
from fastapi.middleware.cors import CORSMiddleware

# Initialize FastAPI app
app = FastAPI(title="Guardian AI Mock Python App")

# Add CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Initialize Guardian with minimal configuration
guardian = Guardian({
    "service_name": "python-app",
    "environment": "development",
    "debug": True
})

# Add Guardian middleware to the FastAPI app
app.add_middleware(guardian.create_fastapi_middleware())

# Configuration
MOCK_LLM_URL = "http://localhost:3457"

@app.get("/")
async def read_root():
    return {"message": "Guardian AI FastAPI Mock Server"}

@app.post("/api/chat")
async def chat(request: Request):
    """
    Chat endpoint that forwards requests to the Mock LLM service.
    Guardian AI automatically monitors and protects this endpoint.
    """
    try:
        # Get request body
        body = await request.json()
        
        # Forward to Mock LLM
        llm_url = f"{MOCK_LLM_URL}/v1/chat/completions"
        response = requests.post(
            llm_url,
            json=body,
            headers={"Content-Type": "application/json"}
        )
        
        return response.json()
    except Exception as e:
        return {
            "error": "Failed to get response from LLM service",
            "details": str(e)
        }

@app.get("/admin/flagged-users")
async def get_flagged_users():
    """Mock endpoint for admin panel."""
    return {"flagged_users": []}

@app.get("/admin/blocked-ips")
async def get_blocked_ips():
    """Mock endpoint for admin panel."""
    return {"blocked_ips": []}

if __name__ == "__main__":
    PORT = 3002
    print(f"Server running at http://localhost:{PORT}")
    print(f"Using Mock LLM at {MOCK_LLM_URL}")
    uvicorn.run(app, host="0.0.0.0", port=PORT)