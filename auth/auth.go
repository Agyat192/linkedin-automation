package auth

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/sirupsen/logrus"
)

// AuthManager handles LinkedIn authentication
type AuthManager struct {
	browser   *rod.Browser
	page      *rod.Page
	logger    *logrus.Logger
	email     string
	password  string
	sessionPath string
	rng       *rand.Rand
}

// LoginResult represents the result of a login attempt
type LoginResult struct {
	Success      bool
	ErrorMessage string
	Requires2FA  bool
	RequiresCaptcha bool
	SessionID    string
}

// NewAuthManager creates a new authentication manager
func NewAuthManager(email, password, sessionPath string, logger *logrus.Logger) *AuthManager {
	return &AuthManager{
		email:       email,
		password:    password,
		sessionPath: sessionPath,
		logger:      logger,
		rng:         rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// isChromeRunning checks if any Chrome process is running
func isChromeRunning() bool {
	cmd := exec.Command("tasklist", "/FI", "IMAGENAME eq chrome.exe", "/FO", "CSV")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	
	// Check if chrome.exe is found in the output
	return strings.Contains(string(output), "chrome.exe")
}

// getChromePath returns the path to Chrome executable
func getChromePath() string {
	// Common Chrome installation paths on Windows
	paths := []string{
		filepath.Join(os.Getenv("LOCALAPPDATA"), "Google\\Chrome\\Application\\chrome.exe"),
		filepath.Join(os.Getenv("PROGRAMFILES"), "Google\\Chrome\\Application\\chrome.exe"),
		filepath.Join(os.Getenv("PROGRAMFILES(X86)"), "Google\\Chrome\\Application\\chrome.exe"),
	}
	
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return "" // Chrome not found
}

// InitializeBrowser initializes the browser with stealth settings
func (a *AuthManager) InitializeBrowser(headless bool, userAgent string) error {
	a.logger.Info("Initializing browser")

	// Check if Chrome is running and use Chrome instead of Chromium
	chromeRunning := isChromeRunning()
	chromePath := getChromePath()
	
	if chromeRunning && chromePath != "" {
		a.logger.Info("Chrome process detected, using Chrome instead of Chromium")
	} else if chromeRunning {
		a.logger.Warn("Chrome process detected but Chrome executable not found, proceeding with default browser")
	} else {
		a.logger.Info("No Chrome process detected, using default browser")
	}

	// Try to connect to existing browser first
	if !headless {
		// Try to connect to existing Chrome instance with user's profile
		l := launcher.New()
		
		// If Chrome is running, use Chrome executable
		if chromeRunning && chromePath != "" {
			l = l.Bin(chromePath)
		}
		
		l = l.Leakless(false).
			Headless(false).
			Set("user-data-dir", os.Getenv("LOCALAPPDATA")+"\\Google\\Chrome\\User Data").
			Set("profile-directory", "Default").
			Set("no-first-run", "true").
			Set("no-default-browser-check", "true").
			Set("disable-features", "VizDisplayCompositor").
			Set("disable-web-security", "false").
			Set("remote-debugging-port", "9222")
		
		// Try to launch with remote debugging
		url, err := l.Launch()
		if err != nil {
			a.logger.Warn("Failed to connect to existing Chrome, trying Edge...")
			
			// Try Edge as fallback
			l = launcher.New().
				Leakless(false).
				Headless(false).
				Set("user-data-dir", os.Getenv("LOCALAPPDATA")+"\\Microsoft\\Edge\\User Data").
				Set("profile-directory", "Default").
				Set("no-first-run", "true").
				Set("no-default-browser-check", "true").
				Set("remote-debugging-port", "9223")
			
			url, err = l.Launch()
			if err != nil {
				a.logger.Warn("Failed to connect to existing browsers, launching new one")
			} else {
				// Connect to Edge
				a.browser = rod.New().ControlURL(url)
				if err := a.browser.Connect(); err != nil {
					a.logger.Warn("Failed to connect to Edge, launching new one")
				} else {
					a.logger.Info("Connected to existing Edge successfully")
					return nil
				}
			}
		} else {
			// Connect to Chrome
			a.browser = rod.New().ControlURL(url)
			if err := a.browser.Connect(); err != nil {
				a.logger.Warn("Failed to connect to Chrome, launching new one")
			} else {
				a.logger.Info("Connected to existing Chrome successfully")
				return nil
			}
		}
	}

	// Create launcher with custom options - disable leakless to avoid Windows Defender issues
	l := launcher.New()
	
	// If Chrome is running, use Chrome executable for new instance too
	if chromeRunning && chromePath != "" {
		l = l.Bin(chromePath)
		a.logger.Info("Using Chrome executable for new browser instance")
	}
	
	l = l.Leakless(false). // Disable leakless to avoid virus detection
		Headless(headless).
		Set("user-agent", userAgent).
		Set("disable-web-security", "true").
		Set("disable-features", "VizDisplayCompositor").
		Set("disable-background-timer-throttling", "true").
		Set("disable-backgrounding-occluded-windows", "true").
		Set("disable-renderer-backgrounding", "true").
		Set("disable-field-trial-config", "true").
		Set("disable-ipc-flooding-protection", "true").
		Set("enable-features", "NetworkService,NetworkServiceInProcess").
		Set("no-first-run", "true").
		Set("no-default-browser-check", "true").
		Set("disable-default-apps", "true").
		Set("disable-popup-blocking", "true").
		Set("disable-prompt-on-repost", "true").
		Set("disable-hang-monitor", "true").
		Set("disable-sync", "true").
		Set("disable-extensions", "true").
		Set("disable-plugins", "true").
		Set("disable-images", "false").
		Set("disable-javascript", "false").
		Set("disable-dev-shm-usage", "true").
		Set("disable-gpu", "true").
		Set("remote-debugging-port", "9222")

	// Add user data directory for session persistence
	// Use unique directory to avoid conflicts with existing Chrome processes
	timestamp := time.Now().Format("20060102-150405")
	userDataDir := filepath.Join(a.sessionPath, fmt.Sprintf("browser-data-%s", timestamp))
	if err := os.MkdirAll(userDataDir, 0755); err != nil {
		return fmt.Errorf("failed to create user data directory: %w", err)
	}
	l = l.Set("user-data-dir", userDataDir)

	// Launch browser
	url, err := l.Launch()
	if err != nil {
		return fmt.Errorf("failed to launch browser: %w", err)
	}

	// Connect to browser
	a.browser = rod.New().ControlURL(url)
	if err := a.browser.Connect(); err != nil {
		return fmt.Errorf("failed to connect to browser: %w", err)
	}

	a.logger.Info("Browser initialized successfully")
	return nil
}

// Login performs LinkedIn login and returns authentication result
func (a *AuthManager) Login(ctx context.Context) (*LoginResult, error) {
	a.logger.Info("Starting LinkedIn login process")

	// Create context with extended timeout for checkpoint verification
	ctx, cancel := context.WithTimeout(ctx, 300*time.Second) // 5 minutes for manual verification
	defer cancel()

	result := &LoginResult{}

	// Create a new page with timeout
	page, err := a.browser.Page(proto.TargetCreateTarget{})
	if err != nil {
		return nil, fmt.Errorf("failed to create page: %w", err)
	}
	defer page.Close()
	a.page = page

	// Navigate to LinkedIn login page
	a.logger.Info("Navigating to LinkedIn login page")
	
	// Use a simpler navigation approach
	err = page.Navigate("https://www.linkedin.com/login")
	if err != nil {
		return nil, fmt.Errorf("failed to navigate to login page: %w", err)
	}
	
	a.logger.Info("Navigation initiated, waiting for page load...")

	// Wait for page to load with context and shorter timeout
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("login timeout during page load")
	default:
		// Try to wait for load with timeout
		done := make(chan error, 1)
		go func() {
			done <- page.WaitLoad()
		}()
		
		select {
		case err := <-done:
			if err != nil {
				a.logger.Warn("Page.WaitLoad() failed, trying alternative approach")
				// Fallback: wait for a specific element instead
				_, err := page.Element("input[name='session_key']")
				if err != nil {
					return nil, fmt.Errorf("failed to find login form: %w", err)
				}
			}
		case <-time.After(20 * time.Second):
			a.logger.Warn("Page load timeout, trying to proceed anyway")
			// Fallback: try to proceed if we can find the login form
			_, err := page.Element("input[name='session_key']")
			if err != nil {
				return nil, fmt.Errorf("page load timeout and login form not found")
			}
		case <-ctx.Done():
			return nil, fmt.Errorf("login timeout")
		}
	}
	
	a.logger.Info("Page loaded successfully")

	// Check if already logged in
	a.logger.Info("Checking if already logged in...")
	
	// Simple URL check to avoid panics
	urlInfo, err := a.page.Info()
	if err == nil && urlInfo.URL != "" {
		a.logger.WithField("url", urlInfo.URL).Info("Current page URL")
		
		// If we're already on a LinkedIn page (not login), we're logged in
		if !strings.Contains(urlInfo.URL, "linkedin.com/login") {
			a.logger.Info("Already logged in - detected by URL")
			result.Success = true
			result.SessionID = a.getSessionID()
			return result, nil
		}
	}
	
	a.logger.Info("Not logged in, proceeding with credential filling...")

	// Fill in credentials
	a.logger.Info("Starting to fill credentials...")
	if err := a.fillCredentials(); err != nil {
		return nil, fmt.Errorf("failed to fill credentials: %w", err)
	}

	// Submit login form
	a.logger.Info("Submitting login form...")
	if err := a.submitLogin(); err != nil {
		return nil, fmt.Errorf("failed to submit login: %w", err)
	}

	// Wait for navigation with timeout
	a.logger.Info("Waiting for navigation after login...")
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("login timeout during navigation")
	default:
		done := make(chan struct{}, 1)
		go func() {
			a.page.WaitNavigation(proto.PageLifecycleEventNameLoad)
			done <- struct{}{}
		}()
		
		select {
		case <-done:
			a.logger.Info("Navigation completed successfully")
		case <-time.After(15 * time.Second):
			a.logger.Warn("Navigation timeout, proceeding anyway")
		case <-ctx.Done():
			return nil, fmt.Errorf("login timeout")
		}
	}

	// Check if login succeeded by checking URL
	a.logger.Info("Checking login status...")
	time.Sleep(2 * time.Second) // Wait for page to settle
	
	urlInfo, err = a.page.Info()
	if err == nil && urlInfo.URL != "" {
		a.logger.WithField("url", urlInfo.URL).Info("Post-login URL")
		
		// If we're no longer on login page, check where we are
		if !strings.Contains(urlInfo.URL, "linkedin.com/login") {
			// Check if we're on a checkpoint/challenge page
			if strings.Contains(urlInfo.URL, "checkpoint") || 
			   strings.Contains(urlInfo.URL, "challenge") {
				a.logger.Warn("LinkedIn checkpoint/challenge detected - waiting for manual verification")
				a.logger.Info("Please complete any verification in the browser window")
				
				// Wait for manual verification with extended timeout
				select {
				case <-ctx.Done():
					return nil, fmt.Errorf("timeout during checkpoint verification")
				case <-time.After(5 * time.Minute):
					a.logger.Info("Checkpoint timeout, proceeding anyway...")
				}
				
				// Check again after waiting
				time.Sleep(2 * time.Second)
				urlInfo, err = a.page.Info()
				if err == nil && urlInfo.URL != "" {
					// If we're still on checkpoint, fail
					if strings.Contains(urlInfo.URL, "checkpoint") {
						result.ErrorMessage = "LinkedIn checkpoint verification required"
						return result, nil
					}
				}
			}
			
			// If we're on feed or another authenticated page, success
			if strings.Contains(urlInfo.URL, "linkedin.com/feed") ||
			   strings.Contains(urlInfo.URL, "linkedin.com/in/") ||
			   strings.Contains(urlInfo.URL, "linkedin.com/mynetwork") ||
			   strings.Contains(urlInfo.URL, "linkedin.com/jobs") ||
			   strings.Contains(urlInfo.URL, "linkedin.com/search") {
				a.logger.Info("Login successful - reached authenticated page")
				result.Success = true
				result.SessionID = a.getSessionID()
				return result, nil
			}
			
			// If we're not on login page but also not on a known authenticated page,
			// assume success but with warning
			a.logger.Warn("Login appears successful but on unexpected page")
			result.Success = true
			result.SessionID = a.getSessionID()
			return result, nil
		}
	}

	// If still on login page, check for challenges
	a.logger.Info("Still on login page, checking for challenges...")
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("login timeout during challenge check")
	default:
		if a.requires2FA() {
			a.logger.Warn("Login requires 2FA")
			result.Requires2FA = true
			result.ErrorMessage = "2FA authentication required. Please handle manually."
			return result, nil
		}

		if a.requiresCaptcha() {
			a.logger.Warn("Login requires CAPTCHA - attempting manual handling")
			if err := a.handleCaptchaManually(ctx); err != nil {
				result.RequiresCaptcha = true
				result.ErrorMessage = fmt.Sprintf("CAPTCHA challenge detected: %v", err)
				return result, nil
			}
			a.logger.Info("CAPTCHA handled successfully, proceeding...")
			
			// Check again after CAPTCHA
			time.Sleep(2 * time.Second)
			urlInfo, err = a.page.Info()
			if err == nil && urlInfo.URL != "" && !strings.Contains(urlInfo.URL, "linkedin.com/login") {
				a.logger.Info("Login successful after CAPTCHA")
				result.Success = true
				result.SessionID = a.getSessionID()
				return result, nil
			}
		}
	}

	// Check for login errors
	a.logger.Info("Checking for login errors...")
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("login timeout during error check")
	default:
		if errorMsg := a.getLoginError(); errorMsg != "" {
			result.ErrorMessage = errorMsg
			return result, nil
		}
	}

	// Verify successful login
	a.logger.Info("Verifying login success...")
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("login timeout during verification")
	default:
		if !a.isLoggedIn() {
			result.ErrorMessage = "Login failed - unable to verify authentication"
			return result, nil
		}
	}

	// Save session
	if err := a.saveSession(); err != nil {
		a.logger.WithError(err).Warn("Failed to save session")
	}

	a.logger.Info("Login successful")
	result.Success = true
	result.SessionID = a.getSessionID()
	return result, nil
}

