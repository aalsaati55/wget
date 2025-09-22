package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"wget/internal/batch"
	"wget/internal/bg"
	"wget/internal/downloader"
	"wget/internal/logging"
	"wget/internal/mirror"
)

type Config struct {
	URL         string
	OutputName  string
	OutputPath  string
	RateLimit   string
	Background  bool
	InputFile   string
	Mirror      bool
	Reject      string
	Exclude     string
	ConvertLinks bool
}

func main() {
	var config Config

	// Define flags
	flag.StringVar(&config.OutputName, "O", "", "Save file with different name")
	flag.StringVar(&config.OutputPath, "P", "", "Save file to specific directory")
	flag.StringVar(&config.RateLimit, "rate-limit", "", "Limit download rate (e.g., 400k, 2M)")
	flag.BoolVar(&config.Background, "B", false, "Download in background")
	flag.StringVar(&config.InputFile, "i", "", "Download URLs from file")
	flag.BoolVar(&config.Mirror, "mirror", false, "Mirror entire website")
	flag.StringVar(&config.Reject, "R", "", "Reject file types (comma-separated)")
	flag.StringVar(&config.Reject, "reject", "", "Reject file types (comma-separated)")
	flag.StringVar(&config.Exclude, "X", "", "Exclude directories (comma-separated)")
	flag.StringVar(&config.Exclude, "exclude", "", "Exclude directories (comma-separated)")
	flag.BoolVar(&config.ConvertLinks, "convert-links", false, "Convert links for offline viewing")

	flag.Parse()

	// Get URL from command line arguments
	args := flag.Args()
	if len(args) == 0 && config.InputFile == "" {
		fmt.Fprintf(os.Stderr, "Error: URL required\n")
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS] URL\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}

	if len(args) > 0 {
		config.URL = args[0]
	}

	// Validate flag combinations
	if err := validateConfig(&config); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Initialize logging
	logger := logging.NewLogger(config.Background)

	// Execute based on configuration
	if err := executeDownload(&config, logger); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func validateConfig(config *Config) error {
	// Mirror-specific validations
	if (config.Reject != "" || config.Exclude != "" || config.ConvertLinks) && !config.Mirror {
		return fmt.Errorf("--reject, --exclude, and --convert-links can only be used with --mirror")
	}

	// Input file vs URL validation
	if config.InputFile != "" && config.URL != "" {
		return fmt.Errorf("cannot specify both input file (-i) and URL")
	}

	return nil
}

func executeDownload(config *Config, logger *logging.Logger) error {
	// Background download
	if config.Background {
		return bg.DownloadInBackground(config.URL, &bg.Options{
			OutputName: config.OutputName,
			OutputPath: config.OutputPath,
			RateLimit:  config.RateLimit,
		})
	}

	// Batch download from file
	if config.InputFile != "" {
		return batch.DownloadFromFile(config.InputFile, &batch.Options{
			OutputPath: config.OutputPath,
			RateLimit:  config.RateLimit,
		}, logger)
	}

	// Website mirroring
	if config.Mirror {
		rejectTypes := parseCommaSeparated(config.Reject)
		excludeDirs := parseCommaSeparated(config.Exclude)
		
		return mirror.MirrorWebsite(config.URL, &mirror.Options{
			RejectTypes:  rejectTypes,
			ExcludeDirs:  excludeDirs,
			ConvertLinks: config.ConvertLinks,
			OutputPath:   config.OutputPath,
			RateLimit:    config.RateLimit,
		}, logger)
	}

	// Single file download
	return downloader.DownloadFile(config.URL, &downloader.Options{
		OutputName: config.OutputName,
		OutputPath: config.OutputPath,
		RateLimit:  config.RateLimit,
	}, logger)
}

func parseCommaSeparated(input string) []string {
	if input == "" {
		return nil
	}
	parts := strings.Split(input, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}