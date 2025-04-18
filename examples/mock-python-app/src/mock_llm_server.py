"""
Mock LLM Server that simulates an AI service like OpenAI.
"""

from fastapi import FastAPI, HTTPException, Request
import uvicorn
from typing import Dict, Any, List
from mock_ai_service import MockAIService
import logging

# Setup logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger("mock-llm")

# Initialize FastAPI app
app = FastAPI(title="Mock LLM Server")

# Create mock AI service
mock_ai = MockAIService()

@app.get("/")
async def read_root():
    """Root endpoint."""
    return {"message": "Mock LLM Server"}

@app.post("/v1/chat/completions")
async def chat_completions(request: Request):
    """
    Process chat completion requests.
    """
    logger.info("[Mock LLM] Processing chat completion")
    
    try:
        # Get request body
        body = await request.json()
        messages = body.get("messages")
        
        if not messages:
            logger.error("[Mock LLM] Missing messages in request body")
            raise HTTPException(
                status_code=400,
                detail="Missing messages in request body"
            )
        
        # Process with mock AI service
        response = mock_ai.chat_completions(messages)
        return response
    except Exception as e:
        logger.error(f"[Mock LLM] Error: {str(e)}")
        raise HTTPException(
            status_code=500,
            detail=str(e)
        )

@app.post("/v1/completions")
async def text_completions(request: Request):
    """
    Process text completion requests.
    """
    logger.info("[Mock LLM] Processing text completion")
    
    try:
        # Get request body
        body = await request.json()
        prompt = body.get("prompt")
        
        if not prompt:
            logger.error("[Mock LLM] Missing prompt in request body")
            raise HTTPException(
                status_code=400,
                detail="Missing prompt in request body"
            )
        
        # Process with mock AI service
        response = mock_ai.text_completions(prompt)
        return response
    except Exception as e:
        logger.error(f"[Mock LLM] Error: {str(e)}")
        raise HTTPException(
            status_code=500,
            detail=str(e)
        )

@app.api_route("/{path:path}", methods=["GET", "POST", "PUT", "DELETE"])
async def catch_all(path: str):
    """Catch all other requests."""
    logger.info(f"[Mock LLM] Unhandled path: {path}")
    return {"error": "404 Not Found"}

if __name__ == "__main__":
    logger.info("Starting Mock LLM Server on port 3456")
    uvicorn.run(app, host="0.0.0.0", port=3456)