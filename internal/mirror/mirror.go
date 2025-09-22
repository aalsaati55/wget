package mirror

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"wget/internal/logging"

	"golang.org/x/time/rate"
)

type Options struct {
	RejectTypes  []string
	ExcludeDirs  []string
	ConvertLinks bool
	OutputPath   string
	RateLimit    string
	MaxDepth     int
	MaxFiles     int
}

type MirrorState struct {
	baseURL      *url.URL
	visited      map[string]bool
	pending      []string
	downloaded   map[string]string // URL -> local file path
	mutex        sync.RWMutex
	fileCount    int
	client       *http.Client
	limiter      *rate.Limiter
	logger       *logging.Logger
}

// MirrorWebsite downloads an entire website with recursive crawling
func MirrorWebsite(urlStr string, options *Options, logger *logging.Logger) error {
	logger.LogStart()
	logger.Printf("Starting website mirroring for: %s\n", urlStr)

	// Parse base URL
	baseURL, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL: %v", err)
	}

	// Set default values
	if options.MaxDepth == 0 {
		options.MaxDepth = 5 // Default depth limit
	}
	if options.MaxFiles == 0 {
		options.MaxFiles = 1000 // Default file limit
	}
	if options.OutputPath == "" {
		options.OutputPath = baseURL.Host
	}

	// Create output directory
	err = os.MkdirAll(options.OutputPath, 0755)
	if err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	// Initialize mirror state
	state := &MirrorState{
		baseURL:    baseURL,
		visited:    make(map[string]bool),
		pending:    []string{urlStr},
		downloaded: make(map[string]string),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}

	// Set up rate limiting
	if options.RateLimit != "" {
		state.limiter, err = parseRateLimit(options.RateLimit)
		if err != nil {
			logger.Printf("Warning: Invalid rate limit, proceeding without rate limiting: %v\n", err)
		}
	}

	// Start mirroring process
	err = state.mirror(options, 0)
	if err != nil {
		return err
	}

	// Convert links if requested
	if options.ConvertLinks {
		logger.Printf("Converting links for offline browsing...\n")
		err = state.convertAllLinks(options)
		if err != nil {
			logger.Printf("Warning: Link conversion failed: %v\n", err)
		}
	}

	logger.Printf("Website mirroring completed! Downloaded %d files to %s\n", state.fileCount, options.OutputPath)
	return nil
}

// mirror performs the recursive crawling and downloading
func (s *MirrorState) mirror(options *Options, depth int) error {
	if depth >= options.MaxDepth {
		s.logger.Printf("Reached maximum depth (%d), stopping recursion\n", options.MaxDepth)
		return nil
	}

	if s.fileCount >= options.MaxFiles {
		s.logger.Printf("Reached maximum file limit (%d), stopping download\n", options.MaxFiles)
		return nil
	}

	// Process all pending URLs at current depth
	currentLevel := make([]string, len(s.pending))
	copy(currentLevel, s.pending)
	s.pending = nil

	for _, urlStr := range currentLevel {
		if s.fileCount >= options.MaxFiles {
			break
		}

		// Skip if already visited
		s.mutex.Lock()
		if s.visited[urlStr] {
			s.mutex.Unlock()
			continue
		}
		s.visited[urlStr] = true
		s.mutex.Unlock()

		// Download and process the URL
		err := s.processURL(urlStr, options)
		if err != nil {
			s.logger.Printf("Warning: Failed to process %s: %v\n", urlStr, err)
			continue
		}
	}

	// Recurse to next depth level if there are pending URLs
	if len(s.pending) > 0 {
		return s.mirror(options, depth+1)
	}

	return nil
}

// processURL downloads a single URL and extracts resources from it
func (s *MirrorState) processURL(urlStr string, options *Options) error {
	// Rate limiting
	if s.limiter != nil {
		err := s.limiter.Wait(context.Background())
		if err != nil {
			return err
		}
	}

	// Download the content
	resp, err := s.client.Get(urlStr)
	if err != nil {
		return fmt.Errorf("failed to fetch %s: %v", urlStr, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d for %s", resp.StatusCode, urlStr)
	}

	// Read content
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read content from %s: %v", urlStr, err)
	}

	// Determine local file path
	localPath := GetLocalFilePath(urlStr, options.OutputPath)
	
	// Create directory structure
	err = os.MkdirAll(filepath.Dir(localPath), 0755)
	if err != nil {
		return fmt.Errorf("failed to create directory structure: %v", err)
	}

	// Save content to file
	err = os.WriteFile(localPath, content, 0644)
	if err != nil {
		return fmt.Errorf("failed to save file %s: %v", localPath, err)
	}

	// Record the download
	s.mutex.Lock()
	s.downloaded[urlStr] = localPath
	s.fileCount++
	s.mutex.Unlock()

	s.logger.Printf("Downloaded: %s -> %s\n", urlStr, localPath)

	// Parse content for additional resources (only for HTML and CSS)
	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "text/html") || strings.HasSuffix(urlStr, ".html") {
		err = s.extractHTMLResources(string(content), urlStr, options)
		if err != nil {
			s.logger.Printf("Warning: Failed to extract resources from %s: %v\n", urlStr, err)
		}
	} else if strings.Contains(contentType, "text/css") || strings.HasSuffix(urlStr, ".css") {
		err = s.extractCSSResources(string(content), urlStr, options)
		if err != nil {
			s.logger.Printf("Warning: Failed to extract CSS resources from %s: %v\n", urlStr, err)
		}
	}

	return nil
}

