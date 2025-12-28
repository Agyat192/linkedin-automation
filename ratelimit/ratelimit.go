package ratelimit

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// RateLimiter implements intelligent rate limiting for LinkedIn automation
type RateLimiter struct {
	logger           *logrus.Logger
	config           Config
	lastActionTime   map[string]time.Time
	actionCounts     map[string]int
	dailyCounts      map[string]int
	mu               sync.RWMutex
	dailyResetTime   time.Time
}

// Config defines rate limiting behavior
type Config struct {
	// General delays
	MinDelay       time.Duration `yaml:"min_delay"`        // Minimum delay between actions
	MaxDelay       time.Duration `yaml:"max_delay"`        // Maximum delay between actions
	
	// Action-specific limits
	SearchDelay    time.Duration `yaml:"search_delay"`     // Delay between searches
	ConnectDelay   time.Duration `yaml:"connect_delay"`    // Delay between connection requests
	MessageDelay   time.Duration `yaml:"message_delay"`    // Delay between messages
	
	// Daily limits
	DailySearches  int           `yaml:"daily_searches"`   // Max searches per day
	DailyConnects  int           `yaml:"daily_connects"`   // Max connection requests per day
	DailyMessages  int           `yaml:"daily_messages"`   // Max messages per day
	
	// Hourly limits
	HourlySearches int           `yaml:"hourly_searches"`  // Max searches per hour
	HourlyConnects int           `yaml:"hourly_connects"`  // Max connection requests per hour
	HourlyMessages int           `yaml:"hourly_messages"`  // Max messages per hour
	
	// Burst protection
	BurstLimit     int           `yaml:"burst_limit"`      // Max actions in burst window
	BurstWindow    time.Duration `yaml:"burst_window"`    // Time window for burst detection
	
	// Humanization
	RandomizeDelay bool          `yaml:"randomize_delay"`  // Add randomness to delays
	JitterPercent  float64       `yaml:"jitter_percent"`   // Percentage of jitter to add
}

// ActionType represents different types of LinkedIn actions
type ActionType string

const (
	ActionSearch  ActionType = "search"
	ActionConnect ActionType = "connect"
	ActionMessage ActionType = "message"
	ActionBrowse  ActionType = "browse"
)

// NewRateLimiter creates a new rate limiter with the given configuration
func NewRateLimiter(config Config, logger *logrus.Logger) *RateLimiter {
	rl := &RateLimiter{
		logger:         logger,
		config:         config,
		lastActionTime: make(map[string]time.Time),
		actionCounts:   make(map[string]int),
		dailyCounts:    make(map[string]int),
		dailyResetTime: getNextMidnight(),
	}
	
	// Start daily reset goroutine
	go rl.dailyReset()
	
	return rl
}

