package downloader

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"wget/internal/logging"

	"golang.org/x/time/rate"
)

type Options struct {
	OutputName string
	OutputPath string
	RateLimit  string
}

type ProgressReader struct {
	reader     io.Reader
	total      int64
	downloaded int64
	lastUpdate time.Time
	startTime  time.Time
	logger     *logging.Logger
	limiter    *rate.Limiter
}

// DownloadFile downloads a single file from the given URL
func DownloadFile(urlStr string, options *Options, logger *logging.Logger) error {
	logger.LogStart()

	// Parse and validate URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL: %v", err)
	}

	// Create HTTP client
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Make HTTP request
	resp, err := client.Get(urlStr)
	if err != nil {
		return fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	// Log response status
	logger.LogStatus(resp.Status)

	// Check if response is successful
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status: %s", resp.Status)
	}

	// Get content length
	contentLength := resp.ContentLength
	if contentLength > 0 {
		logger.LogContentSize(contentLength)
	}

	// Determine output file path
	outputPath, err := determineOutputPath(urlStr, parsedURL, options)
	if err != nil {
		return fmt.Errorf("failed to determine output path: %v", err)
	}

	logger.LogSavingTo(outputPath)

	// Create output directory if needed
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	// Create output file
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer file.Close()

	// Set up rate limiter if specified
	var limiter *rate.Limiter
	if options.RateLimit != "" {
		limiter, err = parseRateLimit(options.RateLimit)
		if err != nil {
			return fmt.Errorf("invalid rate limit: %v", err)
		}
	}

	// Create progress reader
	progressReader := &ProgressReader{
		reader:     resp.Body,
		total:      contentLength,
		downloaded: 0,
		lastUpdate: time.Now(),
		startTime:  time.Now(),
		logger:     logger,
		limiter:    limiter,
	}

	// Copy data with progress tracking
	_, err = io.Copy(file, progressReader)
	if err != nil {
		return fmt.Errorf("failed to download file: %v", err)
	}

	// Final newline after progress bar
	if contentLength > 0 {
		fmt.Println()
	}

	logger.LogDownloaded(urlStr)
	logger.LogFinish()

	return nil
}

// Read implements io.Reader interface with progress tracking and rate limiting
func (pr *ProgressReader) Read(p []byte) (int, error) {
	// Apply rate limiting if configured
	if pr.limiter != nil {
		// Wait for rate limiter permission
		err := pr.limiter.WaitN(nil, len(p))
		if err != nil {
			return 0, err
		}
	}

	n, err := pr.reader.Read(p)
	if n > 0 {
		pr.downloaded += int64(n)

		// Update progress every 100ms to avoid too frequent updates
		now := time.Now()
		if now.Sub(pr.lastUpdate) >= 100*time.Millisecond || err == io.EOF {
			pr.updateProgress()
			pr.lastUpdate = now
		}
	}
	return n, err
}

func (pr *ProgressReader) updateProgress() {
	if pr.total <= 0 {
		return // Can't show progress without content length
	}

	elapsed := time.Since(pr.startTime)
	if elapsed.Seconds() == 0 {
		return
	}

	// Calculate speed (bytes per second)
	speed := float64(pr.downloaded) / elapsed.Seconds()

	// Calculate ETA
	var eta time.Duration
	if speed > 0 {
		remaining := pr.total - pr.downloaded
		eta = time.Duration(float64(remaining)/speed) * time.Second
	}

	pr.logger.LogProgress(pr.downloaded, pr.total, speed, eta)
}

// determineOutputPath determines where to save the downloaded file
func determineOutputPath(urlStr string, parsedURL *url.URL, options *Options) (string, error) {
	var filename string

	// Use custom output name if provided
	if options.OutputName != "" {
		filename = options.OutputName
	} else {
		// Extract filename from URL
		filename = path.Base(parsedURL.Path)
		if filename == "/" || filename == "." {
			// If no filename in URL, use domain name
			filename = parsedURL.Host
		}
	}

	// Use custom output path if provided
	if options.OutputPath != "" {
		// Expand ~ to home directory
		outputPath := options.OutputPath
		if strings.HasPrefix(outputPath, "~/") {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			outputPath = filepath.Join(homeDir, outputPath[2:])
		}
		return filepath.Join(outputPath, filename), nil
	}

	// Default to current directory
	return filepath.Join(".", filename), nil
}

// parseRateLimit parses rate limit string (e.g., "400k", "2M") into rate.Limiter
func parseRateLimit(rateStr string) (*rate.Limiter, error) {
	rateStr = strings.TrimSpace(strings.ToLower(rateStr))
	if rateStr == "" {
		return nil, fmt.Errorf("empty rate limit")
	}

	// Extract number and unit
	var numStr string
	var unit string

	for i, r := range rateStr {
		if r >= '0' && r <= '9' || r == '.' {
			numStr += string(r)
		} else {
			unit = rateStr[i:]
			break
		}
	}

	if numStr == "" {
		return nil, fmt.Errorf("no number found in rate limit")
	}

	// Parse the number
	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid number in rate limit: %v", err)
	}

	// Convert to bytes per second based on unit
	var bytesPerSecond float64
	switch unit {
	case "", "b":
		bytesPerSecond = num
	case "k", "kb":
		bytesPerSecond = num * 1024
	case "m", "mb":
		bytesPerSecond = num * 1024 * 1024
	case "g", "gb":
		bytesPerSecond = num * 1024 * 1024 * 1024
	default:
		return nil, fmt.Errorf("unknown unit in rate limit: %s", unit)
	}

	if bytesPerSecond <= 0 {
		return nil, fmt.Errorf("rate limit must be positive")
	}

	// Create rate limiter
	// Use burst size of 1KB to allow for smooth downloads
	burstSize := int(1024)
	if bytesPerSecond < 1024 {
		burstSize = int(bytesPerSecond)
	}

	return rate.NewLimiter(rate.Limit(bytesPerSecond), burstSize), nil
}
