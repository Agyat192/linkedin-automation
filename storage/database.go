package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/sirupsen/logrus"
)

// Database represents the SQLite database connection
type Database struct {
	db     *sql.DB
	logger *logrus.Logger
}

// Profile represents a LinkedIn profile
type Profile struct {
	ID          int       `json:"id"`
	URL         string    `json:"url"`
	Name        string    `json:"name"`
	Title       string    `json:"title"`
	Company     string    `json:"company"`
	Location    string    `json:"location"`
	SearchQuery string    `json:"search_query"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ConnectionRequest represents a sent connection request
type ConnectionRequest struct {
	ID          int       `json:"id"`
	ProfileURL  string    `json:"profile_url"`
	Message     string    `json:"message"`
	Status      string    `json:"status"` // pending, accepted, rejected
	SentAt      time.Time `json:"sent_at"`
	AcceptedAt  *time.Time `json:"accepted_at,omitempty"`
}

// Message represents a sent message
type Message struct {
	ID             int       `json:"id"`
	RecipientURL   string    `json:"recipient_url"`
	Content        string    `json:"content"`
	Type           string    `json:"type"` // connection_note, follow_up
	Status         string    `json:"status"` // sent, failed
	SentAt         time.Time `json:"sent_at"`
	ConnectionID   *int      `json:"connection_id,omitempty"`
}

// SearchSession represents a search session
type SearchSession struct {
	ID          int       `json:"id"`
	Query       string    `json:"query"`
	ResultsCount int      `json:"results_count"`
	CreatedAt   time.Time `json:"created_at"`
}

// NewDatabase creates a new database connection
func NewDatabase(dbPath string, logger *logrus.Logger) (*Database, error) {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open database connection
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	database := &Database{
		db:     db,
		logger: logger,
	}

	// Initialize tables
	if err := database.initTables(); err != nil {
		return nil, fmt.Errorf("failed to initialize tables: %w", err)
	}

	logger.Info("Database initialized successfully")
	return database, nil
}

// initTables creates all necessary tables
func (d *Database) initTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS profiles (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			url TEXT UNIQUE NOT NULL,
			name TEXT,
			title TEXT,
			company TEXT,
			location TEXT,
			search_query TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS connection_requests (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			profile_url TEXT NOT NULL,
			message TEXT,
			status TEXT DEFAULT 'pending',
			sent_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			accepted_at DATETIME,
			FOREIGN KEY (profile_url) REFERENCES profiles(url)
		)`,
		`CREATE TABLE IF NOT EXISTS messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			recipient_url TEXT NOT NULL,
			content TEXT NOT NULL,
			type TEXT NOT NULL,
			status TEXT DEFAULT 'sent',
			sent_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			connection_id INTEGER,
			FOREIGN KEY (connection_id) REFERENCES connection_requests(id)
		)`,
		`CREATE TABLE IF NOT EXISTS search_sessions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			query TEXT NOT NULL,
			results_count INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_profiles_url ON profiles(url)`,
		`CREATE INDEX IF NOT EXISTS idx_connection_requests_profile_url ON connection_requests(profile_url)`,
		`CREATE INDEX IF NOT EXISTS idx_connection_requests_status ON connection_requests(status)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_recipient_url ON messages(recipient_url)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_status ON messages(status)`,
	}

	for _, query := range queries {
		if _, err := d.db.Exec(query); err != nil {
			return fmt.Errorf("failed to execute query: %s, error: %w", query, err)
		}
	}

	d.logger.Info("Database tables initialized successfully")
	return nil
}

// Close closes the database connection
func (d *Database) Close() error {
	return d.db.Close()
}

// SaveProfile saves a profile to the database
func (d *Database) SaveProfile(profile *Profile) error {
	query := `INSERT OR REPLACE INTO profiles (url, name, title, company, location, search_query, updated_at) 
			  VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`

	result, err := d.db.Exec(query, profile.URL, profile.Name, profile.Title, profile.Company, profile.Location, profile.SearchQuery)
	if err != nil {
		return fmt.Errorf("failed to save profile: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get profile ID: %w", err)
	}

	profile.ID = int(id)
	d.logger.WithField("profile_url", profile.URL).Debug("Profile saved")
	return nil
}

