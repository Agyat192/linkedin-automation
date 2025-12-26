package config

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	LinkedIn LinkedInConfig `yaml:"linkedin"`
	Browser  BrowserConfig  `yaml:"browser"`
	Stealth  StealthConfig  `yaml:"stealth"`
	Limits   LimitsConfig   `yaml:"limits"`
	Storage  StorageConfig  `yaml:"storage"`
	Logging  LoggingConfig  `yaml:"logging"`
}

// LinkedInConfig contains LinkedIn-specific settings
type LinkedInConfig struct {
	Email      string `yaml:"email"`
	Password   string `yaml:"password"`
	BaseURL    string `yaml:"base_url"`
	LoginURL   string `yaml:"login_url"`
	SearchURL  string `yaml:"search_url"`
}

// BrowserConfig contains browser automation settings
type BrowserConfig struct {
	Headless          bool          `yaml:"headless"`
	SlowMo            time.Duration `yaml:"slow_mo"`
	ViewportWidth     int           `yaml:"viewport_width"`
	ViewportHeight    int           `yaml:"viewport_height"`
	UserAgent         string        `yaml:"user_agent"`
	ExecutablePath    string        `yaml:"executable_path"`
	ProfileDir        string        `yaml:"profile_dir"`
	DisableWebSecurity bool         `yaml:"disable_web_security"`
}

// StealthConfig contains anti-bot detection settings
type StealthConfig struct {
	MouseMovement     MouseMovementConfig `yaml:"mouse_movement"`
	Timing            TimingConfig        `yaml:"timing"`
	Typing            TypingConfig        `yaml:"typing"`
	Scrolling         ScrollingConfig     `yaml:"scrolling"`
	Schedule          ScheduleConfig      `yaml:"schedule"`
	Fingerprint       FingerprintConfig   `yaml:"fingerprint"`
}

// MouseMovementConfig for realistic mouse behavior
type MouseMovementConfig struct {
	BezierCurves      bool     `yaml:"bezier_curves"`
	VariableSpeed     bool     `yaml:"variable_speed"`
	Overshoot         bool     `yaml:"overshoot"`
	MicroCorrections  bool     `yaml:"micro_corrections"`
	MinSpeed          float64  `yaml:"min_speed"`
	MaxSpeed          float64  `yaml:"max_speed"`
	IdleMovements     bool     `yaml:"idle_movements"`
	IdleProbability   float64  `yaml:"idle_probability"`
}

// TimingConfig for realistic timing patterns
type TimingConfig struct {
	MinDelay          time.Duration `yaml:"min_delay"`
	MaxDelay          time.Duration `yaml:"max_delay"`
	ThinkTime         time.Duration `yaml:"think_time"`
	ScrollDelay       time.Duration `yaml:"scroll_delay"`
	ClickDelay        time.Duration `yaml:"click_delay"`
	TypeDelay         time.Duration `yaml:"type_delay"`
}

// TypingConfig for realistic typing simulation
type TypingConfig struct {
	VariableSpeed     bool          `yaml:"variable_speed"`
	TypoRate          float64       `yaml:"typo_rate"`
	CorrectionDelay   time.Duration `yaml:"correction_delay"`
	MinSpeed          time.Duration `yaml:"min_speed"`
	MaxSpeed          time.Duration `yaml:"max_speed"`
}

// ScrollingConfig for realistic scrolling behavior
type ScrollingConfig struct {
	VariableSpeed     bool          `yaml:"variable_speed"`
	Acceleration      bool          `yaml:"acceleration"`
	Deceleration      bool          `yaml:"deceleration"`
	ScrollBack        bool          `yaml:"scroll_back"`
	MinSpeed          int           `yaml:"min_speed"`
	MaxSpeed          int           `yaml:"max_speed"`
}

// ScheduleConfig for activity scheduling
type ScheduleConfig struct {
	BusinessHoursOnly bool          `yaml:"business_hours_only"`
	StartHour         int           `yaml:"start_hour"`
	EndHour           int           `yaml:"end_hour"`
	BreakDuration     time.Duration `yaml:"break_duration"`
	BreakFrequency    time.Duration `yaml:"break_frequency"`
	Timezone          string        `yaml:"timezone"`
}

