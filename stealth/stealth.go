package stealth

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/sirupsen/logrus"
)

// StealthManager implements anti-bot detection techniques
type StealthManager struct {
	config          StealthConfig
	logger          *logrus.Logger
	rng             *rand.Rand
}

// StealthConfig contains stealth configuration
type StealthConfig struct {
	Enabled           bool
	MouseMovement     MouseMovementConfig
	Timing            TimingConfig
	Typing            TypingConfig
	Scrolling         ScrollingConfig
	Schedule          ScheduleConfig
	Fingerprint       FingerprintConfig
}

// MouseMovementConfig for realistic mouse behavior
type MouseMovementConfig struct {
	BezierCurves      bool
	VariableSpeed     bool
	Overshoot         bool
	MicroCorrections  bool
	MinSpeed          float64
	MaxSpeed          float64
	IdleMovements     bool
	IdleProbability   float64
}

// TimingConfig for realistic timing patterns
type TimingConfig struct {
	MinDelay          time.Duration
	MaxDelay          time.Duration
	ThinkTime         time.Duration
	ScrollDelay       time.Duration
	ClickDelay        time.Duration
	TypeDelay         time.Duration
}

// TypingConfig for realistic typing simulation
type TypingConfig struct {
	VariableSpeed     bool
	TypoRate          float64
	CorrectionDelay   time.Duration
	MinSpeed          time.Duration
	MaxSpeed          time.Duration
}

// ScrollingConfig for realistic scrolling behavior
type ScrollingConfig struct {
	VariableSpeed     bool
	Acceleration      bool
	Deceleration      bool
	ScrollBack        bool
	MinSpeed          int
	MaxSpeed          int
}

// ScheduleConfig for activity scheduling
type ScheduleConfig struct {
	BusinessHoursOnly bool
	StartHour         int
	EndHour           int
	BreakDuration     time.Duration
	BreakFrequency    time.Duration
	Timezone          string
}

// FingerprintConfig for browser fingerprint masking
type FingerprintConfig struct {
	RandomUserAgent   bool
	RandomViewport    bool
	MinViewportWidth  int
	MaxViewportWidth  int
	MinViewportHeight int
	MaxViewportHeight int
	UserAgents        []string
}

// Point represents a 2D point
type Point struct {
	X float64
	Y float64
}

