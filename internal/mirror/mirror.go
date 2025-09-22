package mirror

import (
	"fmt"
	"wget/internal/logging"
)

type Options struct {
	RejectTypes  []string
	ExcludeDirs  []string
	ConvertLinks bool
	OutputPath   string
	RateLimit    string
}

// MirrorWebsite downloads an entire website (stub implementation)
func MirrorWebsite(url string, options *Options, logger *logging.Logger) error {
	// TODO: Implement website mirroring in Phase 5
	return fmt.Errorf("website mirroring not yet implemented")
}
