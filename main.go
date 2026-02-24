package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"sidecar/services"

	"github.com/gin-gonic/gin"
)

func main() {
	// 1. Initialize the Background Engine context
	engineCtx, engineCancel := context.WithCancel(context.Background())
	services.StartEngine(engineCtx)

	// 2. Setup Gin Router
	r := gin.Default()
	r.Use(AuthMiddleware())
	r.POST("/update-track", UpdateTrackHandler)
	r.POST("/download-track", DownloadTrackHandler)

	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	// 3. Start API Server asynchronously
	go func() {
		log.Println("Starting Glyph API on :8080...")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	// 4. Listen for OS Interrupts (SIGINT/SIGTERM)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutdown signal received...")

	// 5. Gracefully shutdown API Server (stop accepting new requests)
	apiCtx, apiCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer apiCancel()
	if err := srv.Shutdown(apiCtx); err != nil {
		log.Fatal("API Server forced to shutdown:", err)
	}

	log.Println("API Server stopped. Waiting for active background downloads to finish...")

	// 6. Cancel Background Engine Context (stops yt-dlp processes quickly)
	engineCancel()

	// 7. Wait for all background engine workers to finish their cleanup
	services.BackgroundEngine.Wait()

	log.Println("Glyph API safely exited.")
}
