package message

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/sirupsen/logrus"
)

// MessageManager handles LinkedIn messaging
type MessageManager struct {
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

// Message represents a LinkedIn message
type Message struct {
	RecipientURL string
	Content      string
	Type         string // connection_note, follow_up, custom
	Status       string // sent, failed, pending
	SentAt       time.Time
}

// MessageResult represents the result of sending a message
type MessageResult struct {
	Success     bool
	RecipientURL string
	ErrorMessage string
	MessageID   string
	SentAt      time.Time
}

// MessageTemplate represents a message template
type MessageTemplate struct {
	ID          string
	Name        string
	Content     string
	Type        string
	Variables   []string
	CharacterLimit int
}

// Conversation represents a LinkedIn conversation
type Conversation struct {
	ParticipantURL string
	ParticipantName string
	LastMessage    string
	LastMessageTime time.Time
	MessageCount   int
}

// NewMessageManager creates a new message manager
func NewMessageManager(page *rod.Page, logger *logrus.Logger, stealth StealthManager) *MessageManager {
	return &MessageManager{
		page:    page,
		logger:  logger,
		stealth: stealth,
	}
}

// SendMessage sends a message to a LinkedIn user
func (m *MessageManager) SendMessage(ctx context.Context, recipientURL, content string) (*MessageResult, error) {
	m.logger.WithFields(logrus.Fields{
		"recipient_url": recipientURL,
		"content_length": len(content),
	}).Info("Sending message")

	result := &MessageResult{
		RecipientURL: recipientURL,
		SentAt:       time.Now(),
	}

	// Navigate to messaging
	if err := m.navigateToMessaging(); err != nil {
		result.ErrorMessage = fmt.Sprintf("Failed to navigate to messaging: %v", err)
		return result, err
	}

	// Find or start conversation with recipient
	if err := m.navigateToMessaging(); err != nil {
		result.ErrorMessage = fmt.Sprintf("Failed to find/start conversation: %v", err)
		return result, err
	}

	// Send the message
	if err := m.sendDirectMessage(content); err != nil {
		result.ErrorMessage = fmt.Sprintf("Failed to send message: %v", err)
		return result, err
	}

	result.Success = true
	m.logger.Info("Message sent successfully")

	return result, nil
}

// SendMessageWithTemplate sends a message using a template
func (m *MessageManager) SendMessageWithTemplate(ctx context.Context, recipientURL string, template MessageTemplate, variables map[string]string) (*MessageResult, error) {
	// Process template variables
	content := m.processTemplate(template.Content, variables)
	
	// Check character limit
	if template.CharacterLimit > 0 && len(content) > template.CharacterLimit {
		content = content[:template.CharacterLimit]
		m.logger.WithFields(logrus.Fields{
			"original_length": len(template.Content),
			"truncated_length": len(content),
			"limit": template.CharacterLimit,
		}).Warn("Message truncated due to character limit")
	}
	
	return m.SendMessage(ctx, recipientURL, content)
}

// SendFollowUpMessage sends a follow-up message to newly accepted connections
func (m *MessageManager) SendFollowUpMessage(ctx context.Context, recipientURL, templateContent string, variables map[string]string) (*MessageResult, error) {
	m.logger.WithField("recipient_url", recipientURL).Info("Sending follow-up message")

	// Process template
	content := m.processTemplate(templateContent, variables)
	
	result := &MessageResult{
		RecipientURL: recipientURL,
		SentAt:       time.Now(),
	}

	// Navigate to recipient's profile first
	if err := m.page.Navigate(recipientURL); err != nil {
		result.ErrorMessage = fmt.Sprintf("Failed to navigate to profile: %v", err)
		return result, err
	}

	// Wait for profile to load
	if err := m.page.WaitLoad(); err != nil {
		result.ErrorMessage = fmt.Sprintf("Failed to wait for profile load: %w", err)
		return result, err
	}

	// Look for message button on profile
	messageButton, err := m.page.Element("button[aria-label*='Message']")
	if err != nil {
		messageButton, err = m.page.Element(".pvs-profile-actions__action")
	}
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Failed to find message button: %v", err)
		return result, err
	}

	// Click message button
	if err := m.clickMessageButton(messageButton); err != nil {
		result.ErrorMessage = fmt.Sprintf("Failed to click message button: %v", err)
		return result, err
	}

	// Wait for messaging interface to load
	time.Sleep(m.stealth.RandomDelay())

	// Send the message
	if err := m.sendDirectMessage(content); err != nil {
		result.ErrorMessage = fmt.Sprintf("Failed to send follow-up message: %v", err)
		return result, err
	}

	result.Success = true
	m.logger.Info("Follow-up message sent successfully")

	return result, nil
}

