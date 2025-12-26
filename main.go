package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"linkedin-automation/auth"
	"linkedin-automation/config"
	"linkedin-automation/connect"
	"linkedin-automation/logger"
	"linkedin-automation/message"
	"linkedin-automation/search"
	"linkedin-automation/stealth"
	"linkedin-automation/storage"
)

var (
	configFile string
	verbose    bool
	headless   bool
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "linkedin-automation",
		Short: "LinkedIn browser automation tool",
		Long:  `A CLI-based LinkedIn automation tool with stealth capabilities and anti-bot detection techniques.`,
	}

	// Global flags
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "./config/config.yaml", "Configuration file path")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "Enable verbose logging")
	rootCmd.PersistentFlags().BoolVar(&headless, "headless", true, "Run browser in headless mode")

	// Add subcommands
	rootCmd.AddCommand(createSearchCmd())
	rootCmd.AddCommand(createConnectCmd())
	rootCmd.AddCommand(createMessageCmd())
	rootCmd.AddCommand(createStatusCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func createSearchCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "search",
		Short: "Search for LinkedIn users",
		Long:  `Search for LinkedIn users based on keywords, title, company, and location.`,
	}

	cmd.AddCommand(createSearchUsersCmd())
	return cmd
}

func createSearchUsersCmd() *cobra.Command {
	var (
		keywords   string
		title      string
		company    string
		location   string
		maxResults int
		output     string
	)

	var cmd = &cobra.Command{
		Use:   "users",
		Short: "Search for LinkedIn users by criteria",
		Long:  `Search for LinkedIn users using keywords, title, company, and location filters.`,
		RunE:  runSearchUsers,
	}

	cmd.Flags().StringVar(&keywords, "keywords", "", "Search keywords")
	cmd.Flags().StringVar(&title, "title", "", "Job title filter")
	cmd.Flags().StringVar(&company, "company", "", "Company filter")
	cmd.Flags().StringVar(&location, "location", "", "Location filter")
	cmd.Flags().IntVar(&maxResults, "max-results", 100, "Maximum number of results")
	cmd.Flags().StringVar(&output, "output", "", "Output file path")

	return cmd
}

func createConnectCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "connect",
		Short: "Send connection requests",
		Long:  `Send connection requests to LinkedIn users with stealth techniques.`,
	}

	cmd.AddCommand(createConnectToProfilesCmd())
	return cmd
}

func createConnectToProfilesCmd() *cobra.Command {
	var (
		profiles string
		message  string
		template string
	)

	var cmd = &cobra.Command{
		Use:   "to-profiles",
		Short: "Send connection requests to specific profiles",
		Long:  `Send connection requests to a list of LinkedIn profile URLs.`,
		RunE:  runConnectToProfiles,
	}

	cmd.Flags().StringVar(&profiles, "profiles", "", "Comma-separated list of profile URLs")
	cmd.Flags().StringVar(&message, "message", "", "Connection message")
	cmd.Flags().StringVar(&template, "template", "professional", "Message template")

	return cmd
}

func createMessageCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "message",
		Short: "Send messages",
		Long:  `Send messages to LinkedIn connections with stealth techniques.`,
	}

	cmd.AddCommand(createSendMessageCmd())
	return cmd
}

func createSendMessageCmd() *cobra.Command {
	var (
		recipients string
		message   string
		template  string
	)

	var cmd = &cobra.Command{
		Use:   "send",
		Short: "Send messages to recipients",
		Long:  `Send messages to a list of LinkedIn profile URLs.`,
		RunE:  runSendMessage,
	}

	cmd.Flags().StringVar(&recipients, "recipients", "", "Comma-separated list of recipient URLs")
	cmd.Flags().StringVar(&message, "message", "", "Message content")
	cmd.Flags().StringVar(&template, "template", "follow_up_professional", "Message template")

	return cmd
}

func createStatusCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "status",
		Short: "Show status and statistics",
		Long:  `Display current status, statistics, and configuration information.`,
		RunE:  runStatus,
	}

	return cmd
}

// Command runners

