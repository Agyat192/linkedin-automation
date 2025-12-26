package connect

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/sirupsen/logrus"
)

// ConnectManager handles connection requests
type ConnectManager struct {
	page      *rod.Page
	logger    *logrus.Logger
	stealth   StealthManager
}

// StealthManager interface for stealth operations
type StealthManager interface {
	HumanLikeMouseMove(page *rod.Page, fromX, fromY, toX, toY float64) error
	RandomDelay() time.Duration
	HumanLikeType(page *rod.Page, text string) error
	HumanLikeScroll(page *rod.Page, scrollAmount int) error
	AddIdleMovement(page *rod.Page) error
}

// ConnectionRequest represents a connection request
type ConnectionRequest struct {
	ProfileURL string
	Message    string
	Status     string
	SentAt     time.Time
}

// ConnectionResult represents the result of a connection attempt
type ConnectionResult struct {
	Success       bool
	ProfileURL     string
	ErrorMessage   string
	AlreadyConnected bool
	RequestSent    bool
	RequestID      string
}

// MessageTemplate represents a connection message template
type MessageTemplate struct {
	ID      string
	Name    string
	Content string
	Variables []string
}

// NewConnectManager creates a new connection manager
func NewConnectManager(page *rod.Page, logger *logrus.Logger, stealth StealthManager) *ConnectManager {
	return &ConnectManager{
		page:    page,
		logger:  logger,
		stealth: stealth,
	}
}

// SendConnectionRequest sends a connection request to a profile
func (c *ConnectManager) SendConnectionRequest(ctx context.Context, profileURL, message string) (*ConnectionResult, error) {
	c.logger.WithFields(logrus.Fields{
		"profile_url": profileURL,
		"has_message": message != "",
	}).Info("Sending connection request")

	result := &ConnectionResult{
		ProfileURL: profileURL,
	}

	// Navigate to profile
	if err := c.navigateToProfile(profileURL); err != nil {
		result.ErrorMessage = fmt.Sprintf("Failed to navigate to profile: %v", err)
		return result, err
	}

	// Check if already connected
	if connected, err := c.isAlreadyConnected(); err == nil && connected {
		result.AlreadyConnected = true
		result.Success = true
		c.logger.Info("Already connected to profile")
		return result, nil
	}

	// Find and click connect button
	if err := c.clickConnectButton(); err != nil {
		result.ErrorMessage = fmt.Sprintf("Failed to click connect button: %v", err)
		return result, err
	}

	// Add random delay
	time.Sleep(c.stealth.RandomDelay())

	// Handle connection dialog
	dialogResult, err := c.handleConnectionDialog(message)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Failed to handle connection dialog: %v", err)
		return result, err
	}

	result.RequestSent = dialogResult.Success
	result.Success = dialogResult.Success
	result.ErrorMessage = dialogResult.ErrorMessage

	c.logger.WithFields(logrus.Fields{
		"success":      result.Success,
		"request_sent": result.RequestSent,
		"already_connected": result.AlreadyConnected,
	}).Info("Connection request completed")

	return result, nil
}

// SendConnectionRequestWithTemplate sends a connection request using a template
func (c *ConnectManager) SendConnectionRequestWithTemplate(ctx context.Context, profileURL string, template MessageTemplate, variables map[string]string) (*ConnectionResult, error) {
	// Process template variables
	message := c.processTemplate(template.Content, variables)
	
	return c.SendConnectionRequest(ctx, profileURL, message)
}

// CheckConnectionStatus checks the connection status with a profile
func (c *ConnectManager) CheckConnectionStatus(ctx context.Context, profileURL string) (string, error) {
	c.logger.WithField("profile_url", profileURL).Debug("Checking connection status")

	if err := c.navigateToProfile(profileURL); err != nil {
		return "unknown", fmt.Errorf("failed to navigate to profile: %w", err)
	}

	// Check various connection status indicators
	if connected, err := c.isAlreadyConnected(); err == nil && connected {
		return "connected", nil
	}

	if pending, err := c.isRequestPending(); err == nil && pending {
		return "pending", nil
	}

	if notConnected, err := c.isNotConnected(); err == nil && notConnected {
		return "not_connected", nil
	}

	return "unknown", nil
}

// BatchSendConnectionRequests sends multiple connection requests
func (c *ConnectManager) BatchSendConnectionRequests(ctx context.Context, profiles []string, message string) ([]*ConnectionResult, error) {
	c.logger.WithField("count", len(profiles)).Info("Starting batch connection requests")

	results := make([]*ConnectionResult, 0, len(profiles))

	for i, profileURL := range profiles {
		c.logger.WithFields(logrus.Fields{
			"current": i + 1,
			"total":   len(profiles),
			"profile": profileURL,
		}).Debug("Processing profile")

		result, err := c.SendConnectionRequest(ctx, profileURL, message)
		if err != nil {
			c.logger.WithError(err).Error("Failed to send connection request")
		}

		results = append(results, result)

		// Add delay between requests
		if i < len(profiles)-1 {
			time.Sleep(c.stealth.RandomDelay())
			
			// Add idle movement
			if err := c.stealth.AddIdleMovement(c.page); err != nil {
				c.logger.WithError(err).Warn("Failed to add idle movement")
			}
		}
	}

	// Count results
	successCount := 0
	alreadyConnectedCount := 0
	for _, result := range results {
		if result.Success {
			successCount++
		}
		if result.AlreadyConnected {
			alreadyConnectedCount++
		}
	}

	c.logger.WithFields(logrus.Fields{
		"total": len(profiles),
		"success": successCount,
		"already_connected": alreadyConnectedCount,
	}).Info("Batch connection requests completed")

	return results, nil
}