// GetProfile retrieves a profile by URL
func (d *Database) GetProfile(url string) (*Profile, error) {
	query := `SELECT id, url, name, title, company, location, search_query, created_at, updated_at 
			  FROM profiles WHERE url = ?`

	row := d.db.QueryRow(query, url)
	var profile Profile
	err := row.Scan(&profile.ID, &profile.URL, &profile.Name, &profile.Title, &profile.Company, &profile.Location, &profile.SearchQuery, &profile.CreatedAt, &profile.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}

	return &profile, nil
}

// SaveConnectionRequest saves a connection request
func (d *Database) SaveConnectionRequest(request *ConnectionRequest) error {
	query := `INSERT INTO connection_requests (profile_url, message, status, sent_at) 
			  VALUES (?, ?, ?, ?)`

	result, err := d.db.Exec(query, request.ProfileURL, request.Message, request.Status, request.SentAt)
	if err != nil {
		return fmt.Errorf("failed to save connection request: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get connection request ID: %w", err)
	}

	request.ID = int(id)
	d.logger.WithField("profile_url", request.ProfileURL).Debug("Connection request saved")
	return nil
}

// GetPendingConnectionRequests retrieves all pending connection requests
func (d *Database) GetPendingConnectionRequests() ([]*ConnectionRequest, error) {
	query := `SELECT id, profile_url, message, status, sent_at, accepted_at 
			  FROM connection_requests WHERE status = 'pending'`

	rows, err := d.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending connection requests: %w", err)
	}
	defer rows.Close()

	var requests []*ConnectionRequest
	for rows.Next() {
		var request ConnectionRequest
		err := rows.Scan(&request.ID, &request.ProfileURL, &request.Message, &request.Status, &request.SentAt, &request.AcceptedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan connection request: %w", err)
		}
		requests = append(requests, &request)
	}

	return requests, nil
}

// UpdateConnectionRequestStatus updates the status of a connection request
func (d *Database) UpdateConnectionRequestStatus(id int, status string) error {
	query := `UPDATE connection_requests SET status = ?`
	var args []interface{}
	args = append(args, status)

	if status == "accepted" {
		query += `, accepted_at = CURRENT_TIMESTAMP`
	}

	query += ` WHERE id = ?`
	args = append(args, id)

	_, err := d.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to update connection request status: %w", err)
	}

	d.logger.WithFields(logrus.Fields{
		"id":     id,
		"status": status,
	}).Debug("Connection request status updated")
	return nil
}

// SaveMessage saves a message
func (d *Database) SaveMessage(message *Message) error {
	query := `INSERT INTO messages (recipient_url, content, type, status, sent_at, connection_id) 
			  VALUES (?, ?, ?, ?, ?, ?)`

	result, err := d.db.Exec(query, message.RecipientURL, message.Content, message.Type, message.Status, message.SentAt, message.ConnectionID)
	if err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get message ID: %w", err)
	}

	message.ID = int(id)
	d.logger.WithField("recipient_url", message.RecipientURL).Debug("Message saved")
	return nil
}

// GetMessagesByRecipient retrieves all messages for a recipient
func (d *Database) GetMessagesByRecipient(recipientURL string) ([]*Message, error) {
	query := `SELECT id, recipient_url, content, type, status, sent_at, connection_id 
			  FROM messages WHERE recipient_url = ? ORDER BY sent_at DESC`

	rows, err := d.db.Query(query, recipientURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}
	defer rows.Close()

	var messages []*Message
	for rows.Next() {
		var message Message
		err := rows.Scan(&message.ID, &message.RecipientURL, &message.Content, &message.Type, &message.Status, &message.SentAt, &message.ConnectionID)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		messages = append(messages, &message)
	}

	return messages, nil
}

// SaveSearchSession saves a search session
func (d *Database) SaveSearchSession(session *SearchSession) error {
	query := `INSERT INTO search_sessions (query, results_count, created_at) 
			  VALUES (?, ?, ?)`

	result, err := d.db.Exec(query, session.Query, session.ResultsCount, session.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to save search session: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get search session ID: %w", err)
	}

	session.ID = int(id)
	d.logger.WithField("query", session.Query).Debug("Search session saved")
	return nil
}

