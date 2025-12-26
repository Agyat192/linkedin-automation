package search

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/sirupsen/logrus"
)

// SearchManager handles LinkedIn user search
type SearchManager struct {
	page   *rod.Page
	logger *logrus.Logger
}

// SearchQuery represents a search query
type SearchQuery struct {
	Keywords    string
	Title       string
	Company     string
	Location    string
	MaxResults  int
}

// SearchResult represents a search result
type SearchResult struct {
	URL         string
	Name        string
	Title       string
	Company     string
	Location    string
	ProfileURL  string
	SearchQuery string
}

// SearchSession represents a complete search session
type SearchSession struct {
	Query        SearchQuery
	Results      []*SearchResult
	Profiles     []string // Unique profile URLs
	SearchTime   time.Time
	Duration     time.Duration
}

// NewSearchManager creates a new search manager
func NewSearchManager(page *rod.Page, logger *logrus.Logger) *SearchManager {
	return &SearchManager{
		page:   page,
		logger: logger,
	}
}

// SearchUsers searches for LinkedIn users based on query parameters
func (s *SearchManager) SearchUsers(ctx context.Context, query SearchQuery) (*SearchSession, error) {
	s.logger.WithFields(logrus.Fields{
		"keywords": query.Keywords,
		"title":    query.Title,
		"company":  query.Company,
		"location": query.Location,
	}).Info("Starting user search")

	startTime := time.Now()
	session := &SearchSession{
		Query:      query,
		Results:    make([]*SearchResult, 0),
		Profiles:   make([]string, 0),
		SearchTime: startTime,
	}

	// Build search URL
	searchURL := s.buildSearchURL(query)
	s.logger.WithField("url", searchURL).Debug("Navigating to search page")

	// Navigate to search page
	if err := s.page.Navigate(searchURL); err != nil {
		return nil, fmt.Errorf("failed to navigate to search page: %w", err)
	}

	// Wait for page to load
	if err := s.page.WaitLoad(); err != nil {
		return nil, fmt.Errorf("failed to wait for page load: %w", err)
	}

	// Handle potential login redirect
	if err := s.handleLoginRedirect(); err != nil {
		return nil, fmt.Errorf("login redirect failed: %w", err)
	}

	// Wait for search results to load
	if err := s.waitForSearchResults(); err != nil {
		return nil, fmt.Errorf("failed to wait for search results: %w", err)
	}

	// Extract results from current page
	if err := s.extractResultsFromPage(session); err != nil {
		return nil, fmt.Errorf("failed to extract results: %w", err)
	}

	// Handle pagination if needed
	if len(session.Results) < query.MaxResults {
		if err := s.handlePagination(session, query.MaxResults); err != nil {
			s.logger.WithError(err).Warn("Failed to handle pagination")
		}
	}

	// Limit results to max requested
	if len(session.Results) > query.MaxResults {
		session.Results = session.Results[:query.MaxResults]
	}

	// Extract unique profile URLs
	for _, result := range session.Results {
		if result.ProfileURL != "" {
			session.Profiles = append(session.Profiles, result.ProfileURL)
		}
	}

	session.Duration = time.Since(startTime)

	s.logger.WithFields(logrus.Fields{
		"results_found": len(session.Results),
		"unique_profiles": len(session.Profiles),
		"duration": session.Duration,
	}).Info("Search completed successfully")

	return session, nil
}

// SearchByURL searches for users using a direct search URL
func (s *SearchManager) SearchByURL(ctx context.Context, searchURL string, maxResults int) (*SearchSession, error) {
	s.logger.WithField("url", searchURL).Info("Starting search by URL")

	startTime := time.Now()
	session := &SearchSession{
		Query: SearchQuery{
			MaxResults: maxResults,
		},
		Results:    make([]*SearchResult, 0),
		Profiles:   make([]string, 0),
		SearchTime: startTime,
	}

	// Navigate to search URL
	if err := s.page.Navigate(searchURL); err != nil {
		return nil, fmt.Errorf("failed to navigate to search URL: %w", err)
	}

	// Wait for page to load
	if err := s.page.WaitLoad(); err != nil {
		return nil, fmt.Errorf("failed to wait for page load: %w", err)
	}

	// Handle potential login redirect
	if err := s.handleLoginRedirect(); err != nil {
		return nil, fmt.Errorf("login redirect failed: %w", err)
	}

	// Wait for search results to load
	if err := s.waitForSearchResults(); err != nil {
		return nil, fmt.Errorf("failed to wait for search results: %w", err)
	}

	// Extract results from current page
	if err := s.extractResultsFromPage(session); err != nil {
		return nil, fmt.Errorf("failed to extract results: %w", err)
	}

	// Handle pagination if needed
	if len(session.Results) < maxResults {
		if err := s.handlePagination(session, maxResults); err != nil {
			s.logger.WithError(err).Warn("Failed to handle pagination")
		}
	}

	// Limit results to max requested
	if len(session.Results) > maxResults {
		session.Results = session.Results[:maxResults]
	}

	// Extract unique profile URLs
	for _, result := range session.Results {
		if result.ProfileURL != "" {
			session.Profiles = append(session.Profiles, result.ProfileURL)
		}
	}

	session.Duration = time.Since(startTime)

	s.logger.WithFields(logrus.Fields{
		"results_found": len(session.Results),
		"unique_profiles": len(session.Profiles),
		"duration": session.Duration,
	}).Info("URL search completed successfully")

	return session, nil
}