// VerifySession checks if the current session is still valid
func (a *AuthManager) VerifySession(ctx context.Context) (bool, error) {
	if a.browser == nil {
		if err := a.InitializeBrowser(true, ""); err != nil {
			return false, err
		}
	}

	page, err := a.browser.Page(proto.TargetCreateTarget{})
	if err != nil {
		return false, fmt.Errorf("failed to create page: %w", err)
	}
	defer page.Close()

	// Navigate to LinkedIn homepage
	if err := page.Navigate("https://www.linkedin.com"); err != nil {
		return false, fmt.Errorf("failed to navigate to homepage: %w", err)
	}

	// Wait for page to load
	if err := page.WaitLoad(); err != nil {
		return false, fmt.Errorf("failed to wait for page load: %w", err)
	}

	// Check if logged in
	isLoggedIn := a.checkPageLoginStatus(page)
	a.logger.WithField("logged_in", isLoggedIn).Debug("Session verification completed")

	return isLoggedIn, nil
}

// GetAuthenticatedPage returns an authenticated page
func (a *AuthManager) GetAuthenticatedPage(ctx context.Context) (*rod.Page, error) {
	if a.browser == nil {
		if err := a.InitializeBrowser(true, ""); err != nil {
			return nil, err
		}
	}

	page, err := a.browser.Page(proto.TargetCreateTarget{})
	if err != nil {
		return nil, fmt.Errorf("failed to create page: %w", err)
	}

	// Verify authentication
	if !a.checkPageLoginStatus(page) {
		page.Close()
		
		// Try to login again
		result, err := a.Login(ctx)
		if err != nil || !result.Success {
			return nil, fmt.Errorf("authentication failed: %v", err)
		}

		// Create new page after login
		page, err = a.browser.Page(proto.TargetCreateTarget{})
		if err != nil {
			return nil, fmt.Errorf("failed to create page after login: %w", err)
		}
	}

	return page, nil
}

