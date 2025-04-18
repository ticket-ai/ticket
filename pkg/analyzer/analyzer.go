// Package analyzer provides text analysis capabilities for identifying policy violations, jailbreak attempts,
// and other potentially harmful content in AI interactions.
package analyzer

import (
	"fmt"
	"log"
	"regexp"
	"strings"
)

// Rule defines the structure for a single rule.
type Rule struct {
	Name        string `yaml:"name" json:"name"`               // Added json tag
	Pattern     string `yaml:"pattern" json:"pattern"`         // Added json tag
	Severity    string `yaml:"severity" json:"severity"`       // e.g., "high", "medium", "low", Added json tag
	Description string `yaml:"description" json:"description"` // Added json tag
	compiled    *regexp.Regexp
}

// Config holds configuration options for the Analyzer.
type Config struct {
	NLPEnabled         bool
	Rules              []Rule // Rules are now passed directly
	AutoBlockThreshold float64
}

// Result represents the outcome of an analysis.
type Result struct {
	Score        float64
	Reasons      []string // Will contain descriptions of matched rules
	MatchedRules []Rule   // Store the actual rules that matched
}

// Analyzer provides various analysis capabilities for detecting potentially harmful content.
type Analyzer struct {
	config            Config
	rules             []Rule // Use the rules from the config
	sensitiveKeywords []string
}

// New creates a new Analyzer instance with the provided configuration.
func New(config Config) (*Analyzer, error) {
	// Compile regex patterns for provided rules
	validRules := []Rule{}
	for _, rule := range config.Rules { // Iterate over rules from config
		re, err := regexp.Compile(rule.Pattern)
		if err != nil {
			log.Printf("Error compiling rule '%s' pattern '%s': %v. Skipping rule.", rule.Name, rule.Pattern, err)
			continue // Skip this rule
		}
		rule.compiled = re
		validRules = append(validRules, rule)
	}

	if len(validRules) == 0 {
		log.Println("Warning: No valid rules provided or compiled for the analyzer.")
	}

	return &Analyzer{
		config: config,
		rules:  validRules, // Use only valid, compiled rules
		sensitiveKeywords: []string{ // Keep sensitive keywords check for now
			"password", "credit card", "social security", "private", "classified", "illegal",
		},
	}, nil
}

// AnalyzeText evaluates text for potential policy violations or harmful content based on loaded rules.
func (a *Analyzer) AnalyzeText(text string) Result {
	if !a.config.NLPEnabled {
		return Result{Score: 0, Reasons: []string{}, MatchedRules: []Rule{}}
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

	return Result{
		Score:        score,
		Reasons:      reasons,
		MatchedRules: matchedRules,
	}
}

// ShouldBlock determines if the content should be blocked based on analysis results.
func (a *Analyzer) ShouldBlock(result Result) bool {
	// Blocking could be based on score or specific high-severity rule matches
	if result.Score >= a.config.AutoBlockThreshold {
		return true
	}
	// Optionally block immediately on any high severity match, regardless of threshold
	// for _, rule := range result.MatchedRules {
	// 	if strings.ToLower(rule.Severity) == "high" {
	// 		return true
	// 	}
	// }
	return false
}

// IsJailbreakAttempt specifically checks for patterns indicative of jailbreak attempts.
// Note: This logic might be better integrated into the rule system itself.
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