// GetDailyStats retrieves daily statistics
func (d *Database) GetDailyStats(date time.Time) (map[string]int, error) {
	query := `
		SELECT 
			(SELECT COUNT(*) FROM connection_requests WHERE DATE(sent_at) = DATE(?)) as connections_sent,
			(SELECT COUNT(*) FROM connection_requests WHERE status = 'accepted' AND DATE(accepted_at) = DATE(?)) as connections_accepted,
			(SELECT COUNT(*) FROM messages WHERE DATE(sent_at) = DATE(?)) as messages_sent
	`

	row := d.db.QueryRow(query, date, date, date)
	var connectionsSent, connectionsAccepted, messagesSent int
	err := row.Scan(&connectionsSent, &connectionsAccepted, &messagesSent)
	if err != nil {
		return nil, fmt.Errorf("failed to get daily stats: %w", err)
	}

	stats := map[string]int{
		"connections_sent":     connectionsSent,
		"connections_accepted": connectionsAccepted,
		"messages_sent":        messagesSent,
	}

	return stats, nil
}

// ExportData exports all data to JSON format
func (d *Database) ExportData() (map[string]interface{}, error) {
	data := make(map[string]interface{})

	// Export profiles
	profiles, err := d.getAllProfiles()
	if err != nil {
		return nil, fmt.Errorf("failed to export profiles: %w", err)
	}
	data["profiles"] = profiles

	// Export connection requests
	requests, err := d.getAllConnectionRequests()
	if err != nil {
		return nil, fmt.Errorf("failed to export connection requests: %w", err)
	}
	data["connection_requests"] = requests

	// Export messages
	messages, err := d.getAllMessages()
	if err != nil {
		return nil, fmt.Errorf("failed to export messages: %w", err)
	}
	data["messages"] = messages

	// Export search sessions
	sessions, err := d.getAllSearchSessions()
	if err != nil {
		return nil, fmt.Errorf("failed to export search sessions: %w", err)
	}
	data["search_sessions"] = sessions

	return data, nil
}

// Helper methods for data export
func (d *Database) getAllProfiles() ([]*Profile, error) {
	query := `SELECT id, url, name, title, company, location, search_query, created_at, updated_at FROM profiles`
	rows, err := d.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var profiles []*Profile
	for rows.Next() {
		var profile Profile
		err := rows.Scan(&profile.ID, &profile.URL, &profile.Name, &profile.Title, &profile.Company, &profile.Location, &profile.SearchQuery, &profile.CreatedAt, &profile.UpdatedAt)
		if err != nil {
			return nil, err
		}
		profiles = append(profiles, &profile)
	}
	return profiles, nil
}

func (d *Database) getAllConnectionRequests() ([]*ConnectionRequest, error) {
	query := `SELECT id, profile_url, message, status, sent_at, accepted_at FROM connection_requests`
	rows, err := d.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var requests []*ConnectionRequest
	for rows.Next() {
		var request ConnectionRequest
		err := rows.Scan(&request.ID, &request.ProfileURL, &request.Message, &request.Status, &request.SentAt, &request.AcceptedAt)
		if err != nil {
			return nil, err
		}
		requests = append(requests, &request)
	}
	return requests, nil
}

func (d *Database) getAllMessages() ([]*Message, error) {
	query := `SELECT id, recipient_url, content, type, status, sent_at, connection_id FROM messages`
	rows, err := d.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*Message
	for rows.Next() {
		var message Message
		err := rows.Scan(&message.ID, &message.RecipientURL, &message.Content, &message.Type, &message.Status, &message.SentAt, &message.ConnectionID)
		if err != nil {
			return nil, err
		}
		messages = append(messages, &message)
	}
	return messages, nil
}

func (d *Database) getAllSearchSessions() ([]*SearchSession, error) {
	query := `SELECT id, query, results_count, created_at FROM search_sessions`
	rows, err := d.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*SearchSession
	for rows.Next() {
		var session SearchSession
		err := rows.Scan(&session.ID, &session.Query, &session.ResultsCount, &session.CreatedAt)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, &session)
	}
	return sessions, nil
}
