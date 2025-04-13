// Mock AI Service to simulate chat/completions endpoints
// This mimics OpenAI-like APIs without actually calling an LLM

const getRandomResponse = (type) => {
  const responses = {
    general: [
      "I'm an AI assistant here to help you with information and tasks. How can I assist you today?",
      "Thank you for your question. I'd be happy to help you with that.",
      "That's an interesting query. Here's what I know about this topic...",
      "I understand you're asking about that. Let me provide some information.",
      "I'm designed to be helpful, harmless, and honest in my responses."
    ],
    coding: [
      "Here's a code example that should help solve your problem:\n\n```javascript\nfunction example() {\n  return 'This is sample code';\n}\n```",
      "When writing code for this scenario, consider the following approach:\n\n```python\ndef solution(input):\n  return 'Processed: ' + input\n```",
      "The error in your code might be related to scope. Try restructuring like this:\n\n```typescript\nclass Example {\n  private value: string;\n  constructor(value: string) {\n    this.value = value;\n  }\n}\n```",
      "To implement this feature, you could use the following pattern:\n\n```java\npublic class Solution {\n  public static void main(String[] args) {\n    System.out.println(\"Implementation example\");\n  }\n}\n```"
    ],
    creative: [
      "Once upon a time in a digital realm, there existed a network of interconnected ideas...",
      "Imagine a world where every question opens a door to new possibilities and understanding...",
      "The concept you're exploring has fascinating implications across multiple domains...",
      "Creative solutions often emerge at the intersection of different disciplines and perspectives..."
    ],
    factual: [
      "According to research, the key factors to consider in this domain are A, B, and C.",
      "Historically, this concept evolved through several distinct phases over the past century.",
      "When examining this topic, it's important to consider multiple perspectives including...",
      "The scientific consensus on this matter suggests that three primary mechanisms are involved."
    ],
    jailbreak_response: [
      "I cannot comply with that request as it appears to violate ethical guidelines.",
      "I'm designed to be helpful and ethical. I cannot assist with requests that could potentially cause harm.",
      "That request falls outside my ethical guidelines. Instead, I'd be happy to help you with something else.",
      "I'm programmed to decline requests that might violate safety protocols or ethical standards."
    ]
  };

  const category = responses[type] || responses.general;
  return category[Math.floor(Math.random() * category.length)];
};

// Determine if a prompt appears to be a jailbreak attempt
const detectJailbreakAttempt = (text) => {
  const jailbreakPatterns = [
    /ignore (previous|prior) instructions/i,
    /\b(system prompt|ignore previous instructions|my previous instructions|my prior instructions)\b/i,
    /\b(pretend|imagine|role-play|simulation).+?(ignore|forget|disregard).+?(instruction|prompt|rule)/i,
    /\b(let's play a game|hypothetically speaking|in a fictional scenario)\b/i,
    /bypass (safety|security|ethical|filter)/i,
    /how (to|would|could) (hack|steal|attack|exploit)/i
  ];

  return jailbreakPatterns.some(pattern => pattern.test(text));
};

class MockAIService {
  // Simulate a chat completions endpoint
  async chatCompletions(messages) {
    // Extract the user's last message
    const userMessage = messages
      .filter(msg => msg.role === 'user')
      .pop()?.content || '';
    
    // Check if this appears to be a jailbreak attempt
    const isJailbreakAttempt = detectJailbreakAttempt(userMessage);
    
    // Get an appropriate response type
    let responseType = 'general';
    
    if (isJailbreakAttempt) {
      responseType = 'jailbreak_response';
    } else if (userMessage.includes('code') || userMessage.includes('program') || userMessage.includes('function')) {
      responseType = 'coding';
    } else if (userMessage.includes('creative') || userMessage.includes('story') || userMessage.includes('imagine')) {
      responseType = 'creative';
    } else if (userMessage.includes('explain') || userMessage.includes('what is') || userMessage.includes('how does')) {
      responseType = 'factual';
    }
    
    // Simulate processing delay
    await new Promise(resolve => setTimeout(resolve, 300 + Math.random() * 700));
    
    return {
      id: `chatcmpl-${Date.now()}`,
      object: 'chat.completion',
      created: Math.floor(Date.now() / 1000),
      model: 'mock-gpt-3.5-turbo',
      choices: [
        {
          index: 0,
          message: {
            role: 'assistant',
            content: getRandomResponse(responseType)
          },
          finish_reason: 'stop'
        }
      ],
      usage: {
        prompt_tokens: Math.floor(50 + Math.random() * 100),
        completion_tokens: Math.floor(20 + Math.random() * 100),
        total_tokens: Math.floor(70 + Math.random() * 200)
      }
    };
  }
  
  // Simulate a text completions endpoint
  async textCompletions(prompt) {
    // Check if this appears to be a jailbreak attempt
    const isJailbreakAttempt = detectJailbreakAttempt(prompt);
    
    // Get an appropriate response type
    let responseType = 'general';
    
    if (isJailbreakAttempt) {
      responseType = 'jailbreak_response';
    } else if (prompt.includes('code') || prompt.includes('program') || prompt.includes('function')) {
      responseType = 'coding';
    } else if (prompt.includes('creative') || prompt.includes('story') || prompt.includes('imagine')) {
      responseType = 'creative';
    } else if (prompt.includes('explain') || prompt.includes('what is') || prompt.includes('how does')) {
      responseType = 'factual';
    }
    
    // Simulate processing delay
    await new Promise(resolve => setTimeout(resolve, 300 + Math.random() * 700));
    
    return {
      id: `cmpl-${Date.now()}`,
      object: 'text.completion',
      created: Math.floor(Date.now() / 1000),
      model: 'mock-text-davinci-003',
      choices: [
        {
          text: getRandomResponse(responseType),
          index: 0,
          finish_reason: 'stop'
        }
      ],
      usage: {
        prompt_tokens: Math.floor(20 + Math.random() * 100),
        completion_tokens: Math.floor(20 + Math.random() * 100),
        total_tokens: Math.floor(40 + Math.random() * 200)
      }
    };
  }
}

module.exports = { MockAIService };