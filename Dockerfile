FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source code
COPY . .

# Build the Go app
# CGO_ENABLED=0 ensures a static binary, good for minimal containers
RUN CGO_ENABLED=0 GOOS=linux go build -o sidecar .

# Start a new, smaller stage
FROM alpine:latest  

WORKDIR /root/

# Install required tools for yt-dlp to function
RUN apk add --no-cache ffmpeg python3 yt-dlp

# Set the number of parallel downloads
ENV MAX_DOWNLOAD_WORKERS=3

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /app/sidecar .

# EXPOSE port 8080 to the outside world
EXPOSE 8080

# Command to run the executable
CMD ["./sidecar"]
