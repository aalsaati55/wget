package batch

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"wget/internal/downloader"
	"wget/internal/logging"
)

type Options struct {
	OutputPath string
	RateLimit  string
}

type DownloadResult struct {
	URL   string
	Error error
}

// DownloadFromFile downloads multiple files from URLs listed in a file
func DownloadFromFile(filename string, options *Options, logger *logging.Logger) error {
	// Read URLs from file
	urls, err := readURLsFromFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read URLs from file: %v", err)
	}

	if len(urls) == 0 {
		return fmt.Errorf("no URLs found in file: %s", filename)
	}

	// Calculate total content sizes (if possible)
	contentSizes := make([]int64, len(urls))
	totalSize := int64(0)

	logger.Printf("Checking content sizes...\n")
	for i, url := range urls {
		size, err := getContentSize(url)
		if err == nil && size > 0 {
			contentSizes[i] = size
			totalSize += size
		}
	}

	if totalSize > 0 {
		logger.Printf("content size: %v\n", contentSizes)
	}

	// Create channels for coordination
	results := make(chan DownloadResult, len(urls))
	var wg sync.WaitGroup

	// Start downloads concurrently
	for i, url := range urls {
		wg.Add(1)
		go func(url string, index int) {
			defer wg.Done()

			// Create individual logger for this download (no progress bar in batch mode)
			downloadLogger := logging.NewLogger(false)

			// Create downloader options
			downloaderOptions := &downloader.Options{
				OutputPath: options.OutputPath,
				RateLimit:  options.RateLimit,
			}

			// Download the file
			err := downloader.DownloadFile(url, downloaderOptions, downloadLogger)

			// Send result
			results <- DownloadResult{
				URL:   url,
				Error: err,
			}

			// Log completion
			if err == nil {
				filename := getFilenameFromURL(url)
				logger.Printf("finished %s\n", filename)
			}
		}(url, i)
	}

	// Wait for all downloads to complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	var successfulDownloads []string
	var errors []error

	for result := range results {
		if result.Error != nil {
			errors = append(errors, fmt.Errorf("failed to download %s: %v", result.URL, result.Error))
		} else {
			successfulDownloads = append(successfulDownloads, result.URL)
		}
	}

	// Log final results
	if len(successfulDownloads) > 0 {
		logger.Printf("\nDownload finished: %v\n", successfulDownloads)
	}

	// Return first error if any occurred
	if len(errors) > 0 {
		return errors[0]
	}

	return nil
}

// readURLsFromFile reads URLs from a text file, one URL per line
func readURLsFromFile(filename string) ([]string, error) {
	// Read the entire file content first
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	// Convert to string and handle different encodings
	text := string(content)
	
	// Remove BOM if present at the beginning of file
	if strings.HasPrefix(text, "\uFEFF") {
		text = strings.TrimPrefix(text, "\uFEFF")
	}
	
	// Handle UTF-16 BOM by removing the problematic bytes
	if len(content) >= 2 && content[0] == 0xFF && content[1] == 0xFE {
		// UTF-16 LE BOM detected, try to clean it
		text = strings.ReplaceAll(text, "\x00", "") // Remove null bytes from UTF-16
		text = strings.TrimPrefix(text, "\xFF\xFE") // Remove BOM
	}
	
	// Split into lines and process each
	lines := strings.Split(text, "\n")
	var urls []string
	
	for _, line := range lines {
		// Clean the line thoroughly
		line = strings.TrimSpace(line)
		line = strings.ReplaceAll(line, "\r", "")
		line = strings.ReplaceAll(line, "\x00", "") // Remove any remaining null characters
		
		// Remove any non-printable characters at the beginning
		for len(line) > 0 && (line[0] < 32 || line[0] > 126) && line[0] != '\t' {
			line = line[1:]
		}
		
		if line != "" && !strings.HasPrefix(line, "#") {
			urls = append(urls, line)
		}
	}

	return urls, nil
}

// getContentSize makes a HEAD request to get the content size without downloading
func getContentSize(url string) (int64, error) {
	resp, err := http.Head(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("server returned status: %s", resp.Status)
	}

	return resp.ContentLength, nil
}

// getFilenameFromURL extracts filename from URL for logging purposes
func getFilenameFromURL(urlStr string) string {
	parts := strings.Split(urlStr, "/")
	if len(parts) > 0 {
		filename := parts[len(parts)-1]
		if filename != "" {
			return filename
		}
	}
	return urlStr
}