// Private helper methods

func (c *ConnectManager) navigateToProfile(profileURL string) error {
	c.logger.WithField("url", profileURL).Debug("Navigating to profile")

	if err := c.page.Navigate(profileURL); err != nil {
		return fmt.Errorf("failed to navigate to profile: %w", err)
	}

	// Wait for page to load
	if err := c.page.WaitLoad(); err != nil {
		return fmt.Errorf("failed to wait for page load: %w", err)
	}

	// Wait for profile content to load
	if err := c.waitForProfileContent(); err != nil {
		return fmt.Errorf("failed to wait for profile content: %w", err)
	}

	return nil
}

func (c *ConnectManager) waitForProfileContent() error {
	selectors := []string{
		".pv-profile-wrapper",
		".profile-content",
		".pv-top-card",
		"[data-test-id='profile-wrapper']",
	}

	for _, selector := range selectors {
		element, err := c.page.Element(selector)
		if err == nil && element != nil {
			c.logger.WithField("selector", selector).Debug("Found profile content")
			return nil
		}
	}

	// Wait a bit and try again
	time.Sleep(2 * time.Second)
	
	for _, selector := range selectors {
		element, err := c.page.Element(selector)
		if err == nil && element != nil {
			c.logger.WithField("selector", selector).Debug("Found profile content after delay")
			return nil
		}
	}

	return fmt.Errorf("profile content not found")
}

func (c *ConnectManager) isAlreadyConnected() (bool, error) {
	selectors := []string{
		".pv-s-profile-actions--connect.mutual",
		"[data-test-id='profile-connect-button'][aria-label*='Connected']",
		".pv-s-profile-actions--message",
		"[data-test-id='profile-message-button']",
	}

	for _, selector := range selectors {
		element, err := c.page.Element(selector)
		if err == nil && element != nil {
			// Check if it indicates connection
			text, err := element.Text()
			if err == nil && (strings.Contains(text, "Message") || strings.Contains(text, "Connected")) {
				return true, nil
			}
		}
	}

	return false, nil
}

func (c *ConnectManager) isRequestPending() (bool, error) {
	selectors := []string{
		".pv-s-profile-actions--connect.pending",
		"[data-test-id='profile-connect-button'][aria-label*='Pending']",
		".pv-s-profile-actions--withdraw",
	}

	for _, selector := range selectors {
		element, err := c.page.Element(selector)
		if err == nil && element != nil {
			return true, nil
		}
	}

	return false, nil
}

func (c *ConnectManager) isNotConnected() (bool, error) {
	selectors := []string{
		".pv-s-profile-actions--connect:not(.pending):not(.mutual)",
		"[data-test-id='profile-connect-button']",
		".pvs-profile-actions__action",
	}

	for _, selector := range selectors {
		elements, err := c.page.Elements(selector)
		if err == nil && len(elements) > 0 {
			// Check if any element is a connect button
			for _, element := range elements {
				text, err := element.Text()
				if err == nil && strings.Contains(text, "Connect") {
					return true, nil
				}
			}
		}
	}

	return false, nil
}

func (c *ConnectManager) clickConnectButton() error {
	c.logger.Debug("Looking for connect button")

	// Try different selectors for connect button
	selectors := []string{
		".pv-s-profile-actions--connect",
		"[data-test-id='profile-connect-button']",
		".pvs-profile-actions__action",
		"button[aria-label*='Connect']",
		"button:contains('Connect')",
	}

	var connectButton *rod.Element
	var usedSelector string

	for _, selector := range selectors {
		element, err := c.page.Element(selector)
		if err == nil && element != nil {
			// Verify it's actually a connect button
			text, err := element.Text()
			if err == nil && strings.Contains(text, "Connect") {
				connectButton = element
				usedSelector = selector
				break
			}
		}
	}

	if connectButton == nil {
		return fmt.Errorf("connect button not found")
	}

	c.logger.WithField("selector", usedSelector).Debug("Found connect button")

	// Get button position for human-like mouse movement
	shape, err := connectButton.Shape()
	if err != nil {
		return fmt.Errorf("failed to get button position: %w", err)
	}
	box := shape.Box()

	// Get viewport size
	viewport, err := c.page.Eval("({width: window.innerWidth, height: window.innerHeight})")
	if err != nil {
		return fmt.Errorf("failed to get viewport: %w", err)
	}

	fromX := viewport.Value.Get("width").Num()
	fromY := viewport.Value.Get("height").Num()

	// Move mouse to button
	centerX := box.X + box.Width/2
	centerY := box.Y + box.Height/2

	// Human-like mouse movement
	if err := c.stealth.HumanLikeMouseMove(c.page, fromX, fromY, centerX, centerY); err != nil {
		c.logger.WithError(err).Warn("Failed to perform human-like mouse movement")
	}

	// Click connect button
	if err := connectButton.Click("left", 1); err != nil {
		return fmt.Errorf("failed to click connect button: %w", err)
	}

	c.logger.Debug("Connect button clicked")
	return nil
}

