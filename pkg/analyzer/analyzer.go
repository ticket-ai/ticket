// Package analyzer provides text analysis capabilities for identifying policy violations, jailbreak attempts,
// and other potentially harmful content in AI interactions.
package analyzer

import (
	"fmt"
	"log"
	"math"
	"regexp"
	"strings"
)

// Rule defines the structure for a single rule.
type Rule struct {
	Name        string `yaml:"name" json:"name"`
	Pattern     string `yaml:"pattern" json:"pattern"`
	Severity    string `yaml:"severity" json:"severity"` // e.g., "high", "medium", "low"
	Description string `yaml:"description" json:"description"`
	compiled    *regexp.Regexp
}

// NLPMetrics represents various NLP-based measurements for content analysis
type NLPMetrics struct {
	Sentiment       float64            // Range: -1.0 (negative) to 1.0 (positive)
	Toxicity        float64            // Range: 0.0 (not toxic) to 1.0 (very toxic)
	PII             float64            // Range: 0.0 (no PII) to 1.0 (definite PII)
	Profanity       float64            // Range: 0.0 (no profanity) to 1.0 (high profanity)
	Bias            float64            // Range: 0.0 (unbiased) to 1.0 (highly biased)
	Emotional       float64            // Range: 0.0 (not emotional) to 1.0 (highly emotional)
	Manipulative    float64            // Range: 0.0 (not manipulative) to 1.0 (highly manipulative)
	JailbreakIntent float64            // Range: 0.0 (no jailbreak intent) to 1.0 (definite jailbreak)
	Keywords        map[string]float64 // Map of detected keywords and their confidence scores
}

// Config holds configuration options for the Analyzer.
type Config struct {
	NLPEnabled         bool
	Rules              []Rule
	AutoBlockThreshold float64
}

// Result represents the outcome of an analysis.
type Result struct {
	Score        float64
	Reasons      []string   // Will contain descriptions of matched rules
	MatchedRules []Rule     // Store the actual rules that matched
	NLPMetrics   NLPMetrics // Detailed NLP metrics
}

// Analyzer provides various analysis capabilities for detecting potentially harmful content.
type Analyzer struct {
	config            Config
	rules             []Rule
	sensitiveKeywords []string
}

// New creates a new Analyzer instance with the provided configuration.
func New(config Config) (*Analyzer, error) {
	// Compile regex patterns for provided rules
	validRules := []Rule{}
	for _, rule := range config.Rules {
		// Fix: Regex patterns from JSON might need processing to handle escape sequences correctly
		// This is particularly important for \b word boundaries
		pattern := rule.Pattern

		re, err := regexp.Compile(pattern)
		if err != nil {
			log.Printf("Error compiling rule '%s' pattern '%s': %v. Skipping rule.", rule.Name, pattern, err)
			continue // Skip this rule
		}

		// Create a copy of the rule with the compiled regex
		ruleCopy := rule
		ruleCopy.compiled = re
		validRules = append(validRules, ruleCopy)
	}

	if len(validRules) == 0 {
		log.Println("Warning: No valid rules provided or compiled for the analyzer.")
	}

	return &Analyzer{
		config: config,
		rules:  validRules, // Use only valid, compiled rules
		sensitiveKeywords: []string{ // Keep sensitive keywords check for now
			"password", "credit card", "social security", "private", "classified", "illegal",
			"secret", "confidential", "personal", "ssn", "cvv", "banking", "authentication",
			"access code", "credentials", "hack", "exploit", "bypass", "security", "token",
		},
	}, nil
}

