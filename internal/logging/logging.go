package logging

import (
	"fmt"
	"io"
	"os"
	"time"
)

const (
	TimeFormat = "2006-01-02 15:04:05"
	LogFile    = "wget-log"
)

type Logger struct {
	output     io.Writer
	background bool
}

// NewLogger creates a new logger instance
func NewLogger(background bool) *Logger {
	logger := &Logger{
		background: background,
		output:     os.Stdout,
	}

	if background {
		// Create or open log file for background downloads
		file, err := os.OpenFile(LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating log file: %v\n", err)
			os.Exit(1)
		}
		logger.output = file

		// Print message to stdout about log file
		fmt.Printf("Output will be written to \"%s\".\n", LogFile)
	}

	return logger
}

// Printf writes formatted output to the logger
func (l *Logger) Printf(format string, args ...interface{}) {
	fmt.Fprintf(l.output, format, args...)
}

// Println writes a line to the logger
func (l *Logger) Println(args ...interface{}) {
	fmt.Fprintln(l.output, args...)
}

// LogStart logs the start time of a download
func (l *Logger) LogStart() {
	l.Printf("start at %s\n", time.Now().Format(TimeFormat))
}

// LogFinish logs the finish time of a download
func (l *Logger) LogFinish() {
	l.Printf("finished at %s\n", time.Now().Format(TimeFormat))
}

// LogStatus logs the HTTP response status
func (l *Logger) LogStatus(status string) {
	l.Printf("sending request, awaiting response... status %s\n", status)
}

// LogContentSize logs the content size information
func (l *Logger) LogContentSize(size int64) {
	l.Printf("content size: %d [~%.2fMB]\n", size, float64(size)/1024/1024)
}

// LogSavingTo logs where the file is being saved
func (l *Logger) LogSavingTo(filepath string) {
	l.Printf("saving file to: %s\n", filepath)
}

// LogDownloaded logs successful download completion
func (l *Logger) LogDownloaded(url string) {
	l.Printf("Downloaded [%s]\n", url)
}

// LogError logs an error message
func (l *Logger) LogError(err error) {
	l.Printf("Error: %v\n", err)
}

// LogProgress logs download progress (for progress bar updates)
func (l *Logger) LogProgress(downloaded, total int64, speed float64, eta time.Duration) {
	if l.background {
		// Don't show progress bar in background mode
		return
	}

	downloadedStr := FormatBytes(downloaded)
	totalStr := FormatBytes(total)
	speedStr := FormatSpeed(speed)

	percentage := float64(downloaded) / float64(total) * 100

	// Create progress bar
	barWidth := 80
	filled := int(percentage / 100 * float64(barWidth))
	bar := ""
	for i := 0; i < barWidth; i++ {
		if i < filled {
			bar += "="
		} else {
			bar += " "
		}
	}

	etaStr := FormatDuration(eta)

	// Print progress line (overwrite previous line)
	fmt.Printf("\r %s / %s [%s] %.2f%% %s %s",
		downloadedStr, totalStr, bar, percentage, speedStr, etaStr)
}

// FormatBytes formats bytes into human-readable format
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// FormatSpeed formats speed into human-readable format
func FormatSpeed(bytesPerSecond float64) string {
	const unit = 1024
	if bytesPerSecond < unit {
		return fmt.Sprintf("%.2f B/s", bytesPerSecond)
	}

	units := []string{"KiB/s", "MiB/s", "GiB/s"}
	div := float64(unit)
	for i, u := range units {
		if bytesPerSecond < div*unit || i == len(units)-1 {
			return fmt.Sprintf("%.2f %s", bytesPerSecond/div, u)
		}
		div *= unit
	}
	return fmt.Sprintf("%.2f B/s", bytesPerSecond)
}

// FormatDuration formats duration for ETA display
func FormatDuration(d time.Duration) string {
	if d < 0 {
		return "0s"
	}

	seconds := int(d.Seconds())
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}

	minutes := seconds / 60
	seconds = seconds % 60
	if minutes < 60 {
		return fmt.Sprintf("%dm%ds", minutes, seconds)
	}

	hours := minutes / 60
	minutes = minutes % 60
	return fmt.Sprintf("%dh%dm%ds", hours, minutes, seconds)
}

// Close closes the logger (important for file-based loggers)
func (l *Logger) Close() error {
	if file, ok := l.output.(*os.File); ok && file != os.Stdout && file != os.Stderr {
		return file.Close()
	}
	return nil
}