func runSearchUsers(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := setupLogger(cfg.Logging.Level); err != nil {
		return fmt.Errorf("failed to setup logger: %w", err)
	}

	// Get flags
	keywords, _ := cmd.Flags().GetString("keywords")
	title, _ := cmd.Flags().GetString("title")
	company, _ := cmd.Flags().GetString("company")
	location, _ := cmd.Flags().GetString("location")
	maxResults, _ := cmd.Flags().GetInt("maxResults")
	output, _ := cmd.Flags().GetString("output")

	ctx := context.Background()
	
	// Initialize auth manager
	authManager := auth.NewAuthManager(cfg.LinkedIn.Email, cfg.LinkedIn.Password, "./sessions", logger.GetLogger())
	
	// Initialize browser
	if err := authManager.InitializeBrowser(headless, cfg.Browser.UserAgent); err != nil {
		return fmt.Errorf("failed to initialize browser: %w", err)
	}
	defer authManager.Close()

	// Authenticate
	loginResult, err := authManager.Login(ctx)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}
	if !loginResult.Success {
		return fmt.Errorf("authentication unsuccessful: %s", loginResult.ErrorMessage)
	}

	// Get authenticated page
	page, err := authManager.GetAuthenticatedPage(ctx)
	if err != nil {
		return fmt.Errorf("failed to get authenticated page: %w", err)
	}
	defer page.Close()

	// Initialize stealth manager
	stealthConfig := convertConfigToStealth(cfg.Stealth)
	stealthManager := stealth.NewStealthManager(stealthConfig, logger.GetLogger())
	
	// Apply stealth
	if err := stealthManager.ApplyStealth(page); err != nil {
		logger.GetLogger().WithError(err).Warn("Failed to apply some stealth features")
	}

	// Initialize search manager
	searchManager := search.NewSearchManager(page, logger.GetLogger())

	// Create search query
	query := search.SearchQuery{
		Keywords:   keywords,
		Title:      title,
		Company:    company,
		Location:   location,
		MaxResults: maxResults,
	}

	// Perform search
	session, err := searchManager.SearchUsers(ctx, query)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	// Output results
	fmt.Printf("Search completed successfully!\n")
	fmt.Printf("Found %d results in %v\n", len(session.Results), session.Duration)
	fmt.Printf("Unique profiles: %d\n", len(session.Profiles))

	if output != "" {
		if err := saveSearchResults(session, output); err != nil {
			logger.GetLogger().WithError(err).Error("Failed to save results")
		} else {
			fmt.Printf("Results saved to: %s\n", output)
		}
	} else {
		// Print results to console
		for i, result := range session.Results {
			fmt.Printf("%d. %s - %s\n", i+1, result.Name, result.Title)
			fmt.Printf("   %s\n", result.ProfileURL)
		}
	}

	return nil
}

func runConnectToProfiles(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := setupLogger(cfg.Logging.Level); err != nil {
		return fmt.Errorf("failed to setup logger: %w", err)
	}

	// Get flags
	profiles, _ := cmd.Flags().GetString("profiles")
	message, _ := cmd.Flags().GetString("message")
	template, _ := cmd.Flags().GetString("template")

	ctx := context.Background()
	
	// Initialize auth
	authManager := auth.NewAuthManager(cfg.LinkedIn.Email, cfg.LinkedIn.Password, "./sessions", logger.GetLogger())
	
	if err := authManager.InitializeBrowser(headless, cfg.Browser.UserAgent); err != nil {
		return fmt.Errorf("failed to initialize browser: %w", err)
	}
	defer authManager.Close()

	loginResult, err := authManager.Login(ctx)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}
	if !loginResult.Success {
		return fmt.Errorf("authentication unsuccessful: %s", loginResult.ErrorMessage)
	}

	page, err := authManager.GetAuthenticatedPage(ctx)
	if err != nil {
		return fmt.Errorf("failed to get authenticated page: %w", err)
	}
	defer page.Close()

	// Initialize stealth
	stealthConfig := convertConfigToStealth(cfg.Stealth)
	stealthManager := stealth.NewStealthManager(stealthConfig, logger.GetLogger())
	
	if err := stealthManager.ApplyStealth(page); err != nil {
		logger.GetLogger().WithError(err).Warn("Failed to apply some stealth features")
	}

	// Initialize connect manager
	connectManager := connect.NewConnectManager(page, logger.GetLogger(), stealthManager)

	// Parse profiles
	profileList := parseCommaSeparated(profiles)
	if len(profileList) == 0 {
		return fmt.Errorf("no profiles provided")
	}

	// Get message template
	connectionMessage := message
	if connectionMessage == "" {
		templates := connect.GetDefaultTemplates()
		for _, t := range templates {
			if t.ID == template {
				connectionMessage = t.Content
				break
			}
		}
		if connectionMessage == "" {
			connectionMessage = "Hi, I'd like to connect with you on LinkedIn."
		}
	}

	// Send connection requests
	results, err := connectManager.BatchSendConnectionRequests(ctx, profileList, connectionMessage)
	if err != nil {
		return fmt.Errorf("batch connection failed: %w", err)
	}

	// Report results
	successCount := 0
	for _, result := range results {
		if result.Success {
			successCount++
		}
	}

	fmt.Printf("Connection requests completed!\n")
	fmt.Printf("Total profiles: %d\n", len(profileList))
	fmt.Printf("Successful: %d\n", successCount)
	fmt.Printf("Failed: %d\n", len(results)-successCount)

	return nil
}

