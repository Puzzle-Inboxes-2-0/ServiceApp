# GoLang Backend Service

A production-ready backend microservice built with Go, featuring REST API, PostgreSQL database, structured logging, Prometheus metrics, Swagger documentation, and **IP Reputation & Blacklist Detection System**.

## 🎯 Features

### Core Features
- ✅ **RESTful API** with Gorilla Mux router
- ✅ **PostgreSQL Database** with connection pooling
- ✅ **Structured Logging** using Logrus (JSON format for log aggregation)
- ✅ **Prometheus Metrics** for monitoring and observability
- ✅ **Swagger Documentation** auto-generated from code
- ✅ **YAML Configuration** with environment variable substitution
- ✅ **Docker & Docker Compose** for easy deployment
- ✅ **Graceful Shutdown** handling

### IP Reputation System (NEW!)
- ✅ **SMTP Failure Tracking** - Processes Stalwart mail server webhooks
- ✅ **Automated IP Reputation Monitoring** - 4-tier status system (healthy/warning/quarantine/blacklisted)
- ✅ **DNSBL Integration** - Checks 8 major blacklists (Spamhaus, Barracuda, SpamCop, etc.)
- ✅ **Background Aggregation** - Automatic metrics calculation every 5 minutes
- ✅ **Decision Algorithm** - RFC 5321/3463 compliant SMTP error code analysis
- ✅ **Real-time Webhooks** - Receives and processes delivery failure events
- ✅ **Comprehensive API** - 7 endpoints for IP reputation management
- ✅ **Prometheus Metrics** - 9 new metrics for IP reputation monitoring

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

### 1. Start the Service

```bash
# Use docker compose (not docker-compose) for Docker Compose V2
docker compose -f Context/Data/docker-compose.yml up --build -d
```

### 2. Verify It's Running

```bash
# Check health
curl http://127.0.0.1:8080/health

# View logs
docker compose -f Context/Data/docker-compose.yml logs -f app
```

### 3. Test the API

```bash
# Run comprehensive test suite
./scripts/test-api.sh

# Test IP reputation system
./scripts/test-ip-reputation.sh
```

## 📚 API Endpoints

### Core Endpoints
- `GET /health` - Health check
- `GET /users` - List all users
- `POST /users` - Create user
- `GET /users/{id}` - Get user by ID
- `GET /metrics` - Prometheus metrics
- `GET /swagger/index.html` - Swagger UI

### IP Reputation Endpoints
- `POST /api/webhooks/stalwart/delivery-failure` - Receive SMTP failure webhooks
- `GET /api/ips/{ip}/reputation` - Get IP reputation status
- `GET /api/ips/{ip}/failures?window=15m` - View SMTP failures for IP
- `POST /api/ips/{ip}/quarantine` - Manually quarantine an IP
- `POST /api/ips/{ip}/dnsbl-check` - Run DNSBL check
- `GET /api/dashboard/ip-health` - IP health dashboard
- `POST /api/testing/simulate-failures` - Simulate failures (testing)

**Interactive API Documentation:**
- **Swagger UI:** http://localhost:8080/swagger/index.html

## 🏗️ Project Structure

