# GoLang Microservice Template

A production-ready backend microservice template built with Go, featuring REST API, PostgreSQL database, structured logging, Prometheus metrics, and Swagger documentation.

## 🎯 Features

- ✅ **RESTful API** with Gorilla Mux router
- ✅ **PostgreSQL Database** with connection pooling
- ✅ **Structured Logging** using Logrus (JSON format for log aggregation)
- ✅ **Prometheus Metrics** for monitoring and observability
- ✅ **Swagger Documentation** auto-generated from code
- ✅ **YAML Configuration** with environment variable substitution
- ✅ **Docker & Docker Compose** for easy deployment
- ✅ **Graceful Shutdown** handling

## 📦 Tech Stack

- **Language:** Go 1.23+
- **Router:** Gorilla Mux
- **Database:** PostgreSQL 13
- **Database Driver:** lib/pq
- **Logging:** Logrus
- **Metrics:** Prometheus client
- **Configuration:** Viper
- **Documentation:** Swagger/OpenAPI

## 🚀 Quick Start

### Prerequisites

- Docker and Docker Compose installed
- Go 1.23+ (for local development)

### 1. Configuration

The service uses `config.yaml` with environment variable substitution:

```yaml
environment: ${ENVIRONMENT:development}

server:
  port: ${SERVER_PORT:8080}

database:
  host: ${DB_HOST:localhost}
  port: ${DB_PORT:5432}
  user: ${DB_USER:postgres}
  password: ${DB_PASSWORD:postgres}
  name: ${DB_NAME:mydb}
```

Pattern: `${VARIABLE_NAME:default_value}`

### 2. Start the Service

```bash
docker-compose -f Context/Data/docker-compose.yml up --build
```

### 3. Test the API

Run the test script:

```bash
./test-api.sh
```

Or test manually:

```bash
# Health check
curl http://localhost:8080/health

# Get all users
curl http://localhost:8080/users

# Create user
curl -X POST -H "Content-Type: application/json" \
  -d '{"username":"testuser","email":"test@example.com"}' \
  http://localhost:8080/users
```

## 📚 API Documentation

Access the interactive Swagger UI:
- **URL:** http://localhost:8080/swagger/index.html

## 📊 Monitoring

### Prometheus Metrics

Access metrics endpoint:
```bash
curl http://localhost:8080/metrics
```

Available metrics:
- `http_requests_total` - Total HTTP requests by method, endpoint, and status
- `http_request_duration_seconds` - Request duration histogram
- Standard Go runtime metrics (CPU, memory, goroutines)

### Logs

View structured JSON logs:
```bash
docker-compose -f Context/Data/docker-compose.yml logs -f app
```

Log format:
```json
{
  "timestamp": "2025-11-09T01:13:15.295Z",
  "level": "info",
  "message": "User created successfully",
  "user_id": 4,
  "username": "demo_user_1"
}
```

## 🏗️ Project Structure

```
golang-backend-service/
├── cmd/
│   └── server/
│       └── main.go              # Application entry point
├── internal/
│   ├── api/
│   │   └── routes.go            # HTTP handlers and routing
│   ├── config/
│   │   ├── config.go            # Configuration loading
│   │   └── evaluator.go         # Environment variable evaluation
│   ├── database/
│   │   └── postgres.go          # Database operations
│   └── logger/
│       └── logger.go            # Logging setup
├── docs/                        # Swagger documentation (auto-generated)
├── Context/Data/
│   ├── docker-compose.yml       # Container orchestration
│   └── init.sql                 # Database initialization
├── config.yaml                  # Application configuration
├── Dockerfile                   # Container build
├── test-api.sh                  # API testing script
└── README.md                    # This file
```

## 🛠️ Development

### Local Development (without Docker)

1. Install PostgreSQL locally
2. Update `config.yaml` or set environment variables
3. Generate Swagger docs:
   ```bash
   swag init -g cmd/server/main.go -o docs
   ```
4. Run the service:
   ```bash
   go run cmd/server/main.go
   ```

### Adding New Endpoints

1. Add handler function in `internal/api/routes.go`
2. Add Swagger annotations
3. Register route in `SetupRoutes()`
4. Regenerate docs:
   ```bash
   swag init -g cmd/server/main.go -o docs
   ```

## 🔧 Configuration

### Environment Variables

All config values support environment variables:

**Server:**
- `SERVER_PORT` - HTTP port (default: 8080)
- `SERVER_READ_TIMEOUT` - Read timeout (default: 15s)
- `SERVER_WRITE_TIMEOUT` - Write timeout (default: 15s)
- `SERVER_IDLE_TIMEOUT` - Idle timeout (default: 60s)

**Database:**
- `DB_HOST` - Database host (default: localhost)
- `DB_PORT` - Database port (default: 5432)
- `DB_USER` - Database user (default: postgres)
- `DB_PASSWORD` - Database password (default: postgres)
- `DB_NAME` - Database name (default: mydb)
- `DB_SSLMODE` - SSL mode (default: disable)
- `DB_MAX_OPEN_CONNS` - Max open connections (default: 25)
- `DB_MAX_IDLE_CONNS` - Max idle connections (default: 5)
- `DB_CONN_MAX_LIFETIME` - Connection max lifetime (default: 5m)

**Logging:**
- `LOG_LEVEL` - Log level: debug, info, warn, error (default: info)
- `LOG_FORMAT` - Log format: json or console (default: json)

**Monitoring:**
- `MONITORING_ENABLED` - Enable monitoring (default: true)
- `PROMETHEUS_ENABLED` - Enable Prometheus (default: true)
- `PROMETHEUS_PATH` - Metrics path (default: /metrics)

## 🐳 Docker Commands

```bash
# Build and start
docker-compose -f Context/Data/docker-compose.yml up --build

# Start in background
docker-compose -f Context/Data/docker-compose.yml up -d

# View logs
docker-compose -f Context/Data/docker-compose.yml logs -f app

# Stop
docker-compose -f Context/Data/docker-compose.yml down

# Stop and remove volumes
docker-compose -f Context/Data/docker-compose.yml down -v
```

## 🧪 Testing

Run the comprehensive test suite:

```bash
./test-api.sh
```

Tests include:
- Health check endpoint
- GET all users
- POST create user
- GET user by ID
- Metrics endpoint

## 📈 Monitoring Integration

This template is designed to integrate with:

- **EFK Stack** (Elasticsearch, Fluentd, Kibana) - JSON logs
- **Logstash** - Structured logging
- **Datadog** - Metrics and logs
- **Prometheus/Grafana** - Metrics scraping and visualization
- **AWS CloudWatch** - Cloud monitoring
- **Google Cloud Logging** - GCP integration

## 🔒 Security Best Practices

- ✅ Secrets loaded from environment variables
- ✅ No hardcoded credentials
- ✅ Connection timeouts configured
- ✅ Graceful shutdown prevents data loss
- ✅ PostgreSQL SSL mode configurable

## 📝 License

This is a template project for backend microservices.

## 🤝 Contributing

This is a template repository. Fork and customize for your needs!

## 📖 Additional Resources

- [Go Documentation](https://golang.org/doc/)
- [Gorilla Mux](https://github.com/gorilla/mux)
- [Logrus](https://github.com/sirupsen/logrus)
- [Prometheus](https://prometheus.io/)
- [Swagger](https://swagger.io/)
- [Viper](https://github.com/spf13/viper)
