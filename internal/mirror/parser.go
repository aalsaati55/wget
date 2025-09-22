package mirror

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// ResourceType represents different types of web resources
type ResourceType int

const (
	HTML ResourceType = iota
	CSS
	JS
	Image
	Other
)

// Resource represents a web resource found during parsing
type Resource struct {
	URL      string
	Type     ResourceType
	Original string // Original text in the document
}

// ParseHTML extracts all resources (links, images, CSS, JS) from HTML content
func ParseHTML(content string, baseURL *url.URL) ([]Resource, error) {
	var resources []Resource

	// Extract links (href attributes)
	hrefRegex := regexp.MustCompile(`(?i)href\s*=\s*["']([^"']+)["']`)
	hrefMatches := hrefRegex.FindAllStringSubmatch(content, -1)
	for _, match := range hrefMatches {
		if len(match) > 1 {
			absURL, err := resolveURL(match[1], baseURL)
			if err == nil {
				resType := determineResourceType(absURL)
				resources = append(resources, Resource{
					URL:      absURL,
					Type:     resType,
					Original: match[0],
				})
			}
		}
	}

	// Extract images (src attributes)
	srcRegex := regexp.MustCompile(`(?i)src\s*=\s*["']([^"']+)["']`)
	srcMatches := srcRegex.FindAllStringSubmatch(content, -1)
	for _, match := range srcMatches {
		if len(match) > 1 {
			absURL, err := resolveURL(match[1], baseURL)
			if err == nil {
				resources = append(resources, Resource{
					URL:      absURL,
					Type:     Image,
					Original: match[0],
				})
			}
		}
	}

	// Extract CSS imports and links
	cssLinkRegex := regexp.MustCompile(`(?i)<link[^>]*rel\s*=\s*["']stylesheet["'][^>]*href\s*=\s*["']([^"']+)["']`)
	cssMatches := cssLinkRegex.FindAllStringSubmatch(content, -1)
	for _, match := range cssMatches {
		if len(match) > 1 {
			absURL, err := resolveURL(match[1], baseURL)
			if err == nil {
				resources = append(resources, Resource{
					URL:      absURL,
					Type:     CSS,
					Original: match[0],
				})
			}
		}
	}

	// Extract JavaScript files
	jsRegex := regexp.MustCompile(`(?i)<script[^>]*src\s*=\s*["']([^"']+)["']`)
	jsMatches := jsRegex.FindAllStringSubmatch(content, -1)
	for _, match := range jsMatches {
		if len(match) > 1 {
			absURL, err := resolveURL(match[1], baseURL)
			if err == nil {
				resources = append(resources, Resource{
					URL:      absURL,
					Type:     JS,
					Original: match[0],
				})
			}
		}
	}

	return resources, nil
}

// ParseCSS extracts URLs from CSS content (imports, background images, etc.)
func ParseCSS(content string, baseURL *url.URL) ([]Resource, error) {
	var resources []Resource

	// Extract @import statements
	importRegex := regexp.MustCompile(`(?i)@import\s+["']([^"']+)["']`)
	importMatches := importRegex.FindAllStringSubmatch(content, -1)
	for _, match := range importMatches {
		if len(match) > 1 {
			absURL, err := resolveURL(match[1], baseURL)
			if err == nil {
				resources = append(resources, Resource{
					URL:      absURL,
					Type:     CSS,
					Original: match[0],
				})
			}
		}
	}

	// Extract url() references (background images, fonts, etc.)
	urlRegex := regexp.MustCompile(`(?i)url\s*\(\s*["']?([^"')]+)["']?\s*\)`)
	urlMatches := urlRegex.FindAllStringSubmatch(content, -1)
	for _, match := range urlMatches {
		if len(match) > 1 {
			absURL, err := resolveURL(match[1], baseURL)
			if err == nil {
				resType := determineResourceType(absURL)
				resources = append(resources, Resource{
					URL:      absURL,
					Type:     resType,
					Original: match[0],
				})
			}
		}
	}

	return resources, nil
}

// resolveURL converts a relative URL to an absolute URL
func resolveURL(href string, baseURL *url.URL) (string, error) {
	// Skip data URLs, javascript:, mailto:, etc.
	if strings.HasPrefix(href, "data:") || strings.HasPrefix(href, "javascript:") ||
		strings.HasPrefix(href, "mailto:") || strings.HasPrefix(href, "tel:") {
		return "", fmt.Errorf("skipping non-http URL: %s", href)
	}

	// Parse the href
	parsedHref, err := url.Parse(href)
	if err != nil {
		return "", err
	}

	// Resolve relative to base URL
	resolvedURL := baseURL.ResolveReference(parsedHref)
	return resolvedURL.String(), nil
}

// determineResourceType determines the type of resource based on URL
func determineResourceType(urlStr string) ResourceType {
	lower := strings.ToLower(urlStr)

	// Check file extension
	if strings.Contains(lower, ".css") {
		return CSS
	}
	if strings.Contains(lower, ".js") {
		return JS
	}
	if strings.Contains(lower, ".png") || strings.Contains(lower, ".jpg") ||
		strings.Contains(lower, ".jpeg") || strings.Contains(lower, ".gif") ||
		strings.Contains(lower, ".svg") || strings.Contains(lower, ".webp") {
		return Image
	}
	if strings.Contains(lower, ".html") || strings.Contains(lower, ".htm") ||
		!strings.Contains(lower, ".") { // Assume URLs without extensions are HTML
		return HTML
	}

	return Other
}

// FilterResources filters resources based on reject and exclude patterns
func FilterResources(resources []Resource, rejectTypes []string, excludeDirs []string) []Resource {
	var filtered []Resource

	for _, resource := range resources {
		// Check reject patterns (file types)
		rejected := false
		for _, reject := range rejectTypes {
			if strings.Contains(strings.ToLower(resource.URL), strings.ToLower(reject)) {
				rejected = true
				break
			}
		}
		if rejected {
			continue
		}

		// Check exclude patterns (directories)
		excluded := false
		for _, exclude := range excludeDirs {
			if strings.Contains(resource.URL, exclude) {
				excluded = true
				break
			}
		}
		if excluded {
			continue
		}

		filtered = append(filtered, resource)
	}

	return filtered
}