func (c *ConnectManager) handleConnectionDialog(message string) (*ConnectionResult, error) {
	result := &ConnectionResult{
		Success: false,
	}

	c.logger.Debug("Handling connection dialog")

	// Wait for dialog to appear
	if err := c.waitForConnectionDialog(); err != nil {
		result.ErrorMessage = fmt.Sprintf("Connection dialog did not appear: %v", err)
		return result, err
	}

	// Check if message input is present
	messageInput, err := c.page.Element("textarea[name='message']")
	if err != nil {
		messageInput, err = c.page.Element(".send-invite__message-input")
	}
	if err != nil {
		messageInput, err = c.page.Element("textarea[placeholder*='add a note']")
	}

	if err == nil && messageInput != nil && message != "" {
		c.logger.Debug("Found message input, typing message")

		// Click message input
		if err := messageInput.Click("left", 1); err != nil {
			result.ErrorMessage = fmt.Sprintf("Failed to click message input: %v", err)
			return result, err
		}

		// Type message with human-like typing
		if err := c.stealth.HumanLikeType(c.page, message); err != nil {
			result.ErrorMessage = fmt.Sprintf("Failed to type message: %v", err)
			return result, err
		}

		c.logger.WithField("message_length", len(message)).Debug("Message typed")
	}

	// Find and click send button
	sendButton, err := c.page.Element("button[aria-label*='Send invitation']")
	if err != nil {
		sendButton, err = c.page.Element(".send-invite__button")
	}
	if err != nil {
		sendButton, err = c.page.Element("button[type='submit']")
	}

	if err != nil {
		result.ErrorMessage = "Send button not found"
		return result, err
	}

	c.logger.Debug("Clicking send button")

	// Click send button
	if err := sendButton.Click("left", 1); err != nil {
		result.ErrorMessage = fmt.Sprintf("Failed to click send button: %v", err)
		return result, err
	}

	// Wait for dialog to close
	time.Sleep(1 * time.Second)

	// Check if request was sent successfully
	if c.isRequestSentSuccessfully() {
		result.Success = true
		result.RequestSent = true
		c.logger.Info("Connection request sent successfully")
	} else {
		result.ErrorMessage = "Failed to verify request was sent"
		c.logger.Warn("Could not verify connection request was sent")
	}

	return result, nil
}

func (c *ConnectManager) waitForConnectionDialog() error {
	selectors := []string{
		".send-invite-modal",
		".modal__content",
		"[data-test-id='connection-dialog']",
		".artdeco-modal",
	}

	for i := 0; i < 10; i++ {
		for _, selector := range selectors {
			element, err := c.page.Element(selector)
			if err == nil && element != nil {
				c.logger.WithField("selector", selector).Debug("Found connection dialog")
				return nil
			}
		}
		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("connection dialog not found after waiting")
}

func (c *ConnectManager) isRequestSentSuccessfully() bool {
	// Look for success indicators
	selectors := []string{
		".pv-s-profile-actions--connect.pending",
		"[data-test-id='profile-connect-button'][aria-label*='Pending']",
		".pv-s-profile-actions--withdraw",
		".success-indicator",
	}

	for _, selector := range selectors {
		element, err := c.page.Element(selector)
		if err == nil && element != nil {
			return true
		}
	}

	return false
}

func (c *ConnectManager) processTemplate(template string, variables map[string]string) string {
	result := template
	
	for key, value := range variables {
		placeholder := fmt.Sprintf("{{%s}}", key)
		result = strings.ReplaceAll(result, placeholder, value)
	}
	
	return result
}

// GetDefaultTemplates returns default connection message templates
func GetDefaultTemplates() []MessageTemplate {
	return []MessageTemplate{
		{
			ID:      "professional",
			Name:    "Professional",
			Content: "Hi {{name}}, I came across your profile and was impressed by your experience in {{industry}}. I'd love to connect and learn more about your work.",
			Variables: []string{"name", "industry"},
		},
		{
			ID:      "networking",
			Name:    "Networking",
			Content: "Hello {{name}}, I'm looking to expand my professional network in {{field}}. Your background seems very relevant, and I'd be honored to connect.",
			Variables: []string{"name", "field"},
		},
		{
			ID:      "simple",
			Name:    "Simple",
			Content: "Hi {{name}}, I'd like to connect with you on LinkedIn.",
			Variables: []string{"name"},
		},
	}
}