// FingerprintConfig for browser fingerprint masking
type FingerprintConfig struct {
	RandomUserAgent   bool     `yaml:"random_user_agent"`
	RandomViewport    bool     `yaml:"random_viewport"`
	MinViewportWidth  int      `yaml:"min_viewport_width"`
	MaxViewportWidth  int      `yaml:"max_viewport_width"`
	MinViewportHeight int      `yaml:"min_viewport_height"`
	MaxViewportHeight int      `yaml:"max_viewport_height"`
	UserAgents        []string `yaml:"user_agents"`
}

// LimitsConfig contains rate limiting settings
type LimitsConfig struct {
	DailyConnections   int           `yaml:"daily_connections"`
	HourlyConnections   int           `yaml:"hourly_connections"`
	DailyMessages      int           `yaml:"daily_messages"`
	HourlyMessages     int           `yaml:"hourly_messages"`
	SearchResults      int           `yaml:"search_results"`
	CooldownPeriod     time.Duration `yaml:"cooldown_period"`
}

// StorageConfig contains database settings
type StorageConfig struct {
	Type     string `yaml:"type"`
	Path     string `yaml:"path"`
	Backup   bool   `yaml:"backup"`
	Interval time.Duration `yaml:"backup_interval"`
}

// LoggingConfig contains logging settings
type LoggingConfig struct {
	Level      string `yaml:"level"`
	Format     string `yaml:"format"`
	Output     string `yaml:"output"`
	MaxSize    int    `yaml:"max_size"`
	MaxBackups int    `yaml:"max_backups"`
	MaxAge     int    `yaml:"max_age"`
}

