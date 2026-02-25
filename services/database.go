package services

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

// ResolveSongPathByID connects strictly Read-Only to Navidrome's SQLite file
// to securely resolve a Subsonic API ID into its absolute container file path.
func ResolveSongPathByID(songID string) (string, error) {
	dbPath := "/data/navidrome.db"

	// Verify the database file physically exists in the container
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return "", errors.New("navidrome database not found at /data/navidrome.db (please ensure you mounted the /data volume natively)")
	}

	// URI formatting forces modernc.org/sqlite to not take exclusive locks
	// which prevents us from crashing or interfering with Navidrome's background scans.
	dsn := fmt.Sprintf("file:%s?mode=ro", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return "", fmt.Errorf("failed to open navidrome database: %v", err)
	}
	defer db.Close()

	var absPath string

	// Explicitly query the internal media_file table where Navidrome stores its indexed tracks
	err = db.QueryRow("SELECT path FROM media_file WHERE id = ?", songID).Scan(&absPath)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", errors.New("no song found matching that exact Subsonic ID in the database")
		}
		return "", fmt.Errorf("failed scanning database row: %v", err)
	}

	// Navidrome stores paths relative to its internal MusicFolder.
	// We use MUSIC_DIR (defaulting to /music) to accurately construct the internal absolute path!
	if !strings.HasPrefix(absPath, "/") {
		musicDir := os.Getenv("MUSIC_DIR")
		if musicDir == "" {
			musicDir = "/music"
		}
		absPath = filepath.Join(musicDir, absPath)
	}

	return absPath, nil
}