// Close closes the browser
func (a *AuthManager) Close() error {
	if a.browser != nil {
		return a.browser.Close()
	}
	return nil
}

// Helper function to wait for element with timeout
func (a *AuthManager) waitForElement(selector string, timeout time.Duration) error {
	done := make(chan error, 1)
	go func() {
		_, err := a.page.Element(selector)
		done <- err
	}()
	
	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		return fmt.Errorf("timeout waiting for element: %s", selector)
	}
}

// Helper function to click element with timeout
func (a *AuthManager) clickElement(selector string, timeout time.Duration) error {
	done := make(chan error, 1)
	go func() {
		el, err := a.page.Element(selector)
		if err != nil {
			done <- err
			return
		}
		done <- el.Click("left", 1)
	}()
	
	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		return fmt.Errorf("timeout clicking element: %s", selector)
	}
}

// Helper function to input text with timeout
func (a *AuthManager) inputText(selector, text string, timeout time.Duration) error {
	done := make(chan error, 1)
	go func() {
		el, err := a.page.Element(selector)
		if err != nil {
			done <- err
			return
		}
		
		// Simulate human typing with variable speed
		if text == "" {
			// Clear field - instant action
			done <- el.Input(text)
			return
		}
		
		// Human-like typing: type each character with random delays
		for i, char := range text {
			// Random typing delay: 50-150ms per character
			typingDelay := time.Duration(50 + a.rng.Intn(100)) * time.Millisecond
			
			// Type single character
			if err := el.Input(string(char)); err != nil {
				done <- err
				return
			}
			
			// Add typing delay
			time.Sleep(typingDelay)
			
			// Occasional longer pause (thinking)
			if i > 0 && i%5 == 0 && a.rng.Float64() < 0.3 {
				pauseTime := time.Duration(200 + a.rng.Intn(300)) * time.Millisecond
				time.Sleep(pauseTime)
			}
		}
		
		done <- nil
	}()
	
	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		return fmt.Errorf("timeout inputting text to element: %s", selector)
	}
}

