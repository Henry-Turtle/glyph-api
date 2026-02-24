package main

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"

	"sidecar/services"
)

// Get expected API key from environment, fallback to default if not set
func getExpectedAPIKey() string {
	key := os.Getenv("API_KEY")
	if key == "" {
		return "super-secret-key"
	}
	return key
}

// AuthMiddleware ensures that requests contain a valid X-API-KEY header.
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-KEY")
		if apiKey != getExpectedAPIKey() {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: Invalid or missing X-API-KEY"})
			return
		}
		c.Next()
	}
}

// UpdateTrackRequest represents the expected JSON body for the /update-track endpoint.
type UpdateTrackRequest struct {
	FilePath string `json:"file_path" binding:"required"`
	Title    string `json:"title" binding:"required"`
	Artist   string `json:"artist" binding:"required"`
}

// UpdateTrackHandler processes the track update request.
func UpdateTrackHandler(c *gin.Context) {
	var req UpdateTrackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	// Verify the file exists before passing to the engine room
	if _, err := os.Stat(req.FilePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found: " + req.FilePath})
		return
	}

	// Invoke the service to perform the metadata update
	if err := services.UpdateTrackMetadata(req.FilePath, req.Title, req.Artist); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update metadata: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Metadata updated successfully",
		"track": gin.H{
			"file_path": req.FilePath,
			"title":     req.Title,
			"artist":    req.Artist,
		},
	})
}

// DownloadTrackRequest represents the expected JSON body for the /download-track endpoint
type DownloadTrackRequest struct {
	Url     string `json:"url" binding:"required"`
	Quality string `json:"quality" binding:"required,oneof=low mid max"`
	Title   string `json:"title"`
	Artist  string `json:"artist"`
	Album   string `json:"album"`
}

// DownloadTrackHandler kicks off an asynchronous yt-dlp download job
func DownloadTrackHandler(c *gin.Context) {
	var req DownloadTrackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	// Queue download task in the background engine
	services.BackgroundEngine.EnqueueDownload(&services.DownloadTask{
		Url:     req.Url,
		Quality: req.Quality,
		Title:   req.Title,
		Artist:  req.Artist,
		Album:   req.Album,
	})

	c.JSON(http.StatusAccepted, gin.H{
		"message": "Download task queued in the background",
		"task": gin.H{
			"url":     req.Url,
			"quality": req.Quality,
			"status":  "queued",
		},
	})
}

// HealthCheckHandler verifies API connectivity and write access to the /music directory.
func HealthCheckHandler(c *gin.Context) {
	// Attempt to create a temporary file to verify write access
	f, err := os.CreateTemp("/music", ".healthcheck-*")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status": "error",
			"error": "Failed to verify write access to /music directory: " + err.Error(),
			"authenticated": true, // Middleware already verified this
		})
		return
	}
	
	f.Close()
	os.Remove(f.Name())

	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"message": "Connected to Glyph API. Read/Write access to /music verified.",
		"authenticated": true,
	})
}
