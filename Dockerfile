# Pin the builder exclusively to the platform executing the build (e.g. GitHub Actions amd64)
# to completely bypass QEMU software CPU emulation for the Go compiler itself.
FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source code
COPY . .

# Dynamically receive the target platform (e.g. linux/arm64) from Docker Buildx
ARG TARGETOS
ARG TARGETARCH

# Build the Go app
# CGO_ENABLED=0 ensures a static binary, good for minimal containers
# We inject TARGETOS and TARGETARCH natively into the insanely fast Go cross-compiler
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o sidecar .

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