// WaitForPermission waits until the action can be performed
func (rl *RateLimiter) WaitForPermission(ctx context.Context, action ActionType) error {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	// Check daily limits
	if err := rl.checkDailyLimits(action); err != nil {
		return err
	}
	
	// Check hourly limits
	if err := rl.checkHourlyLimits(action); err != nil {
		return err
	}
	
	// Check burst protection
	if err := rl.checkBurstProtection(action); err != nil {
		return err
	}
	
	// Calculate required delay
	delay := rl.calculateDelay(action)
	
	// Add humanization
	if rl.config.RandomizeDelay {
		delay = rl.addJitter(delay)
	}
	
	// Wait if needed
	if delay > 0 {
		rl.logger.WithFields(logrus.Fields{
			"action": string(action),
			"delay":  delay,
		}).Info("Rate limiting - waiting")
		
		select {
		case <-time.After(delay):
			// Continue
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	
	// Update tracking
	rl.updateTracking(action)
	
	return nil
}

// checkDailyLimits ensures we don't exceed daily quotas
func (rl *RateLimiter) checkDailyLimits(action ActionType) error {
	actionStr := string(action)
	
	var dailyLimit int
	switch action {
	case ActionSearch:
		dailyLimit = rl.config.DailySearches
	case ActionConnect:
		dailyLimit = rl.config.DailyConnects
	case ActionMessage:
		dailyLimit = rl.config.DailyMessages
	default:
		return nil // No daily limit for other actions
	}
	
	if dailyLimit > 0 {
		current := rl.dailyCounts[actionStr]
		if current >= dailyLimit {
			return fmt.Errorf("daily limit exceeded for %s: %d/%d", actionStr, current, dailyLimit)
		}
	}
	
	return nil
}

// checkHourlyLimits ensures we don't exceed hourly quotas
func (rl *RateLimiter) checkHourlyLimits(action ActionType) error {
	actionStr := string(action)
	
	var hourlyLimit int
	switch action {
	case ActionSearch:
		hourlyLimit = rl.config.HourlySearches
	case ActionConnect:
		hourlyLimit = rl.config.HourlyConnects
	case ActionMessage:
		hourlyLimit = rl.config.HourlyMessages
	default:
		return nil // No hourly limit for other actions
	}
	
	if hourlyLimit > 0 {
		// Count actions in the last hour
		// This is a simplified implementation - in production, you'd want a sliding window
		// For now, we'll use the actionCounts which reset hourly
		current := rl.actionCounts[actionStr]
		if current >= hourlyLimit {
			return fmt.Errorf("hourly limit exceeded for %s: %d/%d", actionStr, current, hourlyLimit)
		}
	}
	
	return nil
}

// checkBurstProtection prevents rapid successive actions
func (rl *RateLimiter) checkBurstProtection(action ActionType) error {
	if rl.config.BurstLimit <= 0 {
		return nil
	}
	
	actionStr := string(action)
	lastAction := rl.lastActionTime[actionStr]
	
	if !lastAction.IsZero() {
		timeSinceLast := time.Since(lastAction)
		if timeSinceLast < rl.config.BurstWindow {
			// We're in the burst window
			if rl.actionCounts[actionStr] >= rl.config.BurstLimit {
				return fmt.Errorf("burst limit exceeded for %s: %d actions in %v", 
					actionStr, rl.actionCounts[actionStr], rl.config.BurstWindow)
			}
		}
	}
	
	return nil
}

// calculateDelay determines how long to wait before the next action
func (rl *RateLimiter) calculateDelay(action ActionType) time.Duration {
	actionStr := string(action)
	lastAction := rl.lastActionTime[actionStr]
	
	if lastAction.IsZero() {
		return 0 // First action, no delay
	}
	
	timeSinceLast := time.Since(lastAction)
	
	// Get action-specific delay
	var requiredDelay time.Duration
	switch action {
	case ActionSearch:
		requiredDelay = rl.config.SearchDelay
	case ActionConnect:
		requiredDelay = rl.config.ConnectDelay
	case ActionMessage:
		requiredDelay = rl.config.MessageDelay
	default:
		requiredDelay = rl.config.MinDelay
	}
	
	// Use the maximum of action-specific delay and general delay
	if rl.config.MinDelay > requiredDelay {
		requiredDelay = rl.config.MinDelay
	}
	
	// Calculate remaining wait time
	if timeSinceLast >= requiredDelay {
		return 0 // Enough time has passed
	}
	
	return requiredDelay - timeSinceLast
}

// addJitter adds randomness to delays for humanization
func (rl *RateLimiter) addJitter(delay time.Duration) time.Duration {
	if rl.config.JitterPercent <= 0 {
		return delay
	}
	
	// Add +/- jitter_percent randomness
	jitter := float64(delay) * rl.config.JitterPercent / 100.0
	randomJitter := (rand.Float64()*2 - 1) * jitter // Can be positive or negative
	
	newDelay := float64(delay) + randomJitter
	if newDelay < 0 {
		newDelay = 0
	}
	
	return time.Duration(newDelay)
}

// updateTracking updates the internal tracking for rate limiting
func (rl *RateLimiter) updateTracking(action ActionType) {
	actionStr := string(action)
	now := time.Now()
	
	// Update last action time
	rl.lastActionTime[actionStr] = now
	
	// Increment action counts
	rl.actionCounts[actionStr]++
	rl.dailyCounts[actionStr]++
	
	// Start hourly reset goroutine if not already running
	go rl.hourlyReset(actionStr)
}

// hourlyReset resets hourly counts for an action type
func (rl *RateLimiter) hourlyReset(action string) {
	time.Sleep(time.Hour)
	
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	rl.actionCounts[action] = 0
}

// dailyReset resets daily counts at midnight
func (rl *RateLimiter) dailyReset() {
	for {
		now := time.Now()
		nextReset := getNextMidnight()
		sleepDuration := nextReset.Sub(now)
		
		time.Sleep(sleepDuration)
		
		rl.mu.Lock()
		rl.dailyCounts = make(map[string]int) // Reset all daily counts
		rl.dailyResetTime = getNextMidnight()
		rl.mu.Unlock()
		
		rl.logger.Info("Daily rate limits reset")
	}
}

// getNextMidnight returns the next midnight time
func getNextMidnight() time.Time {
	now := time.Now()
	midnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
	return midnight
}

// GetStats returns current rate limiting statistics
func (rl *RateLimiter) GetStats() map[string]interface{} {
	rl.mu.RLock()
	defer rl.mu.RUnlock()
	
	stats := make(map[string]interface{})
	
	// Daily counts
	stats["daily_searches"] = rl.dailyCounts[string(ActionSearch)]
	stats["daily_connects"] = rl.dailyCounts[string(ActionConnect)]
	stats["daily_messages"] = rl.dailyCounts[string(ActionMessage)]
	
	// Last action times
	for action, lastTime := range rl.lastActionTime {
		stats["last_"+action] = lastTime.Format(time.RFC3339)
	}
	
	// Next reset time
	stats["next_daily_reset"] = rl.dailyResetTime.Format(time.RFC3339)
	
	return stats
}

// DefaultConfig returns a sensible default configuration
func DefaultConfig() Config {
	return Config{
		MinDelay:       2 * time.Second,
		MaxDelay:       10 * time.Second,
		SearchDelay:    5 * time.Second,
		ConnectDelay:   30 * time.Second,
		MessageDelay:   60 * time.Second,
		DailySearches:  100,
		DailyConnects:  50,
		DailyMessages:  30,
		HourlySearches: 20,
		HourlyConnects: 10,
		HourlyMessages: 5,
		BurstLimit:     3,
		BurstWindow:    30 * time.Second,
		RandomizeDelay: true,
		JitterPercent:  20.0,
	}
}
