# Glyph API

## Quick Installation

- **[I don't have Navidrome installed](#scenario-a-new-installation-no-existing-navidrome-setup)**
- **[I have Navidrome Installed](#scenario-b-adding-glyph-api-to-an-existing-navidrome-setup)**

A lightweight "Sidecar" REST API built in Go, designed to run alongside a self-hosted [Navidrome](https://www.navidrome.org/) instance. 

I built this project to create a complete music player experience where I have 100% control over my content and files. While Navidrome is excellent for streaming and managing playback, Glyph API runs in the background to make new song download and metadata management easier.

## Features

Glyph API shares the exact same `/music` folder as Navidrome and provides REST endpoints for:

- **Metadata Spot-Edits**: Send a single API request to update Title, Artist, or Album ID3 tags. The API uses precise Mutex locks to safely write changes to disk and prevent concurrent file corruption.
- **YouTube Audio Downloader**: Built-in support for `yt-dlp` and `ffmpeg`. Send a YouTube URL and metadata to the API, and a configurable background engine will asynchronously download the audio stream, apply the metadata, and save it directly to your Navidrome library.

## Installation

This application runs as a self-contained Docker image, natively cross-compiled for `amd64`, `arm64`, and older ARM architectures (like Raspberry Pis).

### Scenario A: New Installation (No existing Navidrome setup)

If you are starting fresh, you can spin up the entire stack (Navidrome + Glyph API + Watchtower automatic updater) simultaneously.

1. **Prepare Directories**: Create folders to store your data safely on the host machine:
   ```bash
   mkdir -p music data
   ``` 
2. **Download Configuration**: Copy the `docker-compose.yml` file from this repository to your server.
3. **Configure Secrets**: Edit the `docker-compose.yml` file:
   - Change the `API_KEY` placeholder to a secure custom password.
   - Ensure the `./music` volume in both the navidrome and glyph-api services point accurately to your host's music directory. Do not get rid of the :ro and :rw flags!
   - *Optional:* Configure `MAX_DOWNLOAD_WORKERS` to control the background `yt-dlp` download queue (default: `3`).
4. **Start the Stack**:
   ```bash
   docker compose up -d
   ```
5. Access Navidrome in your browser at `http://<your-ip>:4533` to set up your admin account. The Glyph API will be silently listening on port `8080`.

---

### Scenario B: Adding Glyph API to an Existing Navidrome Setup

If you already have Navidrome running via Docker Compose, you can add Glyph API directly to your active configuration.

1. Open your existing Navidrome `docker-compose.yml` file.
2. Append the `glyph-api` service block:
   ```yaml
   services:
     # ... your existing navidrome block ...

     glyph-api:
       image: ghcr.io/Henry-Turtle/glyph-api:main
       container_name: glyph-api
       ports:
         - "8080:8080"
       restart: unless-stopped
       environment:
         - API_KEY=replace-with-your-key # Choose a strong password
         - MAX_DOWNLOAD_WORKERS=3
         # If you have custom directories inside the container not mapped to "/music" or "/data", activate these lines:
         # - MUSIC_DIR=/my_custom_music_vault
         # - DATA_DIR=/my_custom_data_vault
       volumes:
         # REQUIRED: Mount Navidrome's database read-only so Glyph can securely resolve Subsonic IDs
         - "/path/to/your/navidrome/data:/data:ro"
         # IMPORTANT: This absolute path on the left MUST perfectly match Navidrome's!
         - "/path/to/your/music/folder:/music:rw"
   ```
3. Pull the image and restart your stack:
   ```bash
   docker compose down
   docker compose up -d
   ```

## Automatic Updates

The `main` branch of this repository automatically builds a new Docker Image via GitHub Actions.

To keep your server updated automatically, the provided `docker-compose.yml` includes **Watchtower**. Watchtower is configured to check for updates at **4:00 AM** every day, gracefully stopping and updating Glyph API if a new version is available.

You can also manually pull the latest updates anytime:
```bash
docker compose pull
docker compose up -d
```

## API Usage

All requests must include your secure password in the `X-API-KEY` header.

### 1. Update Existing Track Metadata
Use the `/update-track` endpoint to instantly rewrite specific ID3 tags. Instead of providing fragile physical paths, simply send the EXACT `id` provided by the Navidrome/Subsonic API (e.g., `id: "6d84bd86287c8d9c5b"`).

The Sidecar safely connects to the local Navidrome SQLite database, resolves your `id` directly to its internal absolute filepath, and applies the physical edits.

```bash
curl -X POST http://localhost:8080/update-track \
  -H "X-API-KEY: my-super-secret-api-key" \
  -H "Content-Type: application/json" \
  -d '{"id": "6d84bd86287c8d9c5b", "title": "New Title", "artist": "New Artist"}'
```

### 2. Download New Track from YouTube
Use the `/download-track` endpoint to asynchronously fetch audio using `yt-dlp`. 
It accepts a `url`, a `quality` parameter (`max`, `mid`, or `low`), and optional exact metadata tags (`title`, `artist`, `album`). 

The API will return `202 Accepted` immediately, placing the download into the background processing queue.

```bash
curl -X POST http://localhost:8080/download-track \
  -H "X-API-KEY: my-super-secret-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://www.youtube.com/watch?v=dQw4w9WgXcQ", 
    "quality": "max", 
    "title": "Never Gonna Give You Up", 
    "artist": "Rick Astley",
    "album": "Whenever You Need Somebody"
  }'
```