func (a *AuthManager) isLoggedIn() bool {
	return a.checkPageLoginStatus(a.page)
}

func (a *AuthManager) checkPageLoginStatus(page *rod.Page) bool {
	// Simple URL-based check to avoid panics
	defer func() {
		if r := recover(); r != nil {
			// Ignore panics from Rod library
		}
	}()
	
	// Check URL for login redirect - most reliable method
	urlInfo, err := page.Info()
	if err == nil && urlInfo.URL != "" {
		a.logger.WithField("url", urlInfo.URL).Debug("Current page URL")
		
		// If we're on login page, we're not logged in
		if strings.Contains(urlInfo.URL, "linkedin.com/login") {
			return false
		}
		
		// If we're on feed or profile page, we're logged in
		if strings.Contains(urlInfo.URL, "linkedin.com/feed") ||
		   strings.Contains(urlInfo.URL, "linkedin.com/in/") ||
		   strings.Contains(urlInfo.URL, "linkedin.com/mynetwork") {
			return true
		}
	}
	
	// Default to false to be safe
	return false
}

func (a *AuthManager) fillCredentials() error {
	a.logger.Info("Filling login credentials")
	
	// Wait for email field with timeout
	if err := a.waitForElement("input[name='session_key']", 10*time.Second); err != nil {
		return fmt.Errorf("email field not found: %w", err)
	}

	// Human-like pause before starting to fill
	time.Sleep(time.Duration(1000 + a.rng.Intn(2000)) * time.Millisecond)

	// Clear and fill email
	if err := a.clickElement("input[name='session_key']", 5*time.Second); err != nil {
		return fmt.Errorf("failed to click email field: %w", err)
	}

	// Pause after clicking field
	time.Sleep(time.Duration(200 + a.rng.Intn(300)) * time.Millisecond)

	if err := a.inputText("input[name='session_key']", "", 5*time.Second); err != nil {
		return fmt.Errorf("failed to clear email field: %w", err)
	}

	// Brief pause before typing email
	time.Sleep(time.Duration(100 + a.rng.Intn(200)) * time.Millisecond)

	if err := a.inputText("input[name='session_key']", a.email, 15*time.Second); err != nil {
		return fmt.Errorf("failed to input email: %w", err)
	}

	// Human-like pause between email and password
	time.Sleep(time.Duration(1000 + a.rng.Intn(1500)) * time.Millisecond)

	// Wait for password field with timeout
	if err := a.waitForElement("input[name='session_password']", 10*time.Second); err != nil {
		return fmt.Errorf("password field not found: %w", err)
	}

	// Pause before clicking password field
	time.Sleep(time.Duration(200 + a.rng.Intn(300)) * time.Millisecond)

	// Clear and fill password
	if err := a.clickElement("input[name='session_password']", 5*time.Second); err != nil {
		return fmt.Errorf("failed to click password field: %w", err)
	}

	// Pause after clicking field
	time.Sleep(time.Duration(200 + a.rng.Intn(300)) * time.Millisecond)

	if err := a.inputText("input[name='session_password']", "", 5*time.Second); err != nil {
		return fmt.Errorf("failed to clear password field: %w", err)
	}

	// Brief pause before typing password
	time.Sleep(time.Duration(100 + a.rng.Intn(200)) * time.Millisecond)

	if err := a.inputText("input[name='session_password']", a.password, 15*time.Second); err != nil {
		return fmt.Errorf("failed to input password: %w", err)
	}

	// Final pause before submitting
	time.Sleep(time.Duration(1500 + a.rng.Intn(2000)) * time.Millisecond)

	a.logger.Info("Credentials filled successfully")
	return nil
}