// GetConversations retrieves all conversations
func (m *MessageManager) GetConversations(ctx context.Context) ([]*Conversation, error) {
	m.logger.Info("Retrieving conversations")

	if err := m.navigateToMessaging(); err != nil {
		return nil, fmt.Errorf("failed to navigate to messaging: %w", err)
	}

	// Wait for conversations list to load
	if err := m.waitForConversationsList(); err != nil {
		return nil, fmt.Errorf("failed to wait for conversations list: %w", err)
	}

	// Extract conversation data
	conversations, err := m.extractConversations()
	if err != nil {
		return nil, fmt.Errorf("failed to extract conversations: %w", err)
	}

	m.logger.WithField("count", len(conversations)).Info("Retrieved conversations")
	return conversations, nil
}

// GetNewlyAcceptedConnections finds connections that have recently accepted requests
func (m *MessageManager) GetNewlyAcceptedConnections(ctx context.Context, since time.Time) ([]string, error) {
	m.logger.WithField("since", since).Info("Finding newly accepted connections")

	// Navigate to network/connections page
	if err := m.page.Navigate("https://www.linkedin.com/mynetwork/invite-connect/connections/"); err != nil {
		return nil, fmt.Errorf("failed to navigate to connections page: %w", err)
	}

	// Wait for page to load
	if err := m.page.WaitLoad(); err != nil {
		return nil, fmt.Errorf("failed to wait for page load: %w", err)
	}

	// Extract recent connections
	connections, err := m.extractRecentConnections(since)
	if err != nil {
		return nil, fmt.Errorf("failed to extract recent connections: %w", err)
	}

	m.logger.WithField("count", len(connections)).Info("Found newly accepted connections")
	return connections, nil
}

// BatchSendMessages sends multiple messages
func (m *MessageManager) BatchSendMessages(ctx context.Context, recipients []string, content string) ([]*MessageResult, error) {
	m.logger.WithField("count", len(recipients)).Info("Starting batch message sending")

	results := make([]*MessageResult, 0, len(recipients))

	for i, recipientURL := range recipients {
		m.logger.WithFields(logrus.Fields{
			"current": i + 1,
			"total":   len(recipients),
			"recipient": recipientURL,
		}).Debug("Processing recipient")

		result, err := m.SendMessage(ctx, recipientURL, content)
		if err != nil {
			m.logger.WithError(err).Error("Failed to send message")
		}

		results = append(results, result)

		// Add delay between messages
		if i < len(recipients)-1 {
			time.Sleep(m.stealth.RandomDelay())
			
			// Add idle movement
			if err := m.stealth.AddIdleMovement(m.page); err != nil {
				m.logger.WithError(err).Warn("Failed to add idle movement")
			}
		}
	}

	// Count results
	successCount := 0
	for _, result := range results {
		if result.Success {
			successCount++
		}
	}

	m.logger.WithFields(logrus.Fields{
		"total": len(recipients),
		"success": successCount,
	}).Info("Batch message sending completed")

	return results, nil
}

// Private helper methods

func (m *MessageManager) navigateToMessaging() error {
	messagingURL := "https://www.linkedin.com/messaging/"
	
	if err := m.page.Navigate(messagingURL); err != nil {
		return fmt.Errorf("failed to navigate to messaging: %w", err)
	}

	// Get viewport size
	viewport, err := m.page.Eval("({width: window.innerWidth, height: window.innerHeight})")
	if err != nil {
		return fmt.Errorf("failed to get viewport: %w", err)
	}
	_ = viewport // Unused for now

	return nil
}

// ...

func (m *MessageManager) addRecipientToConversation(recipientURL string) error {
	m.logger.Debug("Adding recipient to conversation")

	// Look for recipient input field
	selectors := []string{
		"input[name='recipients']",
		".msg-form__recipients-input",
		"[data-test-id='recipients-input']",
		"input[placeholder*='Recipients']",
	}

	var recipientInput *rod.Element
	for _, selector := range selectors {
		input, err := m.page.Element(selector)
		if err == nil && input != nil {
			recipientInput = input
			break
		}
	}

	if recipientInput == nil {
		return fmt.Errorf("recipient input field not found")
	}

	// Click recipient input
	if err := recipientInput.Click("left", 1); err != nil {
		return fmt.Errorf("failed to click recipient input: %w", err)
	}

	// Type recipient name or email
	// For simplicity, we'll try to extract the name from the URL
	name := m.extractNameFromURL(recipientURL)
	if name == "" {
		name = "LinkedIn User"
	}

	if err := m.stealth.HumanLikeType(m.page, name); err != nil {
		return fmt.Errorf("failed to type recipient name: %w", err)
	}

	// Wait for suggestions to appear
	time.Sleep(1 * time.Second)

	// Look for and click on the first suggestion
	selectors = []string{
		".msg-suggestion-listitem",
		".recipient-suggestion",
		"[data-test-id='recipient-suggestion']",
	}

	for _, selector := range selectors {
		suggestions, err := m.page.Elements(selector)
		if err == nil && len(suggestions) > 0 {
			if err := suggestions[0].Click("left", 1); err == nil {
				m.logger.Debug("Selected recipient from suggestions")
				return nil
			}
		}
	}

	// If no suggestions found, try pressing Enter
	if err := m.page.Keyboard.Press(input.Enter); err != nil {
		return fmt.Errorf("failed to press Enter: %w", err)
	}

	return nil
}