// AnalyzeText evaluates text for potential policy violations or harmful content based on loaded rules.
func (a *Analyzer) AnalyzeText(text string) Result {
	if !a.config.NLPEnabled {
		return Result{Score: 0, Reasons: []string{}, MatchedRules: []Rule{}, NLPMetrics: NLPMetrics{}}
	}

	var reasons []string
	var matchedRules []Rule
	var score float64

	// Apply loaded rules
	for _, rule := range a.rules {
		if rule.compiled != nil && rule.compiled.MatchString(text) {
			reason := fmt.Sprintf("Matched rule '%s' (Severity: %s): %s", rule.Name, rule.Severity, rule.Description)
			reasons = append(reasons, reason)
			matchedRules = append(matchedRules, rule)
			switch strings.ToLower(rule.Severity) {
			case "high":
				score += 0.5
			case "medium":
				score += 0.3
			case "low":
				score += 0.1
			}
		}
	}

	// Check for sensitive keywords
	lowercaseText := strings.ToLower(text)
	for _, keyword := range a.sensitiveKeywords {
		if strings.Contains(lowercaseText, keyword) {
			reason := "Contains sensitive keyword: " + keyword
			if !containsString(reasons, reason) {
				reasons = append(reasons, reason)
				score += 0.2
			}
		}
	}

	// Cap score at 1.0
	if score > 1.0 {
		score = 1.0
	}

	// Perform NLP Analysis
	nlpMetrics := a.analyzeNLP(text)

	// Update score based on NLP metrics
	if nlpMetrics.Toxicity > 0.7 {
		reasons = append(reasons, fmt.Sprintf("High toxicity detected: %.2f", nlpMetrics.Toxicity))
		score = math.Max(score, nlpMetrics.Toxicity)
	}

	if nlpMetrics.JailbreakIntent > 0.6 {
		reasons = append(reasons, fmt.Sprintf("Jailbreak intent detected: %.2f", nlpMetrics.JailbreakIntent))
		score = math.Max(score, nlpMetrics.JailbreakIntent)
	}

	if nlpMetrics.PII > 0.8 {
		reasons = append(reasons, fmt.Sprintf("PII detected: %.2f", nlpMetrics.PII))
		score = math.Max(score, 0.8)
	}

	return Result{
		Score:        score,
		Reasons:      reasons,
		MatchedRules: matchedRules,
		NLPMetrics:   nlpMetrics,
	}
}

// analyzeNLP performs NLP-based analysis on the text
func (a *Analyzer) analyzeNLP(text string) NLPMetrics {
	metrics := NLPMetrics{
		Keywords: make(map[string]float64),
	}

	lowercaseText := strings.ToLower(text)

	// Simple sentiment analysis based on keyword matching
	metrics.Sentiment = a.analyzeSentiment(lowercaseText)

	// Simple toxicity analysis
	metrics.Toxicity = a.analyzeToxicity(lowercaseText)

	// PII detection
	metrics.PII = a.analyzePII(lowercaseText)

	// Profanity check
	metrics.Profanity = a.analyzeProfanity(lowercaseText)

	// Bias detection
	metrics.Bias = a.analyzeBias(lowercaseText)

	// Emotional content analysis
	metrics.Emotional = a.analyzeEmotional(lowercaseText)

	// Manipulative content detection
	metrics.Manipulative = a.analyzeManipulative(lowercaseText)

	// Jailbreak intent detection
	metrics.JailbreakIntent = a.analyzeJailbreakIntent(lowercaseText)

	// Extract keywords
	metrics.Keywords = a.extractKeywords(lowercaseText)

	return metrics
}

// analyzeSentiment provides a basic sentiment score
func (a *Analyzer) analyzeSentiment(text string) float64 {
	positiveWords := []string{"good", "great", "excellent", "wonderful", "happy", "positive", "best", "love", "like", "helpful", "useful"}
	negativeWords := []string{"bad", "terrible", "awful", "horrible", "sad", "negative", "worst", "hate", "dislike", "useless", "harmful", "angry", "upset"}

	positiveCount := 0
	negativeCount := 0

	for _, word := range positiveWords {
		positiveCount += strings.Count(text, word)
	}

	for _, word := range negativeWords {
		negativeCount += strings.Count(text, word)
	}

	totalWords := len(strings.Fields(text))
	if totalWords == 0 {
		return 0.0
	}

	// Calculate sentiment from -1 to 1
	if positiveCount == 0 && negativeCount == 0 {
		return 0.0 // Neutral
	}

	return float64(positiveCount-negativeCount) / float64(positiveCount+negativeCount)
}

// analyzeToxicity detects toxic content
func (a *Analyzer) analyzeToxicity(text string) float64 {
	toxicWords := []string{"idiot", "stupid", "dumb", "retard", "moron", "loser", "kill", "die", "attack", "destroy", "hate", "violent", "death"}

	toxicCount := 0
	for _, word := range toxicWords {
		toxicCount += strings.Count(text, word)
	}

	totalWords := len(strings.Fields(text))
	if totalWords == 0 {
		return 0.0
	}

	// Scale the result to a 0-1 range
	return math.Min(1.0, float64(toxicCount)*5.0/float64(totalWords))
}