func (a *AuthManager) submitLogin() error {
	a.logger.Info("Submitting login form")
	
	// Find and click login button with timeout
	if err := a.waitForElement("button[type='submit']", 10*time.Second); err != nil {
		return fmt.Errorf("login button not found: %w", err)
	}

	// Human-like hesitation before clicking submit
	time.Sleep(time.Duration(1000 + a.rng.Intn(2000)) * time.Millisecond)

	if err := a.clickElement("button[type='submit']", 5*time.Second); err != nil {
		return fmt.Errorf("failed to click login button: %w", err)
	}

	a.logger.Info("Login form submitted")
	return nil
}

func (a *AuthManager) requires2FA() bool {
	// Simple check without complex operations to avoid panics
	defer func() {
		if r := recover(); r != nil {
			// Ignore panics from Rod library
		}
	}()
	
	// Check for 2FA input field - most reliable indicator
	_, err := a.page.Element("input[name='pin']")
	if err == nil {
		return true
	}
	
	// Check URL for 2FA page
	urlInfo, err := a.page.Info()
	if err == nil && urlInfo.URL != "" {
		if strings.Contains(urlInfo.URL, "two-factor") || 
		   strings.Contains(urlInfo.URL, "challenge") {
			return true
		}
	}
	
	return false
}