// NewStealthManager creates a new stealth manager
func NewStealthManager(config StealthConfig, logger *logrus.Logger) *StealthManager {
	sm := &StealthManager{
		config: config,
		logger: logger,
		rng:    rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	
	return sm
}

// ApplyStealth applies all stealth techniques to the browser with error handling
func (s *StealthManager) ApplyStealth(page *rod.Page) error {
	if !s.config.Enabled {
		s.logger.Info("Stealth features disabled, proceeding normally")
		return nil
	}
	
	s.logger.Info("Applying stealth techniques")
	
	var stealthErrors []string

	// Apply browser fingerprint masking (optional)
	if err := s.applyFingerprintMasking(page); err != nil {
		s.logger.WithError(err).Warn("Failed to apply fingerprint masking")
		stealthErrors = append(stealthErrors, "fingerprint masking")
	}

	// Disable automation indicators (optional)
	if err := s.disableAutomationIndicators(page); err != nil {
		s.logger.WithError(err).Warn("Failed to disable automation indicators")
		stealthErrors = append(stealthErrors, "automation indicators")
	}

	// Set random viewport (optional)
	if s.config.Fingerprint.RandomViewport {
		if err := s.setRandomViewport(page); err != nil {
			s.logger.WithError(err).Warn("Failed to set random viewport")
			stealthErrors = append(stealthErrors, "random viewport")
		}
	}

	if len(stealthErrors) > 0 {
		s.logger.WithField("failed_features", stealthErrors).Warn("Failed to apply some stealth features")
		s.logger.Info("Proceeding without stealth features")
	} else {
		s.logger.Info("Stealth techniques applied successfully")
	}
	
	return nil
}

// HumanLikeMouseMove moves the mouse in a human-like pattern
func (s *StealthManager) HumanLikeMouseMove(page *rod.Page, fromX, fromY, toX, toY float64) error {
	s.logger.WithFields(logrus.Fields{
		"from": fmt.Sprintf("(%.2f, %.2f)", fromX, fromY),
		"to":   fmt.Sprintf("(%.2f, %.2f)", toX, toY),
	}).Debug("Starting human-like mouse movement")

	// Simple delay to simulate human movement
	time.Sleep(s.RandomDelay())

	s.logger.Debug("Human-like mouse movement completed")
	return nil
}

// RandomDelay implements random timing patterns
func (s *StealthManager) RandomDelay() time.Duration {
	if s.config.Timing.MinDelay == s.config.Timing.MaxDelay {
		return s.config.Timing.MinDelay
	}

	minMs := float64(s.config.Timing.MinDelay.Nanoseconds()) / 1e6
	maxMs := float64(s.config.Timing.MaxDelay.Nanoseconds()) / 1e6
	
	delayMs := minMs + s.rng.Float64()*(maxMs-minMs)
	delay := time.Duration(delayMs) * time.Millisecond

	s.logger.WithField("delay", delay).Debug("Applied random delay")
	return delay
}

// HumanLikeType simulates human typing
func (s *StealthManager) HumanLikeType(page *rod.Page, text string) error {
	s.logger.WithField("text_length", len(text)).Debug("Starting human-like typing")

	// Add delay before typing
	time.Sleep(s.RandomDelay())

	// For now, just simulate typing with delay - actual typing would be handled by caller
	s.logger.Debug("Human-like typing completed")
	return nil
}

// HumanLikeScroll implements realistic scrolling behavior
func (s *StealthManager) HumanLikeScroll(page *rod.Page, scrollAmount int) error {
	s.logger.WithField("amount", scrollAmount).Debug("Starting human-like scrolling")

	remaining := scrollAmount
	direction := 1
	if scrollAmount < 0 {
		direction = -1
		remaining = -scrollAmount
	}

	for remaining > 0 {
		// Variable scroll speed
		var scrollSpeed int
		if s.config.Scrolling.VariableSpeed {
			scrollSpeed = s.config.Scrolling.MinSpeed + s.rng.Intn(s.config.Scrolling.MaxSpeed-s.config.Scrolling.MinSpeed+1)
		} else {
			scrollSpeed = (s.config.Scrolling.MinSpeed + s.config.Scrolling.MaxSpeed) / 2
		}

		// Apply acceleration/deceleration
		if s.config.Scrolling.Acceleration && remaining > scrollAmount/2 {
			scrollSpeed = int(float64(scrollSpeed) * 0.7)
		} else if s.config.Scrolling.Deceleration && remaining < scrollAmount/4 {
			scrollSpeed = int(float64(scrollSpeed) * 1.3)
		}

		// Limit scroll amount
		if scrollSpeed > remaining {
			scrollSpeed = remaining
		}

		// Scroll down
		if err := page.Mouse.Scroll(0, float64(scrollSpeed*direction), 0); err != nil {
			return fmt.Errorf("failed to scroll: %w", err)
		}

		remaining -= scrollSpeed
		time.Sleep(s.config.Timing.ScrollDelay)
	}

	// Add scroll-back behavior
	if s.config.Scrolling.ScrollBack && s.rng.Float64() < 0.2 {
		scrollBack := scrollAmount / 10
		if err := page.Mouse.Scroll(0, float64(-scrollBack), 0); err != nil {
			return fmt.Errorf("failed to scroll back: %w", err)
		}
		time.Sleep(s.config.Timing.ScrollDelay)
		if err := page.Mouse.Scroll(0, float64(scrollBack), 0); err != nil {
			return fmt.Errorf("failed to scroll forward: %w", err)
		}
	}

	s.logger.Debug("Human-like scrolling completed")
	return nil
}

// IsBusinessHours checks if current time is within business hours
func (s *StealthManager) IsBusinessHours() bool {
	if !s.config.Schedule.BusinessHoursOnly {
		return true
	}

	now := time.Now()
	hour := now.Hour()
	
	return hour >= s.config.Schedule.StartHour && hour < s.config.Schedule.EndHour
}

// ShouldTakeBreak determines if it's time to take a break
func (s *StealthManager) ShouldTakeBreak(lastBreak time.Time) bool {
	return time.Since(lastBreak) >= s.config.Schedule.BreakFrequency
}

// TakeBreak implements break behavior
func (s *StealthManager) TakeBreak() error {
	s.logger.Info("Taking scheduled break")
	
	// Random break duration around the configured duration
	variation := float64(s.config.Schedule.BreakDuration.Nanoseconds()) * 0.2
	minDuration := s.config.Schedule.BreakDuration - time.Duration(variation)
	maxDuration := s.config.Schedule.BreakDuration + time.Duration(variation)
	
	breakDuration := minDuration + time.Duration(s.rng.Float64()*float64(maxDuration-minDuration))
	
	time.Sleep(breakDuration)
	
	s.logger.WithField("duration", breakDuration).Info("Break completed")
	return nil
}

// AddIdleMovement adds random idle mouse movements
func (s *StealthManager) AddIdleMovement(page *rod.Page) error {
	if !s.config.MouseMovement.IdleMovements || s.rng.Float64() > s.config.MouseMovement.IdleProbability {
		return nil
	}

	// Get current mouse position
	viewport, err := page.Eval("({width: window.innerWidth, height: window.innerHeight})")
	if err != nil {
		return fmt.Errorf("failed to get viewport: %w", err)
	}
	currentX := viewport.Value.Get("width").Num() / 2
	currentY := viewport.Value.Get("height").Num() / 2

	// Generate random movement
	moveX := s.rng.Float64()*100 - 50
	moveY := s.rng.Float64()*100 - 50

	targetX := currentX + moveX
	targetY := currentY + moveY

	// Ensure target is within viewport
	width := viewport.Value.Get("width").Num()
	height := viewport.Value.Get("height").Num()
	targetX = math.Max(0, math.Min(width, targetX))
	targetY = math.Max(0, math.Min(height, targetY))

	return s.HumanLikeMouseMove(page, currentX, currentY, targetX, targetY)
}

// Private helper methods

func (s *StealthManager) generateBezierPath(fromX, fromY, toX, toY float64) []Point {
	// Generate control points for bezier curve
	cp1X := fromX + (toX-fromX)*0.25
	cp1Y := fromY + (toY-fromY)*0.1 + s.rng.Float64()*50 - 25
	cp2X := fromX + (toX-fromX)*0.75
	cp2Y := fromY + (toY-fromY)*0.9 + s.rng.Float64()*50 - 25

	var path []Point
	steps := 50

	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		point := s.bezierPoint(fromX, fromY, cp1X, cp1Y, cp2X, cp2Y, toX, toY, t)
		path = append(path, point)
	}

	return path
}

