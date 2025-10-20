# URL Shortener Service

A production-ready, high-performance URL shortener service built with Go, PostgreSQL, and Redis. Designed for scalability, security, and ease of deployment.

## 🚀 Features

- **High Performance**: Redis caching, connection pooling, and optimized database queries
- **Scalable Architecture**: Microservices-ready with clear separation of concerns
- **Security First**: Rate limiting, input validation, SQL injection prevention
- **Production Ready**: Health checks, metrics, structured logging, graceful shutdown
- **Developer Friendly**: Comprehensive testing, Docker support, Makefile automation

## 🏗 Architecture

Presentation Layer (HTTP Handlers)
↓
Service Layer (Business Logic)
↓
Repository Layer (Data Access)
↓
Data Layer (PostgreSQL + Redis)


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
git clone https://github.com/your-org/url-shortener.git
cd url-shortener
cp .env.example .env