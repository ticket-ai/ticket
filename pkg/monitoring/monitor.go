// Package monitoring provides real-time monitoring and alerting capabilities for AI applications.
package monitoring

import (
	"fmt"
	"sync"
	"time"
)

// Config holds configuration options for the monitoring component.
type Config struct {
	ServiceName string
	Environment string
}

// FlaggedUser represents a user who has been flagged for suspicious behavior.
type FlaggedUser struct {
	UserID    string
	Score     float64
	FlaggedAt time.Time
	Reasons   []string
}

// Monitor provides capabilities for monitoring and alerting on suspicious behavior.
type Monitor struct {
	config       Config
	blockedIPs   map[string]time.Time
	flaggedUsers map[string]FlaggedUser
	mu           sync.RWMutex
}

// New creates a new Monitor instance.
func New(config Config) *Monitor {
	return &Monitor{
		config:       config,
		blockedIPs:   make(map[string]time.Time),
		flaggedUsers: make(map[string]FlaggedUser),
		mu:           sync.RWMutex{},
	}
}

// BlockIP adds an IP address to the blocked list.
func (m *Monitor) BlockIP(ip string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.blockedIPs[ip] = time.Now()
	fmt.Printf("[%s] Blocked IP: %s\n", time.Now().Format(time.RFC3339), ip)
}

// IsIPBlocked checks if an IP address is blocked.
func (m *Monitor) IsIPBlocked(ip string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, blocked := m.blockedIPs[ip]
	return blocked
}

// FlagUser marks a user as suspicious.
func (m *Monitor) FlagUser(userID string, score float64, reasons []string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if user is already flagged and update score if higher
	if existing, exists := m.flaggedUsers[userID]; exists {
		if score > existing.Score {
			existing.Score = score
			existing.Reasons = append(existing.Reasons, reasons...)
			m.flaggedUsers[userID] = existing
		}
		return
	}

	// Otherwise add new flagged user
	m.flaggedUsers[userID] = FlaggedUser{
		UserID:    userID,
		Score:     score,
		FlaggedAt: time.Now(),
		Reasons:   reasons,
	}

	fmt.Printf("[%s] Flagged user: %s with score %.2f\n",
		time.Now().Format(time.RFC3339), userID, score)
}

// GetFlaggedUsers returns all flagged users.
func (m *Monitor) GetFlaggedUsers() []FlaggedUser {
	m.mu.RLock()
	defer m.mu.RUnlock()

	users := make([]FlaggedUser, 0, len(m.flaggedUsers))
	for _, user := range m.flaggedUsers {
		users = append(users, user)
	}
	return users
}

// GetBlockedIPs returns all blocked IP addresses.
func (m *Monitor) GetBlockedIPs() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ips := make([]string, 0, len(m.blockedIPs))
	for ip := range m.blockedIPs {
		ips = append(ips, ip)
	}
	return ips
}

// ClearFlaggedUser removes a user from the flagged list.
func (m *Monitor) ClearFlaggedUser(userID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.flaggedUsers, userID)
}
