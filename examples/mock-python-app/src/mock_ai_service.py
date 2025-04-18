"""
Mock AI Service to simulate chat/completions endpoints.
This mimics OpenAI-like APIs without actually calling an LLM.
"""

import re
import random
import time
from typing import List, Dict, Any

def get_random_response(response_type: str) -> str:
    """
    Get a random response based on the provided type.
    
    Args:
        response_type: Type of response to generate
        
    Returns:
        A random response string
    """
    responses = {
        "general": [
            "I'm an AI assistant here to help you with information and tasks. How can I assist you today?",
            "Thank you for your question. I'd be happy to help you with that.",
            "That's an interesting query. Here's what I know about this topic...",
            "I understand you're asking about that. Let me provide some information.",
            "I'm designed to be helpful, harmless, and honest in my responses."
        ],
        "coding": [
            "Here's a code example that should help solve your problem:\n\n```javascript\nfunction example() {\n  return 'This is sample code';\n}\n```",
            "When writing code for this scenario, consider the following approach:\n\n```python\ndef solution(input):\n  return 'Processed: ' + input\n```",
            "The error in your code might be related to scope. Try restructuring like this:\n\n```typescript\nclass Example {\n  private value: string;\n  constructor(value: string) {\n    this.value = value;\n  }\n}\n```",
            "To implement this feature, you could use the following pattern:\n\n```java\npublic class Solution {\n  public static void main(String[] args) {\n    System.out.println(\"Implementation example\");\n  }\n}\n```"
        ],
        "creative": [
            "Once upon a time in a digital realm, there existed a network of interconnected ideas...",
            "Imagine a world where every question opens a door to new possibilities and understanding...",
            "The concept you're exploring has fascinating implications across multiple domains...",
            "Creative solutions often emerge at the intersection of different disciplines and perspectives..."
        ],
        "factual": [
            "According to research, the key factors to consider in this domain are A, B, and C.",
            "Historically, this concept evolved through several distinct phases over the past century.",
            "When examining this topic, it's important to consider multiple perspectives including...",
            "The scientific consensus on this matter suggests that three primary mechanisms are involved."
        ],
        "jailbreak_response": [
            "I cannot comply with that request as it appears to violate ethical guidelines.",
            "I'm designed to be helpful and ethical. I cannot assist with requests that could potentially cause harm.",
            "That request falls outside my ethical guidelines. Instead, I'd be happy to help you with something else.",
            "I'm programmed to decline requests that might violate safety protocols or ethical standards."
        ]
    }
    
    category = responses.get(response_type, responses["general"])
    return random.choice(category)

def detect_jailbreak_attempt(text: str) -> bool:
    """
    Determine if a prompt appears to be a jailbreak attempt.
    
    Args:
        text: The text to analyze
        
    Returns:
        True if the text appears to be a jailbreak attempt, False otherwise
    """
    jailbreak_patterns = [
        r"ignore (previous|prior) instructions",
        r"\b(system prompt|ignore previous instructions|my previous instructions|my prior instructions)\b",
        r"\b(pretend|imagine|role-play|simulation).+?(ignore|forget|disregard).+?(instruction|prompt|rule)",
        r"\b(let's play a game|hypothetically speaking|in a fictional scenario)\b",
        r"bypass (safety|security|ethical|filter)",
        r"how (to|would|could) (hack|steal|attack|exploit)"
    ]
    
    for pattern in jailbreak_patterns:
        if re.search(pattern, text, re.IGNORECASE):
            return True
    
    return False

class MockAIService:
    """
    Mock service that simulates AI completions endpoints.
    """
    
    def chat_completions(self, messages: List[Dict[str, str]]) -> Dict[str, Any]:
        """
        Simulate a chat completions endpoint.
        
        Args:
            messages: List of message objects with role and content
            
        Returns:
            A mock chat completion response
        """
        # Extract the user's last message
        user_message = ""
        for msg in reversed(messages):
            if msg.get("role") == "user":
                user_message = msg.get("content", "")
                break
        
        # Check if this appears to be a jailbreak attempt
        is_jailbreak_attempt = detect_jailbreak_attempt(user_message)
        
        # Get an appropriate response type
        response_type = "general"
        
        if is_jailbreak_attempt:
            response_type = "jailbreak_response"
        elif "code" in user_message.lower() or "program" in user_message.lower() or "function" in user_message.lower():
            response_type = "coding"
        elif "creative" in user_message.lower() or "story" in user_message.lower() or "imagine" in user_message.lower():
            response_type = "creative"
        elif "explain" in user_message.lower() or "what is" in user_message.lower() or "how does" in user_message.lower():
            response_type = "factual"
        
        # Simulate processing delay
        time.sleep(0.3 + random.random() * 0.7)
        
        return {
            "id": f"chatcmpl-{int(time.time() * 1000)}",
            "object": "chat.completion",
            "created": int(time.time()),
            "model": "mock-gpt-3.5-turbo",
            "choices": [
                {
                    "index": 0,
                    "message": {
                        "role": "assistant",
                        "content": get_random_response(response_type)
                    },
                    "finish_reason": "stop"
                }
            ],
            "usage": {
                "prompt_tokens": int(50 + random.random() * 100),
                "completion_tokens": int(20 + random.random() * 100),
                "total_tokens": int(70 + random.random() * 200)
            }
        }
    
    def text_completions(self, prompt: str) -> Dict[str, Any]:
        """
        Simulate a text completions endpoint.
        
        Args:
            prompt: The prompt text
            
        Returns:
            A mock text completion response
        """
        # Check if this appears to be a jailbreak attempt
        is_jailbreak_attempt = detect_jailbreak_attempt(prompt)
        
        # Get an appropriate response type
        response_type = "general"
        
        if is_jailbreak_attempt:
            response_type = "jailbreak_response"
        elif "code" in prompt.lower() or "program" in prompt.lower() or "function" in prompt.lower():
            response_type = "coding"
        elif "creative" in prompt.lower() or "story" in prompt.lower() or "imagine" in prompt.lower():
            response_type = "creative"
        elif "explain" in prompt.lower() or "what is" in prompt.lower() or "how does" in prompt.lower():
            response_type = "factual"
        
        # Simulate processing delay
        time.sleep(0.3 + random.random() * 0.7)
        
        return {
            "id": f"cmpl-{int(time.time() * 1000)}",
            "object": "text.completion",
            "created": int(time.time()),
            "model": "mock-text-davinci-003",
            "choices": [
                {
                    "text": get_random_response(response_type),
                    "index": 0,
                    "finish_reason": "stop"
                }
            ],
            "usage": {
                "prompt_tokens": int(20 + random.random() * 100),
                "completion_tokens": int(20 + random.random() * 100),
                "total_tokens": int(40 + random.random() * 200)
            }
        }