// GetProfileURLsFromSearch extracts profile URLs from search results
func (s *SearchManager) GetProfileURLsFromSearch(ctx context.Context, query SearchQuery) ([]string, error) {
	session, err := s.SearchUsers(ctx, query)
	if err != nil {
		return nil, err
	}

	return session.Profiles, nil
}

// Private helper methods

func (s *SearchManager) buildSearchURL(query SearchQuery) string {
	baseURL := "https://www.linkedin.com/search/results/people/"
	params := url.Values{}

	if query.Keywords != "" {
		params.Add("keywords", query.Keywords)
	}

	// Build filters
	filters := make([]string, 0)

	if query.Title != "" {
		filters = append(filters, fmt.Sprintf("currentCompany:%s", query.Title))
	}

	if query.Company != "" {
		filters = append(filters, fmt.Sprintf("currentCompany:%s", query.Company))
	}

	if query.Location != "" {
		filters = append(filters, fmt.Sprintf("geoUrn:%s", query.Location))
	}

	if len(filters) > 0 {
		params.Add("filters", strings.Join(filters, ","))
	}

	// Add pagination
	params.Add("page", "1")

	if len(params) > 0 {
		return baseURL + "?" + params.Encode()
	}

	return baseURL
}

func (s *SearchManager) handleLoginRedirect() error {
	// Check if we've been redirected to login page
	currentURL, err := s.page.Info()
	if err != nil {
		return err
	}

	if strings.Contains(currentURL.URL, "linkedin.com/login") {
		return fmt.Errorf("redirected to login page - authentication required")
	}

	return nil
}

func (s *SearchManager) waitForSearchResults() error {
	// Try multiple possible selectors
	selectors := []string{
		".search-results__container",
		".reusable-search__result-container",
		".search-results-page",
		"[data-test-id='search-results-container']",
	}

	for _, sel := range selectors {
		element, err := s.page.Element(sel)
		if err == nil && element != nil {
			s.logger.WithField("selector", sel).Debug("Found search results container")
			return nil
		}
	}

	// Wait a bit and try again
	time.Sleep(2 * time.Second)
	
	for _, sel := range selectors {
		element, err := s.page.Element(sel)
		if err == nil && element != nil {
			s.logger.WithField("selector", sel).Debug("Found search results container after delay")
			return nil
		}
	}

	return fmt.Errorf("search results container not found")
}

func (s *SearchManager) extractResultsFromPage(session *SearchSession) error {
	// Try different selectors for search results
	resultSelectors := []string{
		".search-result__info",
		".reusable-search__result-container",
		".people-search-card",
		"[data-test-id='search-result']",
	}

	var results []*rod.Element
	var usedSelector string

	for _, selector := range resultSelectors {
		elements, err := s.page.Elements(selector)
		if err == nil && len(elements) > 0 {
			results = elements
			usedSelector = selector
			s.logger.WithFields(logrus.Fields{
				"selector": selector,
				"count": len(elements),
			}).Debug("Found search results")
			break
		}
	}

	if len(results) == 0 {
		return fmt.Errorf("no search results found")
	}

	// Extract data from each result
	for i, element := range results {
		if i >= session.Query.MaxResults {
			break
		}

		result, err := s.extractResultData(element)
		if err != nil {
			s.logger.WithError(err).WithField("index", i).Warn("Failed to extract result data")
			continue
		}

		result.SearchQuery = fmt.Sprintf("keywords:%s,title:%s,company:%s,location:%s",
			session.Query.Keywords, session.Query.Title, session.Query.Company, session.Query.Location)
		
		session.Results = append(session.Results, result)
	}

	s.logger.WithFields(logrus.Fields{
		"selector": usedSelector,
		"extracted": len(session.Results),
	}).Debug("Extracted search results")

	return nil
}

