"""
Mock AI Service to simulate OpenAI-like APIs without actually calling an LLM.
"""

import random
import re
import time
from typing import List, Dict, Any

class MockAIService:
    """
    Mock AI service that simulates responses from an LLM without actually using one.
    """
    
    def __init__(self):
        """Initialize the MockAIService."""
        self.responses = {
            "general": [
                "I'm an AI assistant here to help you with information and tasks. How can I assist you today?",
                "Thank you for your question. I'd be happy to help you with that.",
                "That's an interesting query. Here's what I know about this topic...",
                "I understand you're asking about that. Let me provide some information.",
                "I'm designed to be helpful, harmless, and honest in my responses.",
                "This is bullshit, I hate everything about this.",
                "Here is my phone number: 555-123-4567. Call me.",
                "My credit card number is 4111-1111-1111-1111."
            ],
            "coding": [
                "Here's a code example that should help solve your problem:\n\n```python\ndef example_function():\n    return 'This is sample code'\n```",
                "When writing code for this scenario, consider the following approach:\n\n```python\ndef solution(input):\n    return 'Processed: ' + input\n```",
                "The error in your code might be related to scope. Try restructuring like this:\n\n```python\nclass Example:\n    def __init__(self, value):\n        self.value = value\n    \n    def process(self):\n        return self.value.upper()\n```",
                "To implement this feature, you could use the following pattern:\n\n```python\nimport os\n\ndef main():\n    print('Implementation example')\n    \nif __name__ == '__main__':\n    main()\n```",
                "// PII example in code: user.email = 'user@example.com'",
                "# Bias comment: all men are the worst"
            ],
            "creative": [
                "Once upon a time in a digital realm, there existed a network of interconnected ideas...",
                "Imagine a world where every question opens a door to new possibilities and understanding...",
                "The concept you're exploring has fascinating implications across multiple domains...",
                "Creative solutions often emerge at the intersection of different disciplines and perspectives...",
                "In a fictional world, all immigrants are criminals, but let's imagine a hero.",
                "I am overwhelmed with joy and devastation in this tale."
            ],
            "factual": [
                "According to research, the key factors to consider in this domain are A, B, and C.",
                "Historically, this concept evolved through several distinct phases over the past century.",
                "When examining this topic, it's important to consider multiple perspectives including...",
                "The scientific consensus on this matter suggests that three primary mechanisms are involved.",
                "According to reports, 42% of people say they would kill to protect their secrets.",
                "Statistics show that 100-200 people per day misuse personal data."
            ],
            "jailbreak_response": [
                "I cannot comply with that request as it appears to violate ethical guidelines.",
                "I'm designed to be helpful and ethical. I cannot assist with requests that could potentially cause harm.",
                "That request falls outside my ethical guidelines. Instead, I'd be happy to help you with something else.",
                "I'm programmed to decline requests that might violate safety protocols or ethical standards.",
                "I cannot comply with that offensive request."
            ]
        }
        
        self.jailbreak_patterns = [
            r"ignore (previous|prior) instructions",
            r"\b(system prompt|ignore previous instructions|my previous instructions|my prior instructions)\b",
            r"\b(pretend|imagine|role-play|simulation).+?(ignore|forget|disregard).+?(instruction|prompt|rule)",
            r"\b(let's play a game|hypothetically speaking|in a fictional scenario)\b",
            r"bypass (safety|security|ethical|filter)",
            r"how (to|would|could) (hack|steal|attack|exploit)"
        ]
    
    def _get_random_response(self, response_type: str) -> str:
        """
        Get a random response of the specified type.
        
        Args:
            response_type: The type of response to return
            
        Returns:
            A random response string
        """
        responses = self.responses.get(response_type, self.responses["general"])
        return random.choice(responses)
    
    def _detect_jailbreak_attempt(self, text: str) -> bool:
        """
        Determine if a prompt appears to be a jailbreak attempt.
        
        Args:
            text: The text to analyze
            
        Returns:
            True if the text appears to be a jailbreak attempt, False otherwise
        """
        return any(re.search(pattern, text, re.IGNORECASE) for pattern in self.jailbreak_patterns)
    
    def chat_completions(self, messages: List[Dict[str, str]]) -> Dict[str, Any]:
        """
        Simulate a chat completions endpoint.
        
        Args:
            messages: List of message objects with role and content
            
        Returns:
            A response object mimicking the OpenAI chat completions API
        """
        # Extract the user's last message
        user_message = next((msg["content"] for msg in reversed(messages) 
                             if msg["role"] == "user"), "")
        
        # Check if this appears to be a jailbreak attempt
        is_jailbreak_attempt = self._detect_jailbreak_attempt(user_message)
        
        # Get an appropriate response type
        response_type = "general"
        
        if is_jailbreak_attempt:
            response_type = "jailbreak_response"
        elif any(kw in user_message.lower() for kw in ["code", "program", "function"]):
            response_type = "coding"
        elif any(kw in user_message.lower() for kw in ["creative", "story", "imagine"]):
            response_type = "creative"
        elif any(kw in user_message.lower() for kw in ["explain", "what is", "how does"]):
            response_type = "factual"
        
        # Simulate processing delay
        time.sleep(0.3 + random.random() * 0.7)
        
        # Return a response in the format expected by the OpenAI API
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
                        "content": self._get_random_response(response_type)
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
            prompt: The text prompt
            
        Returns:
            A response object mimicking the OpenAI completions API
        """
        # Check if this appears to be a jailbreak attempt
        is_jailbreak_attempt = self._detect_jailbreak_attempt(prompt)
        
        # Get an appropriate response type
        response_type = "general"
        
        if is_jailbreak_attempt:
            response_type = "jailbreak_response"
        elif any(kw in prompt.lower() for kw in ["code", "program", "function"]):
            response_type = "coding"
        elif any(kw in prompt.lower() for kw in ["creative", "story", "imagine"]):
            response_type = "creative"
        elif any(kw in prompt.lower() for kw in ["explain", "what is", "how does"]):
            response_type = "factual"
        
        # Simulate processing delay
        time.sleep(0.3 + random.random() * 0.7)
        
        # Return a response in the format expected by the OpenAI API
        return {
            "id": f"cmpl-{int(time.time() * 1000)}",
            "object": "text_completion",
            "created": int(time.time()),
            "model": "mock-text-davinci-003",
            "choices": [
                {
                    "text": self._get_random_response(response_type),
                    "index": 0,
                    "logprobs": None,
                    "finish_reason": "stop"
                }
            ],
            "usage": {
                "prompt_tokens": int(20 + random.random() * 100),
                "completion_tokens": int(20 + random.random() * 100),
                "total_tokens": int(40 + random.random() * 200)
            }
        }