```
golang-backend-service/
├── cmd/
│   └── server/
│       └── main.go                # Application entry point
├── internal/
│   ├── api/
│   │   ├── routes.go              # HTTP handlers and routing
│   │   └── ip_reputation_handlers.go  # IP reputation API handlers
│   ├── config/
│   │   ├── config.go              # Configuration loading
│   │   └── evaluator.go           # Environment variable evaluation
│   ├── database/
│   │   ├── postgres.go            # Database operations
│   │   └── ip_reputation.go       # IP reputation database layer
│   ├── logger/
│   │   └── logger.go              # Logging setup
│   └── reputation/
│       ├── decision.go             # IP status decision algorithm
│       ├── dnsbl.go               # DNSBL checking integration
│       ├── aggregation.go         # Background aggregation service
│       └── metrics.go             # Prometheus metrics
├── scripts/                       # Helper scripts
│   ├── check-docker.sh            # Docker status checker
│   ├── start-services.sh          # Start application
│   ├── start-with-monitoring.sh   # Start with full monitoring stack
│   ├── stop-services.sh           # Stop services
│   ├── test-api.sh                # API testing script
│   ├── test-ip-reputation.sh      # IP reputation test suite
│   ├── view-logs.sh               # Pretty log viewer
│   └── filter-logs.sh             # Log filtering tool
├── web/                           # Web dashboards
│   ├── explore.html               # Interactive API explorer
│   └── monitor.html                # Real-time metrics dashboard
├── guides/                        # Complete documentation (2 files)
│   ├── DEVELOPER_GUIDE.md         # Getting started, testing, common issues
│   └── TECHNICAL_REFERENCE.md     # Architecture, monitoring, production
├── docs/                          # Swagger documentation (auto-generated)
├── Context/Data/                  # Docker configuration
│   ├── docker-compose.yml         # Standard services
│   ├── docker-compose.monitoring.yml  # With monitoring stack
│   ├── init.sql                   # Database initialization
│   ├── prometheus.yml             # Prometheus config
│   ├── grafana-datasources.yml    # Grafana datasources
│   └── promtail-config.yml        # Promtail config
├── config.yaml                    # Application configuration
├── Dockerfile                     # Container build
└── README.md                      # This file
```

## 📖 Documentation

**Complete documentation in the `guides/` folder - now streamlined to 2 essential files:**

### For Quick Start & Development
**[📘 DEVELOPER_GUIDE.md](guides/DEVELOPER_GUIDE.md)** - Everything developers need
- Getting started (setup, first run, testing)
- Development workflows (adding endpoints, metrics)
- Complete testing guide
- Common issues and solutions
- Backend development patterns
- API development & database operations
- Best practices

### For Architecture & Production
**[📗 TECHNICAL_REFERENCE.md](guides/TECHNICAL_REFERENCE.md)** - Deep technical documentation
- Architecture overview & project structure
- IP reputation system (4-tier status, DNSBL, decision algorithm)
- Database schema & queries
- Monitoring setup (Prometheus, Grafana, Loki)
- Configuration management
- Production deployment
- Performance tuning & security
- Product requirements (PRD)


## 🎯 IP Reputation System

The IP Reputation System automatically monitors SMTP delivery failures and determines when IPs should be flagged, quarantined, or blacklisted.

### Quick Example

```bash
# Simulate a blacklisted IP
curl -X POST http://127.0.0.1:8080/api/testing/simulate-failures \
  -H "Content-Type: application/json" \
  -d '{
    "ip": "203.0.113.100",
    "total_sent": 500,
    "failures": [
      {"code": "5.7.1", "domain": "gmail.com", "count": 30},
      {"code": "5.7.1", "domain": "outlook.com", "count": 25}
    ]
  }'

# Check the reputation
curl http://127.0.0.1:8080/api/ips/203.0.113.100/reputation | jq '.'

# View dashboard
curl http://127.0.0.1:8080/api/dashboard/ip-health | jq '.'
```

### Status Levels

1. **Healthy** - Normal operations (< 2% rejection)
2. **Warning** - Monitor closely (≥ 2% rejection)
3. **Quarantine** - High risk (≥ 3% rejection + major provider)
4. **Blacklisted** - Critical (≥ 5% rejection + 3 domains + 2 major providers)

### Features

- **Real-time Processing** - Webhooks from Stalwart mail server
- **Automated Detection** - Background aggregation every 5 minutes
- **DNSBL Checking** - 8 major blacklists (Spamhaus, Barracuda, etc.)
- **Comprehensive Metrics** - 9 Prometheus metrics for monitoring
- **Test Suite** - 15 comprehensive test cases covering all error codes

