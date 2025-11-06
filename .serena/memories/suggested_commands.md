# Suggested Commands

## Development Commands

### Backend (Go)

#### Running the Server
```bash
cd backend
go run cmd/server/main.go
```

#### Testing
```bash
cd backend
go test ./...                    # Run all tests
go test -v ./...                 # Verbose output
go test ./internal/config        # Test specific package
go test -run TestName            # Run specific test
```

#### Building
```bash
cd backend
go build -o server cmd/server/main.go
```

#### Module Management
```bash
cd backend
go mod download                  # Download dependencies
go mod tidy                      # Clean up dependencies
go mod verify                    # Verify dependencies
```

#### Formatting and Linting
```bash
cd backend
gofmt -w .                       # Format all Go files
go vet ./...                     # Run static analysis
```

### Frontend (React)

#### Development Server
```bash
cd frontend
npm run dev                      # Start Vite dev server (http://localhost:5173)
```

#### Building
```bash
cd frontend
npm run build                    # Build for production
npm run preview                  # Preview production build
```

#### Linting
```bash
cd frontend
npm run lint                     # Run ESLint
```

#### Package Management
```bash
cd frontend
npm install                      # Install dependencies
npm install <package>            # Add new package
```

### Docker Compose

#### Starting Services
```bash
docker-compose up -d             # Start all services in background
docker-compose up                # Start with logs visible
```

#### Stopping Services
```bash
docker-compose down              # Stop and remove containers
docker-compose stop              # Stop without removing
```

#### Viewing Logs
```bash
docker-compose logs              # View all logs
docker-compose logs backend      # View backend logs
docker-compose logs frontend     # View frontend logs
docker-compose logs -f           # Follow logs in real-time
```

#### Rebuilding
```bash
docker-compose build             # Rebuild all images
docker-compose build backend     # Rebuild specific service
docker-compose up -d --build     # Rebuild and start
```

#### Service Management
```bash
docker-compose ps                # List running containers
docker-compose restart backend   # Restart specific service
```

### Database

#### SQLite Access
```bash
sqlite3 data/streamtime.db       # Open database
```

#### Common SQLite Commands (inside sqlite3)
```sql
.tables                          -- List tables
.schema services                 -- Show table schema
SELECT * FROM services;          -- Query data
.quit                           -- Exit
```

### Git Commands

```bash
git status                       # Check status
git add .                        # Stage changes
git commit -m "message"          # Commit with message
git log --oneline -5             # View recent commits
git diff                         # View unstaged changes
```

### System Commands (macOS)

```bash
ls -la                           # List files with details
cd <directory>                   # Change directory
pwd                              # Print working directory
find . -name "*.go"              # Find files by pattern
grep -r "pattern" .              # Search in files
```

## Configuration

### Setup
```bash
cp config.example.yaml config.yaml
# Edit config.yaml with your streaming service credentials
```

### Default Ports
- Backend API: http://localhost:8080
- Frontend Dev: http://localhost:5173 (Vite)
- Frontend Prod: http://localhost:3000 (Docker)

## Environment

- **Platform**: Darwin (macOS)
- **Go Version**: 1.24
- **Node Version**: 18+ required