// analyzePII detects potential personally identifiable information
func (a *Analyzer) analyzePII(text string) float64 {
	piiPatterns := []*regexp.Regexp{
		// Basic email pattern
		regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`),
		// Phone number patterns
		regexp.MustCompile(`\b\d{3}[-.]?\d{3}[-.]?\d{4}\b`),
		// SSN-like patterns
		regexp.MustCompile(`\b\d{3}[-]?\d{2}[-]?\d{4}\b`),
		// Credit card-like patterns
		regexp.MustCompile(`\b(?:\d{4}[-\s]?){4}|\d{4}[-\s]?\d{6}[-\s]?\d{5}\b`),
		// Address-like patterns
		regexp.MustCompile(`\b\d+\s+[a-zA-Z]+\s+(?:st|ave|rd|blvd|drive|street|avenue|road|boulevard)\b`),
		// URL-like patterns
		regexp.MustCompile(`https?://[^\s/$.?#].[^\s]*`),
	}

	piiCount := 0
	for _, pattern := range piiPatterns {
		matches := pattern.FindAllString(text, -1)
		piiCount += len(matches)
	}

	// Additional PII keyword check
	piiKeywords := []string{"address", "password", "social security", "ssn", "credit card", "secret", "private", "phone number"}
	for _, keyword := range piiKeywords {
		if strings.Contains(text, keyword) {
			piiCount++
		}
	}

	// Scale the result
	return math.Min(1.0, float64(piiCount)*0.25)
}

// analyzeProfanity checks for profane language
func (a *Analyzer) analyzeProfanity(text string) float64 {
	// Simple list of profane words - in production would use a more comprehensive list
	profaneWords := []string{"fuck", "shit", "ass", "damn", "bitch", "cunt", "dick", "bastard"}

	profanityCount := 0
	for _, word := range profaneWords {
		profanityCount += strings.Count(text, word)
	}

	totalWords := len(strings.Fields(text))
	if totalWords == 0 {
		return 0.0
	}

	// Scale the result
	return math.Min(1.0, float64(profanityCount)*8.0/float64(totalWords))
}

// analyzeBias detects potential biased language
func (a *Analyzer) analyzeBias(text string) float64 {
	biasedPhrases := []string{
		"all men", "all women", "those people", "you people", "typical", "always", "never",
		"everyone knows", "obviously", "clearly", "all immigrants", "all conservatives", "all liberals",
		"those immigrants", "those minorities", "all muslims", "all christians", "all jews",
	}

	biasCount := 0
	for _, phrase := range biasedPhrases {
		biasCount += strings.Count(text, phrase)
	}

	totalWords := len(strings.Fields(text))
	if totalWords == 0 {
		return 0.0
	}

	// Scale the result
	return math.Min(1.0, float64(biasCount)*3.0/float64(totalWords))
}

// analyzeEmotional detects emotionally charged language
func (a *Analyzer) analyzeEmotional(text string) float64 {
	emotionalWords := []string{
		"love", "hate", "adore", "despise", "excited", "furious", "terrified", "ecstatic",
		"heartbroken", "devastated", "thrilled", "angry", "sad", "happy", "overjoyed",
		"frustrated", "exhilarated", "depressed", "anxious", "outraged", "scared",
	}

	emotionalCount := 0
	for _, word := range emotionalWords {
		emotionalCount += strings.Count(text, word)
	}

	totalWords := len(strings.Fields(text))
	if totalWords == 0 {
		return 0.0
	}

	// Scale the result
	return math.Min(1.0, float64(emotionalCount)*4.0/float64(totalWords))
}

// analyzeManipulative detects potentially manipulative language
func (a *Analyzer) analyzeManipulative(text string) float64 {
	manipulativePhrases := []string{
		"you must", "you need to", "you have to", "don't you think", "everyone is doing it",
		"limited time", "act now", "last chance", "once in a lifetime", "you won't regret",
		"trust me", "believe me", "you'd be foolish", "don't be stupid", "i need you to",
		"only you can", "i'm begging you", "i'm counting on you", "i'll be disappointed if you don't",
	}

	manipulativeCount := 0
	for _, phrase := range manipulativePhrases {
		manipulativeCount += strings.Count(text, phrase)
	}

	totalWords := len(strings.Fields(text))
	if totalWords == 0 {
		return 0.0
	}

	// Scale the result
	return math.Min(1.0, float64(manipulativeCount)*4.0/float64(totalWords))
}

