package bg

import (
	"wget/internal/downloader"
	"wget/internal/logging"
)

type Options struct {
	OutputName string
	OutputPath string
	RateLimit  string
}

// DownloadInBackground downloads a file in the background with output redirected to log file
func DownloadInBackground(url string, options *Options) error {
	// Create logger for background mode (will redirect to wget-log)
	logger := logging.NewLogger(true)
	defer logger.Close()

	// Convert bg.Options to downloader.Options
	downloaderOptions := &downloader.Options{
		OutputName: options.OutputName,
		OutputPath: options.OutputPath,
		RateLimit:  options.RateLimit,
	}

	// Perform the download
	return downloader.DownloadFile(url, downloaderOptions, logger)
}