// LoadConfig loads configuration from file and environment variables
func LoadConfig(configPath string) (*Config, error) {
	// Set default values
	setDefaults()

	// Read config file
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	// Enable environment variable support
	viper.AutomaticEnv()
	viper.SetEnvPrefix("LINKEDIN")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found, create default config
			if err := createDefaultConfig(configPath); err != nil {
				return nil, fmt.Errorf("failed to create default config: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// Override with environment variables
	overrideFromEnv()

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Manually set limits from viper as workaround for unmarshal issue
	config.Limits.DailyConnections = viper.GetInt("limits.daily_connections")
	config.Limits.HourlyConnections = viper.GetInt("limits.hourly_connections")
	config.Limits.DailyMessages = viper.GetInt("limits.daily_messages")
	config.Limits.HourlyMessages = viper.GetInt("limits.hourly_messages")
	config.Limits.SearchResults = viper.GetInt("limits.search_results")
	config.Limits.CooldownPeriod = viper.GetDuration("limits.cooldown_period")

	// Validate configuration
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// setDefaults sets default configuration values
func setDefaults() {
	viper.SetDefault("linkedin.base_url", "https://www.linkedin.com")
	viper.SetDefault("linkedin.login_url", "https://www.linkedin.com/login")
	viper.SetDefault("linkedin.search_url", "https://www.linkedin.com/search/results/people/")

	viper.SetDefault("browser.headless", true)
	viper.SetDefault("browser.slow_mo", "100ms")
	viper.SetDefault("browser.viewport_width", 1920)
	viper.SetDefault("browser.viewport_height", 1080)
	viper.SetDefault("browser.user_agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	viper.SetDefault("browser.disable_web_security", false)

	viper.SetDefault("stealth.mouse_movement.bezier_curves", true)
	viper.SetDefault("stealth.mouse_movement.variable_speed", true)
	viper.SetDefault("stealth.mouse_movement.overshoot", true)
	viper.SetDefault("stealth.mouse_movement.micro_corrections", true)
	viper.SetDefault("stealth.mouse_movement.min_speed", 100)
	viper.SetDefault("stealth.mouse_movement.max_speed", 800)
	viper.SetDefault("stealth.mouse_movement.idle_movements", true)
	viper.SetDefault("stealth.mouse_movement.idle_probability", 0.1)

	viper.SetDefault("stealth.timing.min_delay", "500ms")
	viper.SetDefault("stealth.timing.max_delay", "3s")
	viper.SetDefault("stealth.timing.think_time", "1s")
	viper.SetDefault("stealth.timing.scroll_delay", "300ms")
	viper.SetDefault("stealth.timing.click_delay", "200ms")
	viper.SetDefault("stealth.timing.type_delay", "50ms")

	viper.SetDefault("stealth.typing.variable_speed", true)
	viper.SetDefault("stealth.typing.typo_rate", 0.02)
	viper.SetDefault("stealth.typing.correction_delay", "500ms")
	viper.SetDefault("stealth.typing.min_speed", "50ms")
	viper.SetDefault("stealth.typing.max_speed", "200ms")

	viper.SetDefault("stealth.scrolling.variable_speed", true)
	viper.SetDefault("stealth.scrolling.acceleration", true)
	viper.SetDefault("stealth.scrolling.deceleration", true)
	viper.SetDefault("stealth.scrolling.scroll_back", true)
	viper.SetDefault("stealth.scrolling.min_speed", 200)
	viper.SetDefault("stealth.scrolling.max_speed", 800)

	viper.SetDefault("stealth.schedule.business_hours_only", true)
	viper.SetDefault("stealth.schedule.start_hour", 9)
	viper.SetDefault("stealth.schedule.end_hour", 17)
	viper.SetDefault("stealth.schedule.break_duration", "15m")
	viper.SetDefault("stealth.schedule.break_frequency", "2h")
	viper.SetDefault("stealth.schedule.timezone", "UTC")

	viper.SetDefault("stealth.fingerprint.random_user_agent", true)
	viper.SetDefault("stealth.fingerprint.random_viewport", true)
	viper.SetDefault("stealth.fingerprint.min_viewport_width", 1366)
	viper.SetDefault("stealth.fingerprint.max_viewport_width", 2560)
	viper.SetDefault("stealth.fingerprint.min_viewport_height", 768)
	viper.SetDefault("stealth.fingerprint.max_viewport_height", 1440)

	viper.SetDefault("limits.daily_connections", 50)
	viper.SetDefault("limits.hourly_connections", 10)
	viper.SetDefault("limits.daily_messages", 100)
	viper.SetDefault("limits.hourly_messages", 20)
	viper.SetDefault("limits.search_results", 100)
	viper.SetDefault("limits.cooldown_period", "30m")

	viper.SetDefault("storage.type", "sqlite")
	viper.SetDefault("storage.path", "./data/linkedin.db")
	viper.SetDefault("storage.backup", true)
	viper.SetDefault("storage.backup_interval", "1h")

	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "json")
	viper.SetDefault("logging.output", "stdout")
	viper.SetDefault("logging.max_size", 100)
	viper.SetDefault("logging.max_backups", 3)
	viper.SetDefault("logging.max_age", 28)
}

// createDefaultConfig creates a default configuration file
func createDefaultConfig(configPath string) error {
	config := Config{
		LinkedIn: LinkedInConfig{
			BaseURL:   "https://www.linkedin.com",
			LoginURL:  "https://www.linkedin.com/login",
			SearchURL: "https://www.linkedin.com/search/results/people/",
		},
		Browser: BrowserConfig{
			Headless:       true,
			SlowMo:         100 * time.Millisecond,
			ViewportWidth:  1920,
			ViewportHeight: 1080,
			UserAgent:      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		},
	}

	data, err := yaml.Marshal(&config)
	if err != nil {
		return err
	}

	// Create directory if it doesn't exist
	dir := configPath[:len(configPath)-len("/config.yaml")]
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

// overrideFromEnv overrides configuration with environment variables
func overrideFromEnv() {
	if email := os.Getenv("LINKEDIN_EMAIL"); email != "" {
		viper.Set("linkedin.email", email)
	}
	if password := os.Getenv("LINKEDIN_PASSWORD"); password != "" {
		viper.Set("linkedin.password", password)
	}
}

// validateConfig validates the configuration
func validateConfig(config *Config) error {
	if config.LinkedIn.Email == "" {
		return fmt.Errorf("linkedin email is required")
	}
	if config.LinkedIn.Password == "" {
		return fmt.Errorf("linkedin password is required")
	}
	if config.Limits.DailyConnections <= 0 {
		return fmt.Errorf("daily connections must be positive")
	}
	if config.Limits.HourlyConnections <= 0 {
		return fmt.Errorf("hourly connections must be positive")
	}
	return nil
}
