package handler

import (
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// formatPosterURL formats the poster URL based on the current request
// If the poster path is already a full URL, it's returned as-is
// If it's a local path, it's converted to a full URL using the request's host and scheme
func formatPosterURL(c *gin.Context, posterPath string) string {
	if posterPath == "" {
		return ""
	}

	// If it's already a full URL, return as-is
	if strings.HasPrefix(posterPath, "http://") || strings.HasPrefix(posterPath, "https://") {
		return posterPath
	}

	// For local paths, construct the full URL
	scheme := "http://"
	if c.Request.TLS != nil || c.Request.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https://"
	}

	// Handle both Windows and Unix paths
	filename := filepath.Base(posterPath)
	if strings.HasPrefix(posterPath, "public/posters/") {
		filename = strings.TrimPrefix(posterPath, "public/posters/")
	}

	return scheme + c.Request.Host + "/posters/" + filename
}
