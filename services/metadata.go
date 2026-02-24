package services

import (
	"fmt"
	"sync"

	"github.com/bogem/id3v2/v2"
)

var editMutex sync.Mutex

// UpdateTrackMetadata opens an audio file using bogus/id3v2, safely updates the Title and Artist tags,
// and saves the changes back. Returns an error if any operation fails.
func UpdateTrackMetadata(filePath, title, artist string) error {
	return UpdateTrackMetadataExpanded(filePath, title, artist, "")
}

// UpdateTrackMetadataExpanded adds support for the Album tag and applies them conditionally
func UpdateTrackMetadataExpanded(filePath, title, artist, album string) error {
	editMutex.Lock()
	defer editMutex.Unlock()

	// Open file and parse tag in it.
	tag, err := id3v2.Open(filePath, id3v2.Options{Parse: true})
	if err != nil {
		return fmt.Errorf("failed to open file as id3v2 tag: %w", err)
	}
	defer tag.Close()

	// Update the necessary tags but only if they are not empty
	if title != "" {
		tag.SetTitle(title)
	}
	if artist != "" {
		tag.SetArtist(artist)
	}
	if album != "" {
		tag.SetAlbum(album)
	}

	// Save the changes back to the file
	if err := tag.Save(); err != nil {
		return fmt.Errorf("failed to save id3v2 tag changes: %w", err)
	}

	return nil
}