func runSendMessage(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := setupLogger(cfg.Logging.Level); err != nil {
		return fmt.Errorf("failed to setup logger: %w", err)
	}

	// Get flags
	recipients, _ := cmd.Flags().GetString("recipients")
	messageText, _ := cmd.Flags().GetString("message")
	template, _ := cmd.Flags().GetString("template")

	ctx := context.Background()
	
	// Initialize auth
	authManager := auth.NewAuthManager(cfg.LinkedIn.Email, cfg.LinkedIn.Password, "./sessions", logger.GetLogger())
	
	if err := authManager.InitializeBrowser(headless, cfg.Browser.UserAgent); err != nil {
		return fmt.Errorf("failed to initialize browser: %w", err)
	}
	defer authManager.Close()

	loginResult, err := authManager.Login(ctx)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}
	if !loginResult.Success {
		return fmt.Errorf("authentication unsuccessful: %s", loginResult.ErrorMessage)
	}

	page, err := authManager.GetAuthenticatedPage(ctx)
	if err != nil {
		return fmt.Errorf("failed to get authenticated page: %w", err)
	}
	defer page.Close()

	// Initialize stealth
	stealthConfig := convertConfigToStealth(cfg.Stealth)
	stealthManager := stealth.NewStealthManager(stealthConfig, logger.GetLogger())
	
	if err := stealthManager.ApplyStealth(page); err != nil {
		logger.GetLogger().WithError(err).Warn("Failed to apply some stealth features")
	}

	// Initialize message manager
	messageManager := message.NewMessageManager(page, logger.GetLogger(), stealthManager)

	// Parse recipients
	recipientList := parseCommaSeparated(recipients)
	if len(recipientList) == 0 {
		return fmt.Errorf("no recipients provided")
	}

	// Get message template
	messageContent := messageText
	if messageContent == "" {
		templates := message.GetDefaultMessageTemplates()
		for _, t := range templates {
			if t.ID == template {
				messageContent = t.Content
				break
			}
		}
		if messageContent == "" {
			messageContent = "Hi, thanks for connecting!"
		}
	}

	// Send messages
	results, err := messageManager.BatchSendMessages(ctx, recipientList, messageContent)
	if err != nil {
		return fmt.Errorf("batch messaging failed: %w", err)
	}

	// Report results
	successCount := 0
	for _, result := range results {
		if result.Success {
			successCount++
		}
	}

	fmt.Printf("Messages sent successfully!\n")
	fmt.Printf("Total recipients: %d\n", len(recipientList))
	fmt.Printf("Successful: %d\n", successCount)
	fmt.Printf("Failed: %d\n", len(results)-successCount)

	return nil
}

func runStatus(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := setupLogger(cfg.Logging.Level); err != nil {
		return fmt.Errorf("failed to setup logger: %w", err)
	}

	// Initialize database
	db, err := storage.NewDatabase(cfg.Storage.Path, logger.GetLogger())
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer db.Close()

	// Get daily stats
	stats, err := db.GetDailyStats(time.Now())
	if err != nil {
		return fmt.Errorf("failed to get daily stats: %w", err)
	}

	// Display status
	fmt.Printf("LinkedIn Automation Status\n")
	fmt.Printf("========================\n\n")
	fmt.Printf("Configuration:\n")
	fmt.Printf("  Config file: %s\n", configFile)
	fmt.Printf("  Headless: %v\n", headless)
	fmt.Printf("  LinkedIn email: %s\n", maskEmail(cfg.LinkedIn.Email))
	fmt.Printf("\n")
	fmt.Printf("Daily Statistics:\n")
	fmt.Printf("  Connections sent: %d\n", stats["connections_sent"])
	fmt.Printf("  Connections accepted: %d\n", stats["connections_accepted"])
	fmt.Printf("  Messages sent: %d\n", stats["messages_sent"])
	fmt.Printf("\n")
	fmt.Printf("Limits:\n")
	fmt.Printf("  Daily connections: %d/%d\n", stats["connections_sent"], cfg.Limits.DailyConnections)
	fmt.Printf("  Daily messages: %d/%d\n", stats["messages_sent"], cfg.Limits.DailyMessages)

	return nil
}