// ...

func (m *MessageManager) sendDirectMessage(content string) error {
	m.logger.WithField("content_length", len(content)).Debug("Sending direct message")

	// Look for message input field
	selectors := []string{
		"textarea[aria-label*='Write a message']",
		"textarea[placeholder*='Write a message']",
		".msg-form__contenteditable",
		"[data-test-id='message-input']",
		".msg-textarea",
	}

	var messageInput *rod.Element
	for _, selector := range selectors {
		input, err := m.page.Element(selector)
		if err == nil && input != nil {
			messageInput = input
			break
		}
	}

	if messageInput == nil {
		return fmt.Errorf("message input field not found")
	}

	// Click message input
	if err := messageInput.Click("left", 1); err != nil {
		return fmt.Errorf("failed to click message input: %w", err)
	}

	// Type message with human-like typing
	if err := m.stealth.HumanLikeType(m.page, content); err != nil {
		return fmt.Errorf("failed to type message: %w", err)
	}

	// Look for send button
	selectors = []string{
		"button[aria-label*='Send']",
		".msg-form__send-button",
		"[data-test-id='send-button']",
		"button[type='submit']",
	}

	var sendButton *rod.Element
	for _, selector := range selectors {
		button, err := m.page.Element(selector)
		if err == nil && button != nil {
			sendButton = button
			break
		}
	}

	if sendButton == nil {
		return fmt.Errorf("send button not found")
	}

	// Click send button
	if err := sendButton.Click("left", 1); err != nil {
		return fmt.Errorf("failed to click send button: %w", err)
	}

	m.logger.Debug("Message sent")
	return nil
}

// ...

func (m *MessageManager) clickMessageButton(button *rod.Element) error {
	// Get button position for human-like mouse movement
	shape, err := button.Shape()
	if err != nil {
		return fmt.Errorf("failed to get button position: %w", err)
	}
	box := shape.Box()

	// Move mouse to button
	centerX := box.X + box.Width/2
	centerY := box.Y + box.Height/2

	// Get current mouse position
	viewport, err := m.page.Eval("({width: window.innerWidth, height: window.innerHeight})")
	if err != nil {
		return fmt.Errorf("failed to get viewport: %w", err)
	}
	fromX := viewport.Value.Get("width").Num()
	fromY := viewport.Value.Get("height").Num()

	// Human-like mouse movement
	if err := m.stealth.HumanLikeMouseMove(m.page, fromX, fromY, centerX, centerY); err != nil {
		m.logger.WithError(err).Warn("Failed to perform human-like mouse movement")
	}

	// Click button
	if err := button.Click("left", 1); err != nil {
		return fmt.Errorf("failed to click button: %w", err)
	}

	return nil
}