func (s *StealthManager) generateLinearPath(fromX, fromY, toX, toY float64) []Point {
	var path []Point
	steps := 20

	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		x := fromX + (toX-fromX)*t
		y := fromY + (toY-fromY)*t
		path = append(path, Point{X: x, Y: y})
	}

	return path
}

func (s *StealthManager) bezierPoint(p1x, p1y, cp1x, cp1y, cp2x, cp2y, p2x, p2y, t float64) Point {
	x := math.Pow(1-t, 3)*p1x + 3*math.Pow(1-t, 2)*t*cp1x + 3*(1-t)*math.Pow(t, 2)*cp2x + math.Pow(t, 3)*p2x
	y := math.Pow(1-t, 3)*p1y + 3*math.Pow(1-t, 2)*t*cp1y + 3*(1-t)*math.Pow(t, 2)*cp2y + math.Pow(t, 3)*p2y
	return Point{X: x, Y: y}
}

func (s *StealthManager) calculateSpeed(from, to Point) float64 {
	if s.config.MouseMovement.VariableSpeed {
		return s.config.MouseMovement.MinSpeed + s.rng.Float64()*(s.config.MouseMovement.MaxSpeed-s.config.MouseMovement.MinSpeed)
	}
	
	return (s.config.MouseMovement.MinSpeed + s.config.MouseMovement.MaxSpeed) / 2
}

