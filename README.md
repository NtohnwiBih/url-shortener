# URL Shortener Service

A production-ready, high-performance URL shortener service built with Go, PostgreSQL, and Redis. Designed for scalability, security, and ease of deployment.

## 🚀 Features

- **High Performance**: Redis caching, connection pooling, and optimized database queries
- **Scalable Architecture**: Microservices-ready with clear separation of concerns
- **Security First**: Rate limiting, input validation, SQL injection prevention
- **Production Ready**: Health checks, metrics, structured logging, graceful shutdown
- **Developer Friendly**: Comprehensive testing, Docker support, Makefile automation

## 🏗 Architecture

```
Presentation Layer (HTTP Handlers)
          ↓
Service Layer (Business Logic)
          ↓
Repository Layer (Data Access)
          ↓
Data Layer (PostgreSQL + Redis)
```

### Design Patterns
- **Repository Pattern** - Abstract data access
- **Dependency Injection** - Loose coupling between layers
- **Factory Pattern** - Short code generation
- **Cache-Aside Pattern** - Optimized read performance

## 📦 Quick Start

### Prerequisites
- Go 1.21+
- PostgreSQL 15+
- Redis 7+
- Docker & Docker Compose

### Local Development

1. **Clone and setup:**
```bash
git clone https://github.com/NtohnwiBih/url-shortener.git
cd url-shortener
cp .env.example .env
```

2. **Configure environment:**
Edit `.env` file with your settings:
```env
ENVIRONMENT=development
SERVER_PORT=8081

DB_HOST=localhost
DB_PORT=5432
DB_USER=urlshortener
DB_PASSWORD=securepassword
DB_NAME=urlshortener
DB_SSLMODE=disable

REDIS_ADDR=localhost:6379
REDIS_PASSWORD=redispassword
REDIS_DB=0

BASE_URL=http://localhost:8081
SHORT_CODE_LENGTH=7
CACHE_TTL_MINUTES=60
RATE_LIMIT_PER_MINUTE=100
```

3. **Run with Docker (Recommended):**
```bash
# Start all services
docker-compose -f docker/docker-compose.yml up -d

# View logs
docker-compose -f docker/docker-compose.yml logs -f app

# Stop services
docker-compose -f docker/docker-compose.yml down
```

4. **Or run locally:**
```bash
# Start PostgreSQL and Redis
docker-compose -f docker/docker-compose.yml up -d postgres redis

# Run migrations
psql -h localhost -U urlshortener -d urlshortener -f migrations/001_create_urls_table.sql

# Run the application
go run cmd/server/main.go
```

## 📡 API Endpoints

### Create Short URL
```bash
POST /api/v1/shorten
Content-Type: application/json

{
  "url": "https://github.com/golang/go",
  "custom_code": "golang" // Optional
}

Response:
{
  "short_code": "fKDdXBb",
  "short_url": "http://localhost:8081/fKDdXBb",
  "original_url": "https://github.com/golang/go",
  "created_at": "2025-10-20T20:26:21Z"
}
```

### Redirect to Original URL
```bash
GET /:shortCode

Response: 301 Redirect to original URL
```

### Get URL Information
```bash
GET /api/v1/urls/:shortCode

Response:
{
  "short_code": "fKDdXBb",
  "original_url": "https://github.com/golang/go",
  "click_count": 42,
  "created_at": "2025-10-20T20:26:21Z",
  "expires_at": null
}
```

### Get Click Statistics
```bash
GET /api/v1/urls/:shortCode/stats

Response:
{
  "short_code": "fKDdXBb",
  "click_count": 42,
  "created_at": "2025-10-20T20:26:21Z",
  "last_accessed": "2025-10-20T21:30:15Z"
}
```

### Delete Short URL
```bash
DELETE /api/v1/urls/:shortCode

Response:
{
  "message": "URL deleted successfully"
}
```

### Health Check
```bash
GET /health

Response:
{
  "status": "healthy",
  "service": "url-shortener",
  "version": "1.0.0"
}
```

## 🧪 Testing

### PowerShell (Windows)
```powershell
# Create a short URL
Invoke-RestMethod -Uri http://localhost:8081/api/v1/shorten -Method Post -ContentType "application/json" -Body '{"url": "https://example.com"}'

# Get URL info
Invoke-RestMethod -Uri http://localhost:8081/api/v1/urls/fKDdXBb -Method Get

# Get statistics
Invoke-RestMethod -Uri http://localhost:8081/api/v1/urls/fKDdXBb/stats -Method Get

# Test redirect (with no redirect follow)
Invoke-WebRequest -Uri http://localhost:8081/fKDdXBb -Method Get -MaximumRedirection 0
```

### Bash/Linux/Mac
```bash
# Create a short URL
curl -X POST http://localhost:8081/api/v1/shorten \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com"}'

# Get URL info
curl http://localhost:8081/api/v1/urls/fKDdXBb

# Test redirect
curl -I http://localhost:8081/fKDdXBb
```

