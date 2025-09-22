package mirror

import (
	"net/url"
	"path/filepath"
	"strings"
)

// ConvertLinks converts absolute URLs in content to relative paths for offline browsing
func ConvertLinks(content string, baseURL *url.URL, outputDir string, currentFilePath string) string {
	resources, err := ParseHTML(content, baseURL)
	if err != nil {
		return content
	}

	convertedContent := content

	// Convert each resource URL to a relative path
	for _, resource := range resources {
		originalURL := resource.URL
		relativePath := convertURLToRelativePath(originalURL, baseURL, outputDir, currentFilePath)
		
		if relativePath != "" {
			// Replace the original URL with the relative path
			convertedContent = strings.ReplaceAll(convertedContent, originalURL, relativePath)
		}
	}

	return convertedContent
}

// ConvertCSSLinks converts URLs in CSS content to relative paths
func ConvertCSSLinks(content string, baseURL *url.URL, outputDir string, currentFilePath string) string {
	resources, err := ParseCSS(content, baseURL)
	if err != nil {
		return content
	}

	convertedContent := content

	// Convert each resource URL to a relative path
	for _, resource := range resources {
		originalURL := resource.URL
		relativePath := convertURLToRelativePath(originalURL, baseURL, outputDir, currentFilePath)
		
		if relativePath != "" {
			// Replace the original URL with the relative path in CSS url() syntax
			convertedContent = strings.ReplaceAll(convertedContent, originalURL, relativePath)
		}
	}

	return convertedContent
}

// convertURLToRelativePath converts an absolute URL to a relative file path
func convertURLToRelativePath(urlStr string, baseURL *url.URL, outputDir string, currentFilePath string) string {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return ""
	}

	// Only convert URLs from the same domain
	if parsedURL.Host != baseURL.Host {
		return ""
	}

	// Convert URL path to local file path
	localPath := convertURLPathToLocalPath(parsedURL.Path, outputDir)
	
	// Calculate relative path from current file to target file
	currentDir := filepath.Dir(currentFilePath)
	relativePath, err := filepath.Rel(currentDir, localPath)
	if err != nil {
		return ""
	}

	// Convert backslashes to forward slashes for web compatibility
	return strings.ReplaceAll(relativePath, "\\", "/")
}

// convertURLPathToLocalPath converts a URL path to a local file system path
func convertURLPathToLocalPath(urlPath string, outputDir string) string {
	// Remove leading slash
	if strings.HasPrefix(urlPath, "/") {
		urlPath = urlPath[1:]
	}

	// If path is empty or ends with /, assume index.html
	if urlPath == "" || strings.HasSuffix(urlPath, "/") {
		urlPath = filepath.Join(urlPath, "index.html")
	}

	// Convert URL path separators to OS-specific path separators
	localPath := filepath.Join(outputDir, filepath.FromSlash(urlPath))
	
	return localPath
}

// GetLocalFilePath determines the local file path for a given URL
func GetLocalFilePath(urlStr string, outputDir string) string {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return ""
	}

	return convertURLPathToLocalPath(parsedURL.Path, outputDir)
}