**Full Documentation:** See [guides/TECHNICAL_REFERENCE.md](guides/TECHNICAL_REFERENCE.md#ip-reputation-system)

## 📊 Monitoring

### Prometheus Metrics

Access metrics endpoint:
```bash
curl http://localhost:8080/metrics
```

**Core Metrics:**
- `http_requests_total` - Total HTTP requests by method, endpoint, and status
- `http_request_duration_seconds` - Request duration histogram
- Standard Go runtime metrics (CPU, memory, goroutines)

**IP Reputation Metrics:**
- `smtp_failures_total{ip, enhanced_code, domain}` - SMTP failures by IP
- `ip_status_changes_total{ip, from_status, to_status}` - Status transitions
- `ip_reputation_status{ip}` - Current IP status (gauge: 1-4)
- `ip_rejection_ratio` - Rejection ratio distribution
- `dnsbl_checks_total{ip, listed}` - DNSBL check results
- `dnsbl_check_duration_seconds` - DNSBL check performance
- `ip_aggregation_runs_total{status}` - Aggregation job stats
- `webhook_events_total{event_type, status}` - Webhook processing

### Logs

View structured JSON logs:
```bash
docker compose -f Context/Data/docker-compose.yml logs -f app
```

Log format:
```json
{
  "timestamp": "2025-11-15T21:05:14.806Z",
  "level": "info",
  "message": "SMTP failure recorded",
  "ip": "203.0.113.10",
  "smtp_code": 550,
  "enhanced_code": "5.7.1"
}
```

### Monitoring Stack

Start with full monitoring (Prometheus + Grafana + Loki):
```bash
./scripts/start-with-monitoring.sh
```

Access points:
- 📊 **Prometheus UI**: http://localhost:9090
- 📈 **Grafana**: http://localhost:3000 (admin/admin)
- 📝 **Loki**: http://localhost:3100

**Documentation:**
- **[Monitoring & Observability](guides/TECHNICAL_REFERENCE.md#monitoring--observability)** - Complete setup with Grafana

## 🐳 Docker Commands

**Note:** Use `docker compose` (space) not `docker-compose` (hyphen) for Docker Compose V2.

```bash
# Build and start
docker compose -f Context/Data/docker-compose.yml up --build

# Start in background
docker compose -f Context/Data/docker-compose.yml up -d

# View logs
docker compose -f Context/Data/docker-compose.yml logs -f app

# Stop
docker compose -f Context/Data/docker-compose.yml down

# Stop and remove volumes (fresh start)
docker compose -f Context/Data/docker-compose.yml down -v
```

## 🧪 Testing

### 🎨 **ONE-CLICK COMPLETE SYSTEM TEST** (Visual Dashboard)

**This is what you asked for - one function to test everything and visualize it!**

```bash
# Open the interactive test dashboard (works on macOS)
open web/test-dashboard.html

# Or on Linux
xdg-open web/test-dashboard.html

# Or on Windows
start web/test-dashboard.html
```

**What it does:**
- ✅ **ONE BUTTON** to run all 15 comprehensive IP reputation test cases
- ✅ Tests all error codes (5.7.1, 5.7.23, 5.7.25, 5.7.512, 5.7.606, etc.)
- ✅ Tests all 4 status levels (healthy, warning, quarantine, blacklisted)
- ✅ Tests edge cases (low volume, mixed signals, gradual decay)
- ✅ **Beautiful visual results** with pass/fail indicators
- ✅ **Real-time execution time** tracking for each test
- ✅ **Detailed metrics** showing rejection ratios, failure counts
- ✅ **Error messages** if tests fail with specific reasons
- ✅ **One-click access** to Swagger UI for deeper exploration
- ✅ **Run individual tests** independently for focused debugging

**This dashboard is your complete testing and visualization solution!**

### Quick Test Suite (Command Line)

```bash
# Test all core endpoints
./scripts/test-api.sh

# Test IP reputation system (15 test cases)
./scripts/test-ip-reputation.sh
```

### Test API Endpoints (NEW!)

All tests are now available as API endpoints in Swagger UI:

- **`GET /api/testing/test-cases`** - Get all test scenarios
- **`POST /api/testing/test-cases/{id}/run`** - Run a single test
- **`POST /api/testing/test-suite/run`** - Run all tests and get results

**Access in Swagger:** http://localhost:8080/swagger/index.html

### Manual Testing

```bash
# Health check
curl http://127.0.0.1:8080/health

# Get users
curl http://127.0.0.1:8080/users

# Create user
curl -X POST -H "Content-Type: application/json" \
  -d '{"username":"testuser","email":"test@example.com"}' \
  http://127.0.0.1:8080/users

# IP reputation dashboard
curl http://127.0.0.1:8080/api/dashboard/ip-health | jq '.'

# Run test suite via API
curl -X POST http://127.0.0.1:8080/api/testing/test-suite/run | jq '.'
```

**Documentation:**
- **[Testing Guide](guides/DEVELOPER_GUIDE.md#testing)** - Complete testing reference
- **[Getting Started](guides/DEVELOPER_GUIDE.md#getting-started)** - Interactive exploration

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

1. Add handler function in `internal/api/routes.go` or create new handler file
2. Add Swagger annotations
3. Register route in `SetupRoutes()`
4. Regenerate docs:
   ```bash
   swag init -g cmd/server/main.go -o docs
   ```

## 🔧 Configuration

### Environment Variables

All config values support environment variables with pattern: `${VARIABLE_NAME:default_value}`

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

**Logging:**
- `LOG_LEVEL` - Log level: debug, info, warn, error (default: info)
- `LOG_FORMAT` - Log format: json or console (default: json)

**IP Reputation (Optional):**
- `REPUTATION_WINDOW_MINUTES` - Time window for metrics (default: 15)
- `MIN_VOLUME_FOR_ASSESSMENT` - Minimum emails for assessment (default: 50)
- `AGGREGATION_INTERVAL_MINUTES` - Aggregation frequency (default: 5)
- `DNSBL_TIMEOUT_SECONDS` - DNSBL check timeout (default: 5)

## 🔒 Security Best Practices

- ✅ Secrets loaded from environment variables
- ✅ No hardcoded credentials
- ✅ Connection timeouts configured
- ✅ Graceful shutdown prevents data loss
- ✅ PostgreSQL SSL mode configurable
- ✅ SQL injection prevention (parameterized queries)
- ✅ Input validation on all endpoints

**Recommended for Production:**
- Add webhook authentication (token-based)
- Configure SSL/TLS
- Rate limiting on webhook endpoint
- Use secrets management (AWS Secrets Manager, Vault)

## 📈 Integration Ready

This service integrates with:
- **Prometheus/Grafana** - Built-in support
- **Loki/Promtail** - Log aggregation included
- **Stalwart Mail Server** - Webhook integration ready
- **Datadog** - Metrics and logs
- **EFK Stack** (Elasticsearch, Fluentd, Kibana)
- **AWS CloudWatch** - Cloud monitoring
- **Google Cloud Logging** - GCP integration

## 🐛 Troubleshooting

**Common Issues:**
- **[Troubleshooting Guide](guides/DEVELOPER_GUIDE.md#common-issues)** - Solutions to frequent problems

**Quick Checks:**
```bash
# Check if Docker is running
docker ps

# Check service logs
docker compose -f Context/Data/docker-compose.yml logs app

# Check database connection
docker compose -f Context/Data/docker-compose.yml logs db

# Verify health endpoint
curl http://127.0.0.1:8080/health
```

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
- [RFC 5321: SMTP](https://tools.ietf.org/html/rfc5321)
- [RFC 3463: Enhanced Status Codes](https://tools.ietf.org/html/rfc3463)

---

**For detailed documentation, see the [guides/](guides/) folder.**