### Run Unit Tests
```bash
# Run all tests
go test ./... -v

# Run with coverage
go test ./... -cover

# Generate coverage report
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## 📁 Project Structure

```
url-shortener/
├── cmd/
│   └── server/
│       └── main.go              # Application entry point
├── internal/
│   ├── cache/
│   │   └── redis.go             # Redis cache implementation
│   ├── config/
│   │   └── config.go            # Configuration management
│   ├── handler/
│   │   ├── url_handler.go       # HTTP request handlers
│   │   └── middleware.go        # Custom middleware
│   ├── model/
│   │   └── url.go               # Domain models
│   ├── repository/
│   │   └── postgres/
│   │       └── url_repository.go # Data access layer
│   └── service/
│       └── url_service.go       # Business logic layer
├── pkg/
│   └── logger/
│       └── logger.go            # Structured logging
├── migrations/
│   └── 001_create_urls_table.sql # Database migrations
├── docker/
│   ├── Dockerfile               # Multi-stage Docker build
│   └── docker-compose.yml       # Docker Compose config
├── .env.example                 # Environment variables template
├── go.mod                       # Go module dependencies
├── go.sum                       # Dependency checksums
├── Makefile                     # Build automation
└── README.md                    # This file
```

## 🔧 Configuration

All configuration is done via environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `ENVIRONMENT` | Environment (development/production) | `development` |
| `SERVER_PORT` | HTTP server port | `8081` |
| `DB_HOST` | PostgreSQL host | `localhost` |
| `DB_PORT` | PostgreSQL port | `5432` |
| `DB_USER` | Database user | `urlshortener` |
| `DB_PASSWORD` | Database password | - |
| `DB_NAME` | Database name | `urlshortener` |
| `DB_SSLMODE` | SSL mode (disable/require) | `disable` |
| `REDIS_ADDR` | Redis address | `localhost:6379` |
| `REDIS_PASSWORD` | Redis password | - |
| `REDIS_DB` | Redis database number | `0` |
| `BASE_URL` | Base URL for short links | `http://localhost:8081` |
| `SHORT_CODE_LENGTH` | Length of generated codes | `7` |
| `CACHE_TTL_MINUTES` | Cache expiration time | `60` |
| `RATE_LIMIT_PER_MINUTE` | API rate limit | `100` |

## 🚀 Deployment

### Docker Production Build

```bash
# Build production image
docker build -f docker/Dockerfile -t url-shortener:latest .

# Run with custom configuration
docker run -d \
  --name url-shortener \
  -p 8081:8081 \
  -e ENVIRONMENT=production \
  -e DB_HOST=your-db-host \
  -e DB_PASSWORD=your-db-password \
  -e REDIS_ADDR=your-redis-host:6379 \
  url-shortener:latest
```

### Kubernetes

Create a `deployment.yaml`:
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: url-shortener
spec:
  replicas: 3
  selector:
    matchLabels:
      app: url-shortener
  template:
    metadata:
      labels:
        app: url-shortener
    spec:
      containers:
      - name: url-shortener
        image: url-shortener:latest
        ports:
        - containerPort: 8081
        env:
        - name: ENVIRONMENT
          value: "production"
        - name: DB_HOST
          valueFrom:
            secretKeyRef:
              name: db-secret
              key: host
```

## 🔒 Security Features

- **Rate Limiting**: Prevents abuse with configurable limits
- **Input Validation**: Validates URLs and sanitizes input
- **SQL Injection Prevention**: Parameterized queries with GORM
- **CORS Configuration**: Configurable cross-origin policies
- **Security Headers**: X-Content-Type-Options, X-Frame-Options, etc.
- **Graceful Shutdown**: Prevents data loss on shutdown

## 📊 Monitoring & Observability

- **Structured Logging**: JSON logs with contextual information
- **Health Checks**: `/health` endpoint for load balancers
- **Metrics Ready**: Easily integrate with Prometheus
- **Error Tracking**: Comprehensive error logging and handling

## 🛠 Development

### Building

```bash
# Build binary
go build -o bin/server cmd/server/main.go

# Build with optimizations
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
  -ldflags="-w -s" \
  -o bin/server \
  cmd/server/main.go
```

### Code Quality

```bash
# Format code
go fmt ./...

# Run linter
golangci-lint run

# Run tests with race detector
go test -race ./...

# Check dependencies
go mod tidy
go mod verify
```

## 🐛 Troubleshooting

### Port Already in Use
```bash
# Windows
netstat -ano | findstr :8081
taskkill /PID <pid> /F

# Linux/Mac
lsof -ti:8081 | xargs kill -9
```

### Database Connection Issues
```bash
# Check if PostgreSQL is running
docker ps | grep postgres

# Check connection
psql -h localhost -U urlshortener -d urlshortener -c "SELECT 1;"
```

### Redis Connection Issues
```bash
# Check if Redis is running
docker ps | grep redis

# Test connection
redis-cli -h localhost -p 6379 -a redispassword ping
```

### View Docker Logs
```bash
# All services
docker-compose -f docker/docker-compose.yml logs

# Specific service
docker-compose -f docker/docker-compose.yml logs app
docker-compose -f docker/docker-compose.yml logs postgres
docker-compose -f docker/docker-compose.yml logs redis
```

## 📝 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 👥 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📧 Contact

For questions or support, please open an issue on GitHub.

## 🙏 Acknowledgments

- [Gin Web Framework](https://github.com/gin-gonic/gin)
- [GORM](https://gorm.io/)
- [Go Redis](https://github.com/go-redis/redis)
- [PostgreSQL](https://www.postgresql.org/)

---

**Built with ❤️ using Go**