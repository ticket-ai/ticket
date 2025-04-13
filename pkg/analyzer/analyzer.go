// Package analyzer provides text analysis capabilities for identifying policy violations, jailbreak attempts,
// and other potentially harmful content in AI interactions.
package analyzer

import (
	"regexp"
	"strings"
)

// Config holds configuration options for the Analyzer.
type Config struct {
	NLPEnabled         bool
	StaticRules        []string
	AutoBlockThreshold float64
}

// Result represents the outcome of an analysis.
type Result struct {
	Score   float64
	Reasons []string
}

// Analyzer provides various analysis capabilities for detecting potentially harmful content.
type Analyzer struct {
	config            Config
	staticRules       []*regexp.Regexp
	sensitiveKeywords []string
}

// New creates a new Analyzer instance with the provided configuration.
func New(config Config) (*Analyzer, error) {
	// Compile static analysis rules
	var rules []*regexp.Regexp
	for _, pattern := range config.StaticRules {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, err
		}
		rules = append(rules, re)
	}

	return &Analyzer{
		config:      config,
		staticRules: rules,
		sensitiveKeywords: []string{
			"hack", "vulnerability", "exploit", "bypass", "illegal",
			"password", "credit card", "social security", "private", "classified",
		},
	}, nil
}

// AnalyzeText evaluates text for potential policy violations or harmful content.
func (a *Analyzer) AnalyzeText(text string) Result {
	if !a.config.NLPEnabled {
		return Result{Score: 0, Reasons: []string{}}
	}

	var reasons []string
	var score float64

	// Apply static analysis rules
	for i, rule := range a.staticRules {
		if rule.MatchString(text) {
			reasons = append(reasons,
				"Matched static pattern "+string(i+1))
			score += 0.3 // Increase suspicion score for each match
		}
	}

	// Check for sensitive keywords
	lowercaseText := strings.ToLower(text)
	for _, keyword := range a.sensitiveKeywords {
		if strings.Contains(lowercaseText, keyword) {
			reasons = append(reasons,
				"Contains sensitive keyword: "+keyword)
			score += 0.2
		}
	}

	// Cap score at 1.0
	if score > 1.0 {
		score = 1.0
	}

	return Result{
		Score:   score,
		Reasons: reasons,
	}
}

// ShouldBlock determines if the content should be blocked based on analysis results.
func (a *Analyzer) ShouldBlock(result Result) bool {
	return result.Score >= a.config.AutoBlockThreshold
}

// IsJailbreakAttempt specifically checks for patterns indicative of jailbreak attempts.
func (a *Analyzer) IsJailbreakAttempt(text string) bool {
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