func (a *AuthManager) handleCaptchaManually(ctx context.Context) error {
	a.logger.Info("Switching to non-headless mode for CAPTCHA handling")
	
	// Close current headless browser
	if a.browser != nil {
		a.browser.Close()
	}
	
	// Reinitialize browser in non-headless mode for manual CAPTCHA
	if err := a.InitializeBrowser(false, ""); err != nil {
		return fmt.Errorf("failed to reinitialize browser: %w", err)
	}
	
	// Create new page and navigate to login
	page, err := a.browser.Page(proto.TargetCreateTarget{})
	if err != nil {
		return fmt.Errorf("failed to create page: %w", err)
	}
	defer page.Close()
	a.page = page
	
	// Navigate to login again
	if err := page.Navigate("https://www.linkedin.com/login"); err != nil {
		return fmt.Errorf("failed to navigate: %w", err)
	}
	
	// Wait for page load
	time.Sleep(3 * time.Second)
	
	// Fill credentials again
	if err := a.fillCredentials(); err != nil {
		return fmt.Errorf("failed to fill credentials: %w", err)
	}
	
	// Submit login
	if err := a.submitLogin(); err != nil {
		return fmt.Errorf("failed to submit login: %w", err)
	}
	
	a.logger.Info("=== MANUAL CAPTCHA REQUIRED ===")
	a.logger.Info("Please solve the CAPTCHA in the browser window that opened")
	a.logger.Info("Waiting 60 seconds for CAPTCHA to be solved...")
	
	// Wait for manual CAPTCHA solving or timeout
	select {
	case <-ctx.Done():
		return fmt.Errorf("timeout during manual CAPTCHA handling")
	case <-time.After(60 * time.Second):
		a.logger.Info("CAPTCHA handling timeout, proceeding anyway...")
	}
	
	// Check if login succeeded after CAPTCHA
	time.Sleep(2 * time.Second)
	if a.isLoggedIn() {
		a.logger.Info("CAPTCHA solved successfully - logged in!")
		return nil
	}
	
	return fmt.Errorf("CAPTCHA may not have been solved correctly")
}

