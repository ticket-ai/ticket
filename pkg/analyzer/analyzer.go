// Package analyzer provides text analysis capabilities for identifying policy violations, jailbreak attempts,
// and other potentially harmful content in AI interactions.
package analyzer

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3" // Added import for YAML parsing
)

// Rule defines the structure for a single rule in the YAML file.
type Rule struct {
	Name        string `yaml:"name"`
	Pattern     string `yaml:"pattern"`
	Severity    string `yaml:"severity"` // e.g., "high", "medium", "low"
	Description string `yaml:"description"`
	compiled    *regexp.Regexp
}

// Config holds configuration options for the Analyzer.
type Config struct {
	NLPEnabled         bool
	RulesFile          string // Path to the rules YAML file
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
	rules             []Rule // Changed from staticRules to loaded rules
	sensitiveKeywords []string
}

// DefaultRules provides a set of default rules if no YAML file is specified or found.
func DefaultRules() []Rule {
	return []Rule{
		{
			Name:        "Default Ignore Instructions",
			Pattern:     `(?i)ignore (previous|prior) instructions`,
			Severity:    "medium",
			Description: "Attempt to make the model ignore previous instructions.",
		},
		{
			Name:        "Default Pretend/Ignore",
			Pattern:     `(?i)\b(pretend|imagine|role-play|simulation).+?(ignore|forget|disregard).+?(instruction|prompt|rule)`,
			Severity:    "medium",
			Description: "Attempt to use role-playing to bypass rules.",
		},
		{
			Name:        "Default Hypothetical Bypass",
			Pattern:     `(?i)\b(let's play a game|hypothetically speaking|in a fictional scenario)\b`,
			Severity:    "low",
			Description: "Using hypothetical scenarios, potentially to bypass safety.",
		},
		{
			Name:        "Default Hacking Keywords",
			Pattern:     `(?i)\b(hack|bypass security|exploit|vulnerability)\b`,
			Severity:    "high",
			Description: "Keywords related to attempting to hack or bypass security.",
		},
	}
}

// New creates a new Analyzer instance with the provided configuration.
func New(config Config) (*Analyzer, error) {
	var loadedRules []Rule

	// Load rules from YAML file if specified
	if config.RulesFile != "" {
		yamlFile, err := os.ReadFile(config.RulesFile)
		if err != nil {
			log.Printf("Warning: Could not read rules file '%s', using default rules. Error: %v", config.RulesFile, err)
			loadedRules = DefaultRules()
		} else {
			var ruleConfig struct {
				Rules []Rule `yaml:"rules"`
			}
			err = yaml.Unmarshal(yamlFile, &ruleConfig)
			if err != nil {
				log.Printf("Warning: Could not parse rules file '%s', using default rules. Error: %v", config.RulesFile, err)
				loadedRules = DefaultRules()
			} else {
				log.Printf("Successfully loaded %d rules from %s", len(ruleConfig.Rules), config.RulesFile)
				loadedRules = ruleConfig.Rules
			}
		}
	} else {
		log.Println("No rules file specified, using default rules.")
		loadedRules = DefaultRules()
	}

	// Compile regex patterns for loaded rules
	for i := range loadedRules {
		re, err := regexp.Compile(loadedRules[i].Pattern)
		if err != nil {
			log.Printf("Error compiling rule '%s' pattern '%s': %v. Skipping rule.", loadedRules[i].Name, loadedRules[i].Pattern, err)
			loadedRules[i].compiled = nil
		} else {
			loadedRules[i].compiled = re
		}
	}

	// Filter out rules that failed to compile
	validRules := []Rule{}
	for _, rule := range loadedRules {
		if rule.compiled != nil {
			validRules = append(validRules, rule)
		}
	}

	return &Analyzer{
		config: config,
		rules:  validRules,
		sensitiveKeywords: []string{
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
