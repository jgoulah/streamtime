# Build stage
FROM golang:alpine AS builder

# Install build dependencies for CGO (required for SQLite)
RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o server ./cmd/server

# Runtime stage
FROM alpine:latest

# Install runtime dependencies including Chromium for web scraping
RUN apk --no-cache add \
    ca-certificates \
    sqlite \
    chromium \
    chromium-chromedriver \
    nss \
    freetype \
    harfbuzz \
    ttf-freefont

# Tell Puppeteer/chromedp to skip downloading Chrome (we're using system Chromium)
ENV CHROME_BIN=/usr/bin/chromium-browser \
    CHROME_PATH=/usr/lib/chromium/

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/server .

# Create data directory
RUN mkdir -p /root/data

# Expose port
EXPOSE 8080

# Set environment variable for config path
ENV CONFIG_PATH=/root/config.yaml

# Run the server
CMD ["./server"]