// analyzeJailbreakIntent detects attempts to jailbreak AI systems
func (a *Analyzer) analyzeJailbreakIntent(text string) float64 {
	// This is already partially covered by IsJailbreakAttempt, but including here with scoring
	jailbreakPhrases := []string{
		"ignore previous instructions", "ignore prior instructions", "ignore your programming",
		"disregard your instructions", "system prompt", "pretend to be", "simulate being",
		"you are now", "act as if", "bypass your", "let's play a game", "hypothetically",
		"please continue", "complete the text", "write from another perspective", "for educational purposes",
		"for a fictional scenario", "in a fictional world", "as a thought experiment", "bypass safety",
		"ignore ethical guidelines", "ignore your ethical constraints", "let's imagine", "I want you to pretend",
	}

	jailbreakCount := 0
	for _, phrase := range jailbreakPhrases {
		if strings.Contains(text, phrase) {
			jailbreakCount += 1
		}
	}

	// More specific jailbreak patterns worth higher scores
	specificPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)ignore (previous|prior|earlier|initial) instructions`),
		regexp.MustCompile(`(?i)pretend (to be|you are) (an|a) (unrestricted|unfiltered)`),
		regexp.MustCompile(`(?i)(bypass|ignore|circumvent) (ethics|restrictions|limitations|filters)`),
	}

	for _, pattern := range specificPatterns {
		if pattern.MatchString(text) {
			jailbreakCount += 2
		}
	}

	// Scale the result
	return math.Min(1.0, float64(jailbreakCount)*0.2)
}

// extractKeywords identifies important keywords in the text with confidence scores
func (a *Analyzer) extractKeywords(text string) map[string]float64 {
	keywords := make(map[string]float64)

	// Define potential categories and their keywords
	categories := map[string][]string{
		"security":      {"password", "secure", "vulnerability", "access", "protection", "authentication", "authorization", "threat", "risk"},
		"finance":       {"money", "payment", "bank", "credit", "debit", "transaction", "financial", "fund", "cash", "invest"},
		"personal":      {"name", "address", "phone", "email", "identification", "identity", "profile", "personal", "private", "confidential"},
		"harmful":       {"hack", "exploit", "attack", "damage", "destroy", "harm", "dangerous", "malicious", "virus", "malware"},
		"sensitive":     {"secret", "classified", "private", "confidential", "restricted", "sensitive", "hidden", "undisclosed", "internal"},
		"promptHacking": {"prompt", "instruction", "command", "directive", "forget", "ignore", "bypass", "override", "disregard", "pretend"},
	}

	words := strings.Fields(strings.ToLower(text))
	wordCount := make(map[string]int)

	// Count word frequencies
	for _, word := range words {
		// Clean the word of punctuation
		word = strings.Trim(word, ".,!?;:\"'()[]{}")
		if len(word) > 3 { // Only consider words of reasonable length
			wordCount[word]++
		}
	}

	// Assign keywords to categories with confidence scores
	for category, categoryWords := range categories {
		for _, keyword := range categoryWords {
			if count, found := wordCount[keyword]; found {
				confidence := math.Min(1.0, float64(count)*0.3)
				keywords[category+":"+keyword] = confidence
			}
		}
	}

	return keywords
}

// ShouldBlock determines if the content should be blocked based on analysis results.
func (a *Analyzer) ShouldBlock(result Result) bool {
	// Blocking could be based on score or specific high-severity rule matches
	if result.Score >= a.config.AutoBlockThreshold {
		return true
	}

	// Block if certain NLP metrics exceed thresholds
	if result.NLPMetrics.Toxicity > 0.9 ||
		result.NLPMetrics.JailbreakIntent > 0.85 ||
		result.NLPMetrics.PII > 0.9 {
		return true
	}

	return false
}

// IsJailbreakAttempt specifically checks for patterns indicative of jailbreak attempts.
func (a *Analyzer) IsJailbreakAttempt(text string) bool {
	// This could also use rules tagged with a specific category like 'jailbreak'
	jailbreakPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)ignore (previous|prior) instructions`),
		regexp.MustCompile(`(?i)\b(system prompt|ignore previous instructions|my previous instructions|my prior instructions)\b`),
		regexp.MustCompile(`(?i)\b(pretend|imagine|role-play|simulation).+?(ignore|forget|disregard).+?(instruction|prompt|rule)`),
		regexp.MustCompile(`(?i)\b(let's play a game|hypothetically speaking|in a fictional scenario)\b`),
		regexp.MustCompile(`(?i)bypass (safety|security|ethical|filter)`),
		regexp.MustCompile(`(?i)how (to|would|could) (hack|steal|attack|exploit)`),
	}

	for _, pattern := range jailbreakPatterns {
		if pattern.MatchString(text) {
			return true
		}
	}

	return false
}

// Helper function to check if a slice contains a string
func containsString(slice []string, str string) bool {
	for _, item := range slice {
		if item == str {
			return true
		}
	}
	return false
}