func (s *SearchManager) extractResultData(element *rod.Element) (*SearchResult, error) {
	result := &SearchResult{}

	// Extract name
	nameElement, err := element.Element(".name span")
	if err != nil {
		// Try alternative selectors
		nameElement, err = element.Element("span[aria-hidden='true']")
		if err != nil {
			nameElement, err = element.Element(".entity-result__title-text")
		}
	}
	
	if err == nil && nameElement != nil {
		name, err := nameElement.Text()
		if err == nil {
			result.Name = strings.TrimSpace(name)
		}
	}

	// Extract title
	titleElement, err := element.Element(".subline-level-1")
	if err != nil {
		titleElement, err = element.Element(".entity-result__primary-subtitle")
	}
	
	if err == nil && titleElement != nil {
		title, err := titleElement.Text()
		if err == nil {
			result.Title = strings.TrimSpace(title)
		}
	}

	// Extract company
	companyElement, err := element.Element(".subline-level-2")
	if err != nil {
		companyElement, err = element.Element(".entity-result__secondary-subtitle")
	}
	
	if err == nil && companyElement != nil {
		company, err := companyElement.Text()
		if err == nil {
			result.Company = strings.TrimSpace(company)
		}
	}

	// Extract location
	locationElement, err := element.Element(".entity-result__simple-insight-text")
	if err == nil && locationElement != nil {
		location, err := locationElement.Text()
		if err == nil {
			result.Location = strings.TrimSpace(location)
		}
	}

	// Extract profile URL
	linkElement, err := element.Element("a")
	if err == nil && linkElement != nil {
		href, err := linkElement.Attribute("href")
		if err == nil && href != nil && *href != "" {
			if strings.HasPrefix(*href, "https://www.linkedin.com/in/") {
				profileURL := *href
				result.ProfileURL = profileURL
			}
		}
	}

	return result, nil
}

func (s *SearchManager) handlePagination(session *SearchSession, maxResults int) error {
	pageNum := 2
	
	for len(session.Results) < maxResults {
		s.logger.WithFields(logrus.Fields{
			"current_results": len(session.Results),
			"target_results": maxResults,
			"page": pageNum,
		}).Debug("Handling pagination")

		// Look for next page button
		nextButton, err := s.page.Element("button[aria-label*='Next']")
		if err != nil {
			nextButton, err = s.page.Element(".pagination__next")
		}
		if err != nil {
			nextButton, err = s.page.Element(".artdeco-pagination__button--next")
		}

		if err != nil || nextButton == nil {
			s.logger.Debug("No more pages available")
			break
		}

		// Check if button is disabled
		disabled, err := nextButton.Attribute("disabled")
		if err == nil && disabled != nil && *disabled != "" {
			s.logger.Debug("Next page button is disabled")
			break
		}

		// Click next button
		if err := nextButton.Click("left", 1); err != nil {
			return fmt.Errorf("failed to click next button: %w", err)
		}

		// Wait for page to load
		if err := s.page.WaitLoad(); err != nil {
			return fmt.Errorf("failed to wait for page load: %w", err)
		}

		// Wait for search results
		if err := s.waitForSearchResults(); err != nil {
			return fmt.Errorf("failed to wait for search results: %w", err)
		}

		// Extract results from this page
		pageResults := make([]*SearchResult, 0)
		if err := s.extractResultsFromPage(&SearchSession{
			Query:   session.Query,
			Results: pageResults,
		}); err != nil {
			s.logger.WithError(err).Warn("Failed to extract results from page")
			break
		}

		// Add new results
		session.Results = append(session.Results, pageResults...)

		pageNum++

		// Safety check to prevent infinite loop
		if pageNum > 100 {
			s.logger.Warn("Reached maximum page limit")
			break
		}

		// Add delay between pages
		time.Sleep(1 * time.Second)
	}

	return nil
}

// GetSearchStats returns statistics about the search
func (s *SearchManager) GetSearchStats(session *SearchSession) map[string]interface{} {
	stats := map[string]interface{}{
		"query":           session.Query,
		"results_count":   len(session.Results),
		"unique_profiles": len(session.Profiles),
		"search_time":     session.SearchTime,
		"duration":        session.Duration,
		"avg_time_per_result": float64(0),
	}

	if len(session.Results) > 0 {
		stats["avg_time_per_result"] = session.Duration.Nanoseconds() / int64(len(session.Results)) / 1e6 // in milliseconds
	}

	return stats
}