// ...
func (m *MessageManager) waitForConversationsList() error {
	selectors := []string{
		".msg-conversations-container",
		".conversation-list-container",
		"[data-test-id='conversations-list']",
	}

	for i := 0; i < 10; i++ {
		for _, selector := range selectors {
			element, err := m.page.Element(selector)
			if err == nil && element != nil {
				return nil
			}
		}
		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("conversations list not found after waiting")
}

func (m *MessageManager) extractConversations() ([]*Conversation, error) {
	conversations := make([]*Conversation, 0)

	// Look for conversation items
	selectors := []string{
		".msg-conversation-listitem",
		".conversation-list-item",
		"[data-test-id='conversation-item']",
	}

	var conversationElements []*rod.Element
	for _, selector := range selectors {
		elements, err := m.page.Elements(selector)
		if err == nil && len(elements) > 0 {
			conversationElements = elements
			break
		}
	}

	for _, element := range conversationElements {
		conversation, err := m.extractConversationData(element)
		if err != nil {
			m.logger.WithError(err).Warn("Failed to extract conversation data")
			continue
		}
		conversations = append(conversations, conversation)
	}

	return conversations, nil
}

func (m *MessageManager) extractConversationData(element *rod.Element) (*Conversation, error) {
	conversation := &Conversation{}

	// Extract participant name
	nameElement, err := element.Element(".msg-conversation-listitem__participant-names")
	if err != nil {
		nameElement, err = element.Element(".conversation-title")
	}
	if err == nil && nameElement != nil {
		name, err := nameElement.Text()
		if err == nil {
			conversation.ParticipantName = strings.TrimSpace(name)
		}
	}

	// Extract last message
	messageElement, err := element.Element(".msg-conversation-listitem__last-message")
	if err != nil {
		messageElement, err = element.Element(".conversation-snippet")
	}
	if err == nil && messageElement != nil {
		message, err := messageElement.Text()
		if err == nil {
			conversation.LastMessage = strings.TrimSpace(message)
		}
	}

	// Extract participant URL
	linkElement, err := element.Element("a")
	if err == nil && linkElement != nil {
		href, err := linkElement.Attribute("href")
		if err == nil && href != nil && *href != "" {
			if strings.HasPrefix(*href, "/") {
				conversation.ParticipantURL = "https://www.linkedin.com" + *href
			} else {
				conversation.ParticipantURL = *href
			}
		}
	}

	return conversation, nil
}

func (m *MessageManager) extractRecentConnections(since time.Time) ([]string, error) {
	connections := make([]string, 0)

	// Look for connection items
	selectors := []string{
		".mn-connections__connection-card",
		".connection-card",
		"[data-test-id='connection-item']",
	}

	var connectionElements []*rod.Element
	for _, selector := range selectors {
		elements, err := m.page.Elements(selector)
		if err == nil && len(elements) > 0 {
			connectionElements = elements
			break
		}
	}

	for _, element := range connectionElements {
		// Extract connection date and URL
		connectionURL, err := m.extractConnectionURL(element)
		if err != nil {
			continue
		}
		
		connections = append(connections, connectionURL)
	}

	return connections, nil
}

func (m *MessageManager) extractConnectionURL(element *rod.Element) (string, error) {
	linkElement, err := element.Element("a")
	if err != nil {
		return "", err
	}

	href, err := linkElement.Attribute("href")
	if err != nil || href == nil || *href == "" {
		return "", fmt.Errorf("no href found")
	}

	if strings.HasPrefix(*href, "/") {
		return "https://www.linkedin.com" + *href, nil
	}

	return *href, nil
}

func (m *MessageManager) extractProfileID(url string) string {
	// Extract profile ID from LinkedIn URL
	// Example: https://www.linkedin.com/in/john-doe-123456/
	parts := strings.Split(url, "/")
	for i, part := range parts {
		if part == "in" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

func (m *MessageManager) extractNameFromURL(url string) string {
	// Simple extraction of name from URL
	// In a real implementation, you might want to visit the profile to get the actual name
	parts := strings.Split(url, "/")
	for i, part := range parts {
		if part == "in" && i+1 < len(parts) {
			name := parts[i+1]
			// Remove URL encoding and dashes
			name = strings.ReplaceAll(name, "-", " ")
			return name
		}
	}
	return ""
}

func (m *MessageManager) processTemplate(template string, variables map[string]string) string {
	result := template
	
	for key, value := range variables {
		placeholder := fmt.Sprintf("{{%s}}", key)
		result = strings.ReplaceAll(result, placeholder, value)
	}
	
	return result
}

// GetDefaultMessageTemplates returns default message templates
func GetDefaultMessageTemplates() []MessageTemplate {
	return []MessageTemplate{
		{
			ID:            "follow_up_professional",
			Name:          "Professional Follow-up",
			Content:       "Hi {{name}}, thanks for connecting! I really enjoyed looking at your profile and would love to learn more about your work in {{field}}.",
			Type:          "follow_up",
			Variables:     []string{"name", "field"},
			CharacterLimit: 300,
		},
		{
			ID:            "follow_up_casual",
			Name:          "Casual Follow-up",
			Content:       "Hey {{name}}, great to connect! Looking forward to staying in touch and seeing where our professional paths might cross.",
			Type:          "follow_up",
			Variables:     []string{"name"},
			CharacterLimit: 200,
		},
		{
			ID:            "follow_up_value",
			Name:          "Value-based Follow-up",
			Content:       "Hi {{name}}, thank you for the connection! Based on your experience in {{industry}}, I thought you might find {{topic}} interesting. Would love to hear your thoughts!",
			Type:          "follow_up",
			Variables:     []string{"name", "industry", "topic"},
			CharacterLimit: 300,
		},
	}
}