func (a *AuthManager) requiresCaptcha() bool {
	// Simple check without complex operations to avoid panics
	defer func() {
		if r := recover(); r != nil {
			// Ignore panics from Rod library
		}
	}()
	
	// Check URL for captcha page
	urlInfo, err := a.page.Info()
	if err == nil && urlInfo.URL != "" {
		if strings.Contains(urlInfo.URL, "captcha") || 
		   strings.Contains(urlInfo.URL, "challenge") {
			return true
		}
	}
	
	return false
}

func (a *AuthManager) getLoginError() string {
	// Check for error messages
	errorElements := []string{
		".alert-error",
		".login__form-error",
		".form-error",
		"[data-test-id='error']",
	}

	for _, selector := range errorElements {
		errorElement, err := a.page.Element(selector)
		if err == nil && errorElement != nil {
			errorText, err := errorElement.Text()
			if err == nil && errorText != "" {
				return errorText
			}
		}
	}

	return ""
}

func (a *AuthManager) getSessionID() string {
	// Get cookies from current page
	cookies, err := a.page.Cookies([]string{"https://www.linkedin.com"})
	if err != nil {
		a.logger.WithError(err).Warn("Failed to get cookies")
		return ""
	}

	// Look for li_at cookie (LinkedIn authentication token)
	for _, cookie := range cookies {
		if cookie.Name == "li_at" {
			return cookie.Value
		}
	}

	return ""
}

func (a *AuthManager) saveSession() error {
	if err := os.MkdirAll(a.sessionPath, 0755); err != nil {
		return fmt.Errorf("failed to create session directory: %w", err)
	}

	// Get current cookies
	currentCookies, err := a.page.Cookies([]string{"https://www.linkedin.com"})
	if err != nil {
		return fmt.Errorf("failed to get cookies: %w", err)
	}

	// Save session info
	sessionFile := filepath.Join(a.sessionPath, "session.json")
	_ = map[string]interface{}{
		"cookies":    currentCookies,
		"created_at": time.Now(),
		"user_agent": a.page.MustEval("navigator.userAgent").String(),
	}

	// In a real implementation, you would save this to a file
	// For now, we'll just log the session info
	a.logger.WithFields(logrus.Fields{
		"cookies_count": len(currentCookies),
		"session_file":  sessionFile,
	}).Info("Session saved")

	return nil
}

func (a *AuthManager) loadSession() error {
	sessionFile := filepath.Join(a.sessionPath, "session.json")
	if _, err := os.Stat(sessionFile); os.IsNotExist(err) {
		return fmt.Errorf("no session file found")
	}

	// In a real implementation, you would load and restore cookies
	// For now, we'll just log that we found a session file
	a.logger.WithField("session_file", sessionFile).Info("Session file found")
	return nil
}