// extractHTMLResources extracts and queues resources from HTML content
func (s *MirrorState) extractHTMLResources(content, baseURLStr string, options *Options) error {
	baseURL, err := url.Parse(baseURLStr)
	if err != nil {
		return err
	}

	resources, err := ParseHTML(content, baseURL)
	if err != nil {
		return err
	}

	// Filter resources
	filtered := FilterResources(resources, options.RejectTypes, options.ExcludeDirs)

	// Add new resources to pending queue
	s.mutex.Lock()
	for _, resource := range filtered {
		// Only queue resources from the same domain
		resURL, err := url.Parse(resource.URL)
		if err != nil {
			continue
		}
		if resURL.Host != s.baseURL.Host {
			continue
		}

		// Skip if already visited or pending
		if !s.visited[resource.URL] {
			s.pending = append(s.pending, resource.URL)
		}
	}
	s.mutex.Unlock()

	return nil
}

// extractCSSResources extracts and queues resources from CSS content
func (s *MirrorState) extractCSSResources(content, baseURLStr string, options *Options) error {
	baseURL, err := url.Parse(baseURLStr)
	if err != nil {
		return err
	}

	resources, err := ParseCSS(content, baseURL)
	if err != nil {
		return err
	}

	// Filter resources
	filtered := FilterResources(resources, options.RejectTypes, options.ExcludeDirs)

	// Add new resources to pending queue
	s.mutex.Lock()
	for _, resource := range filtered {
		// Only queue resources from the same domain
		resURL, err := url.Parse(resource.URL)
		if err != nil {
			continue
		}
		if resURL.Host != s.baseURL.Host {
			continue
		}

		// Skip if already visited or pending
		if !s.visited[resource.URL] {
			s.pending = append(s.pending, resource.URL)
		}
	}
	s.mutex.Unlock()

	return nil
}

// convertAllLinks converts all links in downloaded files for offline browsing
func (s *MirrorState) convertAllLinks(options *Options) error {
	for _, localPath := range s.downloaded {
		// Read file content
		content, err := os.ReadFile(localPath)
		if err != nil {
			s.logger.Printf("Warning: Failed to read %s for link conversion: %v\n", localPath, err)
			continue
		}

		// Convert links based on file type
		var convertedContent string
		if strings.HasSuffix(localPath, ".html") || strings.HasSuffix(localPath, ".htm") {
			convertedContent = ConvertLinks(string(content), s.baseURL, options.OutputPath, localPath)
		} else if strings.HasSuffix(localPath, ".css") {
			convertedContent = ConvertCSSLinks(string(content), s.baseURL, options.OutputPath, localPath)
		} else {
			continue // Skip non-HTML/CSS files
		}

		// Write converted content back to file
		err = os.WriteFile(localPath, []byte(convertedContent), 0644)
		if err != nil {
			s.logger.Printf("Warning: Failed to write converted content to %s: %v\n", localPath, err)
		}
	}

	return nil
}

// parseRateLimit parses rate limit string and returns a rate limiter
func parseRateLimit(rateStr string) (*rate.Limiter, error) {
	// Use our simple rate limit parser directly
	return parseRateLimitSimple(rateStr)
}

// parseRateLimitSimple provides a simple rate limit parser
func parseRateLimitSimple(rateStr string) (*rate.Limiter, error) {
	rateStr = strings.ToLower(strings.TrimSpace(rateStr))
	
	var bytesPerSecond float64
	
	if strings.HasSuffix(rateStr, "k") {
		// Parse kilobytes per second
		var kb float64
		_, err := fmt.Sscanf(rateStr, "%fk", &kb)
		if err != nil {
			return nil, fmt.Errorf("invalid rate format: %s", rateStr)
		}
		bytesPerSecond = kb * 1024
	} else if strings.HasSuffix(rateStr, "m") {
		// Parse megabytes per second
		var mb float64
		_, err := fmt.Sscanf(rateStr, "%fm", &mb)
		if err != nil {
			return nil, fmt.Errorf("invalid rate format: %s", rateStr)
		}
		bytesPerSecond = mb * 1024 * 1024
	} else {
		// Parse bytes per second
		_, err := fmt.Sscanf(rateStr, "%f", &bytesPerSecond)
		if err != nil {
			return nil, fmt.Errorf("invalid rate format: %s", rateStr)
		}
	}
	
	if bytesPerSecond <= 0 {
		return nil, fmt.Errorf("rate must be positive: %s", rateStr)
	}
	
	// Create rate limiter (assuming average request size of 1KB for simplicity)
	requestsPerSecond := bytesPerSecond / 1024
	return rate.NewLimiter(rate.Limit(requestsPerSecond), 1), nil
}