// Helper functions

func setupLogger(level string) error {
	logLevel := "info"
	if verbose {
		logLevel = "debug"
	}
	if level != "" {
		logLevel = level
	}

	return logger.InitLogger(logLevel, "json", "stdout", 100, 3, 28)
}

func convertConfigToStealth(cfg config.StealthConfig) stealth.StealthConfig {
	return stealth.StealthConfig{
		MouseMovement: stealth.MouseMovementConfig{
			BezierCurves:     cfg.MouseMovement.BezierCurves,
			VariableSpeed:    cfg.MouseMovement.VariableSpeed,
			Overshoot:        cfg.MouseMovement.Overshoot,
			MicroCorrections: cfg.MouseMovement.MicroCorrections,
			MinSpeed:         cfg.MouseMovement.MinSpeed,
			MaxSpeed:         cfg.MouseMovement.MaxSpeed,
			IdleMovements:    cfg.MouseMovement.IdleMovements,
			IdleProbability:  cfg.MouseMovement.IdleProbability,
		},
		Timing: stealth.TimingConfig{
			MinDelay:    cfg.Timing.MinDelay,
			MaxDelay:    cfg.Timing.MaxDelay,
			ThinkTime:   cfg.Timing.ThinkTime,
			ScrollDelay: cfg.Timing.ScrollDelay,
			ClickDelay:  cfg.Timing.ClickDelay,
			TypeDelay:   cfg.Timing.TypeDelay,
		},
		Typing: stealth.TypingConfig{
			VariableSpeed:   cfg.Typing.VariableSpeed,
			TypoRate:       cfg.Typing.TypoRate,
			CorrectionDelay: cfg.Typing.CorrectionDelay,
			MinSpeed:       cfg.Typing.MinSpeed,
			MaxSpeed:       cfg.Typing.MaxSpeed,
		},
		Scrolling: stealth.ScrollingConfig{
			VariableSpeed: cfg.Scrolling.VariableSpeed,
			Acceleration:  cfg.Scrolling.Acceleration,
			Deceleration:  cfg.Scrolling.Deceleration,
			ScrollBack:    cfg.Scrolling.ScrollBack,
			MinSpeed:      cfg.Scrolling.MinSpeed,
			MaxSpeed:      cfg.Scrolling.MaxSpeed,
		},
		Schedule: stealth.ScheduleConfig{
			BusinessHoursOnly: cfg.Schedule.BusinessHoursOnly,
			StartHour:        cfg.Schedule.StartHour,
			EndHour:          cfg.Schedule.EndHour,
			BreakDuration:    cfg.Schedule.BreakDuration,
			BreakFrequency:   cfg.Schedule.BreakFrequency,
			Timezone:         cfg.Schedule.Timezone,
		},
		Fingerprint: stealth.FingerprintConfig{
			RandomUserAgent:    cfg.Fingerprint.RandomUserAgent,
			RandomViewport:     cfg.Fingerprint.RandomViewport,
			MinViewportWidth:   cfg.Fingerprint.MinViewportWidth,
			MaxViewportWidth:   cfg.Fingerprint.MaxViewportWidth,
			MinViewportHeight:  cfg.Fingerprint.MinViewportHeight,
			MaxViewportHeight:  cfg.Fingerprint.MaxViewportHeight,
			UserAgents:         cfg.Fingerprint.UserAgents,
		},
	}
}

func parseCommaSeparated(input string) []string {
	var result []string
	for _, item := range strings.Split(input, ",") {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func saveSearchResults(session *search.SearchSession, outputPath string) error {
	data := map[string]interface{}{
		"query":        session.Query,
		"results":      session.Results,
		"profiles":     session.Profiles,
		"search_time":  session.SearchTime,
		"duration":     session.Duration,
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal search results: %w", err)
	}

	// Create directory if needed
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Write to file
	return os.WriteFile(outputPath, jsonData, 0644)
}

func maskEmail(email string) string {
	if email == "" {
		return ""
	}
	
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return email
	}
	
	username := parts[0]
	domain := parts[1]
	
	if len(username) <= 2 {
		return email
	}
	
	masked := username[:2] + strings.Repeat("*", len(username)-2)
	return masked + "@" + domain
}
