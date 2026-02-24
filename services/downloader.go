package services

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// DownloadTrackAsync runs yt-dlp to fetch the audio, saves it to the /music folder,
// and applies the optional metadata using ID3 tags.
func DownloadTrackAsync(url, quality, title, artist, album string) {
	log.Printf("Starting async download for: %s", url)

	// Determine output directory structure
	// We'll create a folder structure like: /music/ArtistName/AlbumName/
	// If artist or album are not provided, we fall back to "Unknown" folders or similar.

	safeArtist := sanitizePath(artist)
	if safeArtist == "" {
		safeArtist = "Unknown Artist"
	}

	safeAlbum := sanitizePath(album)
	if safeAlbum == "" {
		safeAlbum = "Unknown Album"
	}

	targetDir := filepath.Join("/music", safeArtist, safeAlbum)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		log.Printf("ERROR: Failed to create target directory %s: %v", targetDir, err)
		return
	}

	// Map requested quality to a yt-dlp audio quality parameter
	// 0 is best, 5 is default (~192k), 9 is worst
	ytQuality := "5" // Default to mid
	switch quality {
	case "max":
		ytQuality = "0"
	case "low":
		ytQuality = "9"
	}

	// We don't know the exact filename until yt-dlp finishes downloading.
	// yt-dlp's `-o` template will define it.
	// We'll use the YouTube video title or ID as the file name, enforcing .mp3 extension.
	outputTemplate := filepath.Join(targetDir, "%(title)s (%(id)s).%(ext)s")

	log.Printf("Executing yt-dlp into: %s (quality: %s)", targetDir, quality)

	// Construct the yt-dlp command
	cmd := exec.Command("yt-dlp",
		"-x", // Extract audio
		"--audio-format", "mp3",
		"--audio-quality", ytQuality,
		"-o", outputTemplate,
		url,
	)

	// Capture output for debugging
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("ERROR: yt-dlp failed: %v\nOutput: %s", err, string(output))
		return
	}

	// The command succeeded.
	// Let's attempt to locate the downloaded .mp3 file by looking at the directory contents
	// (Since we don't know the exact title yt-dlp used).
	// For a more robust solution we could parse `output` or use --print "after_move:filepath",
	// but a fast approach is to extract the filename from the log output or search the dir locally.

	// A robust way to parse the final filename from yt-dlp:
	// "Destination: /music/Artist/Album/VideoTitle (ID).mp3"
	downloadedFile := extractFilenameFromYTDLPOutput(string(output))
	if downloadedFile == "" {
		log.Printf("ERROR: yt-dlp succeeded but could not determine final output filepath from logs.")
		return
	}

	log.Printf("Successfully downloaded file: %s", downloadedFile)

	// Finally, apply the requested metadata using our existing engine room logic
	if title != "" || artist != "" || album != "" {
		log.Printf("Applying metadata tags to %s...", downloadedFile)
		err = UpdateTrackMetadataExpanded(downloadedFile, title, artist, album)
		if err != nil {
			log.Printf("ERROR: Failed to apply ID3 tags to new file %s: %v", downloadedFile, err)
			return
		}
		log.Printf("Successfully applied metadata tags to %s", downloadedFile)
	}

	log.Printf("Finished async job for %s", url)
}

// sanitizePath helps prevent directory traversal or invalid characters in path creation
func sanitizePath(input string) string {
	s := strings.ReplaceAll(input, "/", "_")
	s = strings.ReplaceAll(s, "\\", "_")
	s = strings.ReplaceAll(s, ":", "_")
	return strings.TrimSpace(s)
}

// extractFilenameFromYTDLPOutput parses the standard yt-dlp CLI output to find the exact final .mp3 path
func extractFilenameFromYTDLPOutput(output string) string {
	lines := strings.Split(output, "\n")
	// The final move log looks like:
	// [ExtractAudio] Destination: /music/Unknown Artist/Unknown Album/SongTitle (xxxxxx).mp3
	for _, line := range lines {
		if strings.Contains(line, "[ExtractAudio] Destination:") {
			parts := strings.SplitN(line, "Destination: ", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return ""
}