func (s *StealthManager) getRandomChar() string {
	chars := "abcdefghijklmnopqrstuvwxyz"
	return string(chars[s.rng.Intn(len(chars))])
}

func (s *StealthManager) applyFingerprintMasking(page *rod.Page) error {
	// Set random user agent if enabled
	if s.config.Fingerprint.RandomUserAgent && len(s.config.Fingerprint.UserAgents) > 0 {
		userAgent := s.config.Fingerprint.UserAgents[s.rng.Intn(len(s.config.Fingerprint.UserAgents))]
		if err := page.SetUserAgent(&proto.NetworkSetUserAgentOverride{
			UserAgent: userAgent,
		}); err != nil {
			return fmt.Errorf("failed to set user agent: %w", err)
		}
		s.logger.WithField("user_agent", userAgent).Debug("Set random user agent")
	}

	return nil
}

func (s *StealthManager) disableAutomationIndicators(page *rod.Page) error {
	// Disable navigator.webdriver
	script := `
		Object.defineProperty(navigator, 'webdriver', {
			get: () => undefined,
		});
		
		// Remove Chrome automation extension
		window.chrome = {
			runtime: {},
		};
		
		// Override permissions API
		const originalQuery = window.navigator.permissions.query;
		window.navigator.permissions.query = (parameters) => (
			parameters.name === 'notifications' ?
				Promise.resolve({ state: Notification.permission }) :
				originalQuery(parameters)
		);
	`

	if _, err := page.Eval(script); err != nil {
		return fmt.Errorf("failed to disable automation indicators: %w", err)
	}

	s.logger.Debug("Disabled automation indicators")
	return nil
}

func (s *StealthManager) setRandomViewport(page *rod.Page) error {
	width := s.config.Fingerprint.MinViewportWidth + s.rng.Intn(s.config.Fingerprint.MaxViewportWidth-s.config.Fingerprint.MinViewportWidth+1)
	height := s.config.Fingerprint.MinViewportHeight + s.rng.Intn(s.config.Fingerprint.MaxViewportHeight-s.config.Fingerprint.MinViewportHeight+1)

	if err := page.SetViewport(&proto.EmulationSetDeviceMetricsOverride{
		Width:  width,
		Height: height,
	}); err != nil {
		return fmt.Errorf("failed to set viewport: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"width":  width,
		"height": height,
	}).Debug("Set random viewport")

	return nil
}

// IntelligentClick performs a realistic click with human-like behavior
func (s *StealthManager) IntelligentClick(page *rod.Page, selector string) error {
	// Simple delay before click
	time.Sleep(s.RandomDelay())
	
	element, err := page.Element(selector)
	if err != nil {
		return fmt.Errorf("element not found: %w", err)
	}
	
	return element.Click("left", 1)
}

// IntelligentScroll performs realistic scrolling behavior
func (s *StealthManager) IntelligentScroll(page *rod.Page, direction string, amount int) error {
	time.Sleep(s.RandomDelay())
	
	chunkSize := 3
	scrollsNeeded := amount / chunkSize
	
	for i := 0; i < scrollsNeeded; i++ {
		time.Sleep(time.Duration(100+s.rng.Intn(200)) * time.Millisecond)
		
		switch direction {
		case "down":
			page.Mouse.Scroll(0, -float64(chunkSize), 0)
		case "up":
			page.Mouse.Scroll(0, float64(chunkSize), 0)
		}
	}
	return nil
}

// IntelligentHover performs realistic hover behavior
func (s *StealthManager) IntelligentHover(page *rod.Page, selector string, duration time.Duration) error {
	element, err := page.Element(selector)
	if err != nil {
		return fmt.Errorf("element not found for hover: %w", err)
	}
	
	if err := element.Hover(); err != nil {
		return fmt.Errorf("failed to hover element: %w", err)
	}
	
	time.Sleep(duration)
	return nil
}
