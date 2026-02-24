package services

import (
	"context"
	"log"
	"os"
	"strconv"
	"sync"
)

// Task represents a background job that can be executed and cancelled.
type Task interface {
	Execute(ctx context.Context)
}

// Engine manages the background worker pools.
type Engine struct {
	downloadChan chan Task
	wg           sync.WaitGroup
}

// Global engine instance
var BackgroundEngine *Engine

// StartEngine initializes the channels, reads environment variables, and spins up workers.
func StartEngine(ctx context.Context) {
	BackgroundEngine = &Engine{
		// Buffered channel to queue downloads
		downloadChan: make(chan Task, 1000), 
	}

	// Determine pool size for yt-dlp
	workers := 3 // Default
	if envVal := os.Getenv("MAX_DOWNLOAD_WORKERS"); envVal != "" {
		if parsed, err := strconv.Atoi(envVal); err == nil && parsed > 0 {
			workers = parsed
		} else {
			log.Printf("[WARN] Invalid MAX_DOWNLOAD_WORKERS value '%s', defaulting to 3", envVal)
		}
	}

	// Enforce Sanity Cap
	if workers > 10 {
		log.Printf("[WARN] MAX_DOWNLOAD_WORKERS set too high (%d) for average I/O; capping at 10.", workers)
		workers = 10
	}

	log.Printf("Starting Background Engine with %d yt-dlp workers...", workers)

	// Spin up Bulk Lane workers
	for i := 1; i <= workers; i++ {
		BackgroundEngine.wg.Add(1)
		go BackgroundEngine.workerLane(ctx, i)
	}
}

// workerLane pulls jobs off the channel until the channel closes or context cancels.
func (e *Engine) workerLane(ctx context.Context, id int) {
	defer e.wg.Done()
	
	for {
		select {
		case <-ctx.Done():
			log.Printf("[LANE:BULK-%d] Shutting down cleanly...", id)
			return
		case task, ok := <-e.downloadChan:
			if !ok {
				return // Channel closed
			}
			task.Execute(ctx)
		}
	}
}

// EnqueueDownload adds a task to the bulk queue.
func (e *Engine) EnqueueDownload(t Task) {
	e.downloadChan <- t
}

// Wait blocks until all active workers finish their current task.
// Ensure main context is cancelled *before* calling this.
func (e *Engine) Wait() {
	e.wg.Wait()
	log.Println("All background tasks finished successfully.")
}
