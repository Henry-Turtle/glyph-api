package main

import (
	"log"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	// Apply API Key Middleware globally or to specific routes
	r.Use(AuthMiddleware())

	// Define our single POST endpoint
	r.POST("/update-track", UpdateTrackHandler)
	r.POST("/download-track", DownloadTrackHandler)

	log.Println("Starting Sidecar API on :8080...")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}
