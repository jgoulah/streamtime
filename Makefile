.PHONY: build build-export-cookies install-remote run clean help refresh-cookies setup-systemd-remote

# Variables
BACKEND_DIR = backend
FRONTEND_DIR = frontend
EXPORT_COOKIES_DIR = $(BACKEND_DIR)/cmd/export-cookies
SERVER_DIR = $(BACKEND_DIR)/cmd/server
SCRAPER_DIR = $(BACKEND_DIR)/cmd/scraper
BACKEND_BINARY = $(BACKEND_DIR)/server
EXPORT_COOKIES_BINARY = $(BACKEND_DIR)/export-cookies
SCRAPER_BINARY = $(BACKEND_DIR)/scraper
FRONTEND_DIST = $(FRONTEND_DIR)/dist

# Cross-compilation settings for remote Linux x86_64 host
GOOS = linux
GOARCH = amd64
CGO_ENABLED = 1
CC = x86_64-linux-musl-gcc

# Remote host configuration
REMOTE_HOST ?= mediaserver
REMOTE_BIN_DIR ?= /opt/streamtime
REMOTE_ETC_DIR ?= /usr/local/etc/streamtime

## build: Build the backend server binary for local use
build:
	@echo "Building backend server (local)..."
	cd $(BACKEND_DIR) && go build -o server ./cmd/server

## build-scraper: Build the scraper-only binary for local use
build-scraper:
	@echo "Building scraper (local)..."
	cd $(BACKEND_DIR) && go build -o scraper ./cmd/scraper

## build-export-cookies: Build the export-cookies tool for local use
build-export-cookies:
	@echo "Building export-cookies tool (local)..."
	cd $(BACKEND_DIR) && go build -o export-cookies ./cmd/export-cookies

## build-frontend: Build the frontend for production
build-frontend:
	@echo "Building frontend..."
	cd $(FRONTEND_DIR) && npm install && npm run build

## build-all: Build all binaries for local use
build-all: build build-scraper build-export-cookies

## build-remote: Build all binaries for remote Linux x86_64 using Docker (static)
build-remote:
	@echo "Building static binaries for remote Linux x86_64 using Docker..."
	@docker run --rm --platform linux/amd64 -v $(PWD)/$(BACKEND_DIR):/build -w /build golang:alpine sh -c '\
		apk add --no-cache gcc musl-dev sqlite-dev && \
		go build -ldflags "-linkmode external -extldflags -static" -o server ./cmd/server && \
		go build -ldflags "-linkmode external -extldflags -static" -o scraper ./cmd/scraper && \
		go build -ldflags "-linkmode external -extldflags -static" -o export-cookies ./cmd/export-cookies'
	@echo "Remote build complete!"

## install-remote: Build and install binaries to remote host
install-remote: build-remote build-frontend
	@echo "Installing to $(REMOTE_HOST)..."
	@echo "  Binaries -> $(REMOTE_BIN_DIR)"
	@echo "  Config/DB -> $(REMOTE_ETC_DIR)"
	@echo "  Frontend -> $(REMOTE_BIN_DIR)/frontend/dist"
	@echo ""
	@ssh $(REMOTE_HOST) "mkdir -p $(REMOTE_BIN_DIR) $(REMOTE_ETC_DIR) $(REMOTE_BIN_DIR)/frontend"
	@scp $(BACKEND_BINARY) $(REMOTE_HOST):$(REMOTE_BIN_DIR)/server.new
	@scp $(SCRAPER_BINARY) $(REMOTE_HOST):$(REMOTE_BIN_DIR)/scraper.new
	@scp $(EXPORT_COOKIES_BINARY) $(REMOTE_HOST):$(REMOTE_BIN_DIR)/export-cookies.new
	@scp -r $(FRONTEND_DIST) $(REMOTE_HOST):$(REMOTE_BIN_DIR)/frontend/
	@scp config.yaml $(REMOTE_HOST):$(REMOTE_ETC_DIR)/config.yaml
	@scp deployment/streamtime.service $(REMOTE_HOST):$(REMOTE_BIN_DIR)/streamtime.service
	@ssh $(REMOTE_HOST) "mv $(REMOTE_BIN_DIR)/server.new $(REMOTE_BIN_DIR)/server && \
		mv $(REMOTE_BIN_DIR)/scraper.new $(REMOTE_BIN_DIR)/scraper && \
		mv $(REMOTE_BIN_DIR)/export-cookies.new $(REMOTE_BIN_DIR)/export-cookies && \
		chmod +x $(REMOTE_BIN_DIR)/server $(REMOTE_BIN_DIR)/scraper $(REMOTE_BIN_DIR)/export-cookies"
	@echo ""
	@echo "Restarting service if it exists..."
	@ssh $(REMOTE_HOST) "sudo systemctl restart streamtime 2>/dev/null && echo 'Service restarted' || echo 'Service not yet set up - run: make setup-systemd-remote'"
	@echo ""
	@echo "Files installed successfully!"
	@echo ""
	@echo "Access web UI at: http://$(REMOTE_HOST):8080"

## run: Run the backend server locally
run: build
	@echo "Starting backend server locally..."
	cd $(BACKEND_DIR) && ./server

## docker-build: Build Docker images
docker-build:
	@echo "Building Docker images..."
	docker-compose build

## docker-up: Start Docker containers
docker-up:
	@echo "Starting Docker containers..."
	docker-compose up -d

## docker-down: Stop Docker containers
docker-down:
	@echo "Stopping Docker containers..."
	docker-compose down

## docker-logs: Show Docker logs
docker-logs:
	docker-compose logs -f backend

## refresh-cookies: Refresh cookies for a specific service (use SERVICE=netflix|youtube_tv|amazon_video)
refresh-cookies: build-export-cookies
	@if [ -z "$(SERVICE)" ]; then \
		echo "Error: SERVICE variable required"; \
		echo "Usage: make refresh-cookies SERVICE=netflix"; \
		echo "       make refresh-cookies SERVICE=youtube_tv"; \
		echo "       make refresh-cookies SERVICE=amazon_video"; \
		exit 1; \
	fi
	@echo "Refreshing cookies for $(SERVICE)..."
	cd $(BACKEND_DIR) && ./export-cookies --service $(SERVICE)

## setup-systemd-remote: Set up and start systemd service on remote host
setup-systemd-remote:
	@echo "Setting up systemd service on $(REMOTE_HOST)..."
	@ssh -t $(REMOTE_HOST) "sudo cp $(REMOTE_BIN_DIR)/streamtime.service /etc/systemd/system/ && \
		sudo systemctl daemon-reload && \
		sudo systemctl enable streamtime && \
		sudo systemctl restart streamtime && \
		sudo systemctl status streamtime --no-pager"
	@echo ""
	@echo "Service started successfully!"
	@echo ""
	@echo "Useful commands:"
	@echo "  View logs: ssh $(REMOTE_HOST) 'sudo journalctl -u streamtime -f'"
	@echo "  Restart:   ssh $(REMOTE_HOST) 'sudo systemctl restart streamtime'"
	@echo "  Stop:      ssh $(REMOTE_HOST) 'sudo systemctl stop streamtime'"
	@echo ""
	@echo "Access web UI at: http://$(REMOTE_HOST):8080"

## clean: Remove built binaries
clean:
	@echo "Cleaning build artifacts..."
	rm -f $(BACKEND_BINARY) $(SCRAPER_BINARY) $(EXPORT_COOKIES_BINARY)

## help: Show this help message
help:
	@echo "StreamTime Makefile"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'

.DEFAULT_GOAL := help
