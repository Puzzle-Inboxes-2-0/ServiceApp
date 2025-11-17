# Developer Guide

**Complete guide for building, testing, and troubleshooting the GoLang Backend Service**

Last Updated: November 16, 2025

---

## Table of Contents

1. [Getting Started](#getting-started)
2. [Development Workflows](#development-workflows)
3. [Testing](#testing)
4. [Common Issues](#common-issues)
5. [Backend Development Patterns](#backend-development-patterns)
6. [API Development](#api-development)
7. [Database Operations](#database-operations)
8. [Best Practices](#best-practices)

---

## Getting Started

### Prerequisites

- **Docker Desktop** installed and running
- **Terminal** access
- **Web browser** (for Swagger UI)
- **Go 1.23+** (for local development)

### Quick Health Check

```bash
# Navigate to project directory
cd /Users/Mounir/Task-Master/Codebase/golang-backend-service

# Check Docker is running
./scripts/check-docker.sh
```

✅ If you see "Docker is ready", you're good to go!

---

### First Time Setup (3 minutes)

#### Step 1: Start the Service

```bash
./scripts/start-services.sh
```

Wait for these confirmations:
```
✓ Docker ready
✓ Database initialized
✓ Service started on http://localhost:8080
✓ Swagger UI available
```

#### Step 2: Verify It's Working

```bash
# Health check
curl http://localhost:8080/health

# Expected response:
# {"status":"healthy","timestamp":"2025-11-16T..."}
```

#### Step 3: Run API Tests

```bash
./scripts/test-api.sh
```

Expected: **✅ All tests passed!**

#### Step 4: Open Swagger UI

```bash
open http://localhost:8080/swagger/index.html
```

**Try it now:**
1. Click **GET /users** → **Try it out** → **Execute**
2. See the list of users
3. Click **POST /users** → **Try it out** → Edit the JSON → **Execute**
4. Create your first user!

🎉 **Core service is running!**

---

### IP Reputation System Quick Start

#### Step 1: Run IP Reputation Tests

```bash
./scripts/test-ip-reputation.sh
```

Expected: **✅ All 15 tests passed!**

#### Step 2: Simulate a Blacklisted IP

```bash
curl -X POST http://localhost:8080/api/testing/simulate-failures \
  -H "Content-Type: application/json" \
  -d '{
    "ip": "203.0.113.99",
    "total_sent": 500,
    "failures": [
      {"code": "5.7.1", "domain": "gmail.com", "count": 12, "reason": "IP reputation"},
      {"code": "5.7.1", "domain": "outlook.com", "count": 10, "reason": "Blocked"},
      {"code": "5.7.1", "domain": "yahoo.com", "count": 8, "reason": "Spam"}
    ]
  }'
```

#### Step 3: Check the Reputation

```bash
curl http://localhost:8080/api/ips/203.0.113.99/reputation | jq '.'
```

You'll see status: `blacklisted` with recommendations.

---

### Interactive Exploration

#### Explore Docker Desktop

1. **Open Docker Desktop**
2. **Click "Containers"** in the left sidebar
3. You'll see:
   - `data-app-1` (Go application) - Port 8080
   - `data-db-1` (PostgreSQL) - Port 5433

**Things to Try:**
- Click on `data-app-1` → **Logs** tab → See real-time JSON logs
- Click on `data-app-1` → **Stats** tab → Monitor CPU/Memory

#### Explore Swagger UI

**Open:** http://localhost:8080/swagger/index.html

1. **GET /health** - Check API health
2. **GET /users** - View all users
3. **POST /users** - Create a new user
4. **GET /users/{id}** - Get specific user

#### Explore the Database

```bash
# Connect to PostgreSQL
docker exec -it data-db-1 psql -U postgres -d mydb

# Try these commands:
SELECT * FROM users;
SELECT COUNT(*) FROM smtp_failures;
SELECT ip, status FROM ip_reputation_metrics;
\q
```

#### Explore Logs

```bash
# View pretty logs
./scripts/view-logs.sh

# Filter by level
./scripts/filter-logs.sh error
./scripts/filter-logs.sh info
```

---

### Essential Commands

```bash
# Service Management
./scripts/start-services.sh              # Start everything
./scripts/stop-services.sh               # Stop everything
docker restart data-app-1                # Restart app only

# Testing
./scripts/test-api.sh                    # Test core API
./scripts/test-ip-reputation.sh          # Test IP reputation

# Monitoring
./scripts/view-logs.sh                   # Pretty logs
./scripts/filter-logs.sh error           # Filter logs
curl http://localhost:8080/metrics       # Prometheus metrics
curl http://localhost:8080/health        # Health check

# Database
docker exec -it data-db-1 psql -U postgres -d mydb

# Development
docker compose -f Context/Data/docker-compose.yml up --build  # Rebuild
docker logs -f data-app-1                                     # Follow logs
```

---

## Development Workflows

### Adding a New Endpoint

#### 1. Define the Model

```go
// internal/database/posts.go

type Post struct {
    ID        int       `json:"id"`
    Title     string    `json:"title"`
    Content   string    `json:"content"`
    CreatedAt time.Time `json:"created_at"`
}
```

#### 2. Add Database Function

```go
// internal/database/posts.go

func GetPosts(db *sql.DB) ([]Post, error) {
    rows, err := db.Query("SELECT id, title, content, created_at FROM posts ORDER BY created_at DESC")
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var posts []Post
    for rows.Next() {
        var p Post
        if err := rows.Scan(&p.ID, &p.Title, &p.Content, &p.CreatedAt); err != nil {
            return nil, err
        }
        posts = append(posts, p)
    }
    return posts, rows.Err()
}

func CreatePost(db *sql.DB, post *Post) error {
    return db.QueryRow(
        "INSERT INTO posts (title, content) VALUES ($1, $2) RETURNING id, created_at",
        post.Title, post.Content,
    ).Scan(&post.ID, &post.CreatedAt)
}
```

#### 3. Create Handler

```go
// internal/api/handlers.go

// GetPosts godoc
// @Summary Get all posts
// @Description Retrieve list of all posts
// @Tags posts
// @Produce json
// @Success 200 {array} Post
// @Router /posts [get]
func getPostsHandler(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        posts, err := database.GetPosts(db)
        if err != nil {
            http.Error(w, "Failed to fetch posts", http.StatusInternalServerError)
            return
        }
        
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(posts)
    }
}

func createPostHandler(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var post Post
        if err := json.NewDecoder(r.Body).Decode(&post); err != nil {
            http.Error(w, "Invalid request body", http.StatusBadRequest)
            return
        }
        
        if post.Title == "" {
            http.Error(w, "Title is required", http.StatusBadRequest)
            return
        }
        
        if err := database.CreatePost(db, &post); err != nil {
            http.Error(w, "Failed to create post", http.StatusInternalServerError)
            return
        }
        
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusCreated)
        json.NewEncoder(w).Encode(post)
    }
}
```

#### 4. Register Route

```go
// internal/api/routes.go

router.HandleFunc("/posts", getPostsHandler(db)).Methods("GET")
router.HandleFunc("/posts", createPostHandler(db)).Methods("POST")
router.HandleFunc("/posts/{id}", getPostHandler(db)).Methods("GET")
```

#### 5. Update Swagger

```bash
# Regenerate Swagger docs
swag init -g cmd/server/main.go -o docs
```

#### 6. Add Database Migration

```sql
-- migrations/003_add_posts_table.sql

CREATE TABLE posts (
    id SERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    content TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_posts_created ON posts(created_at);
```

```bash
# Apply migration
docker exec -i data-db-1 psql -U postgres -d mydb < migrations/003_add_posts_table.sql
```

#### 7. Test

```bash
# Restart service
./scripts/stop-services.sh
./scripts/start-services.sh

# Test new endpoint
curl http://localhost:8080/posts | jq

# Create a post
curl -X POST http://localhost:8080/posts \
  -H "Content-Type: application/json" \
  -d '{"title":"My First Post","content":"Hello World"}' | jq

# Check Swagger
open http://localhost:8080/swagger/index.html
```

---

### Adding a Prometheus Metric

```go
// internal/metrics/metrics.go

var postsCreatedTotal = prometheus.NewCounter(
    prometheus.CounterOpts{
        Name: "posts_created_total",
        Help: "Total number of posts created",
    },
)

func init() {
    prometheus.MustRegister(postsCreatedTotal)
}

// In handler
func createPostHandler(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // ... create post logic ...
        postsCreatedTotal.Inc()
        // ... return response ...
    }
}
```

---

### Local Development (without Docker)

```bash
# Install dependencies
go mod download

# Set environment variables
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=postgres
export DB_PASSWORD=postgres
export DB_NAME=mydb
export SERVER_PORT=8080
export LOG_LEVEL=debug

# Run locally
go run cmd/server/main.go
```

---

## Testing

### Running Tests

#### Quick Testing (Recommended)

```bash
# Start services
./scripts/start-services.sh

# Run API tests
./scripts/test-api.sh

# Run IP reputation tests
./scripts/test-ip-reputation.sh

# Stop services
./scripts/stop-services.sh
```

---

### Manual Testing

#### Health Check

```bash
curl http://localhost:8080/health | jq
```

#### Create a User

```bash
curl -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{
    "username": "test_user",
    "email": "test@example.com"
  }' | jq
```

#### Get All Users

```bash
curl http://localhost:8080/users | jq
```

#### Get User by ID

```bash
curl http://localhost:8080/users/1 | jq
```

#### Check Metrics

```bash
curl http://localhost:8080/metrics
```

---

### Interactive Testing (Swagger UI)

**Open:** http://localhost:8080/swagger/index.html

Benefits:
- 🎯 Interactive API documentation
- ✅ Test endpoints directly in browser
- 📝 See request/response schemas
- 🔍 No need to remember curl commands

---

### Performance Testing

#### Using Apache Bench (comes with macOS)

```bash
# Test health endpoint (100 requests, 10 concurrent)
ab -n 100 -c 10 http://localhost:8080/health

# Test GET users
ab -n 1000 -c 50 http://localhost:8080/users
```

#### Using hey (modern load testing)

```bash
# Install hey
go install github.com/rakyll/hey@latest

# Run load test
hey -n 1000 -c 50 http://localhost:8080/users
```

---

### Test Error Handling

```bash
# Invalid user data (missing required field)
curl -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"username": "test"}' | jq

# Get non-existent user
curl http://localhost:8080/users/9999 | jq

# Duplicate username
curl -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"username": "john_doe", "email": "new@example.com"}' | jq
```

---

### Unit Testing (Go)

```go
// internal/reputation/decision_test.go

func TestDetermineIPStatus(t *testing.T) {
    config := DefaultReputationConfig()
    
    tests := []struct {
        name     string
        metrics  *IPHealthMetrics
        expected string
    }{
        {
            name: "healthy IP",
            metrics: &IPHealthMetrics{
                TotalSent:      500,
                TotalRejected:  2,
                RejectionRatio: 0.004,
            },
            expected: "healthy",
        },
        {
            name: "blacklisted IP",
            metrics: &IPHealthMetrics{
                TotalSent:               500,
                TotalRejected:           35,
                RejectionRatio:          0.07,
                UniqueDomainsRejected:   4,
                MajorProvidersRejecting: []string{"gmail.com", "outlook.com"},
            },
            expected: "blacklisted",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := determineIPStatus(tt.metrics, config)
            if result != tt.expected {
                t.Errorf("Expected %s, got %s", tt.expected, result)
            }
        })
    }
}
```

```bash
# Run unit tests
go test ./...

# With coverage
go test -cover ./...

# Verbose
go test -v ./...
```

---

## Common Issues

### Before You Start - Critical Checks

#### 1. Is Docker Actually Running?

```bash
./scripts/check-docker.sh
```

Should say: "Docker daemon is running"

If not:
- Open Docker Desktop app
- Wait for whale icon to become solid (not animated)
- Try again

#### 2. Are the Right Ports Free?

```bash
# Check if ports are in use
lsof -i :8080    # App
lsof -i :3000    # Grafana
lsof -i :9090    # Prometheus
lsof -i :5433    # PostgreSQL
```

If port is taken:
```bash
# Kill whatever is using it
kill -9 <PID>

# Or stop services first
./scripts/stop-services.sh
```

#### 3. Did You Actually Wait Long Enough?

⏱️ **TIMING MATTERS:**
- `./scripts/start-services.sh` takes **30-60 seconds** to fully start
- Don't open services immediately
- Wait for confirmation messages

---

### Service Won't Start

**Check Docker status:**
```bash
./scripts/check-docker.sh
```

**Check ports:**
```bash
lsof -i :8080
lsof -i :5433
```

**View logs:**
```bash
docker logs data-app-1
docker logs data-db-1
```

**Solution: Clean and restart**
```bash
docker compose -f Context/Data/docker-compose.yml down -v
./scripts/start-services.sh
```

---

### Database Connection Issues

**Test connection:**
```bash
docker exec -it data-db-1 psql -U postgres -d mydb -c "SELECT 1;"
```

**Check connection string:**
```bash
# Should be: postgres://user:password@host:port/database?sslmode=disable
echo $DB_HOST $DB_PORT $DB_USER $DB_NAME
```

**Common fix: Restart database**
```bash
docker restart data-db-1
```

---

### API Not Responding

**Check health:**
```bash
curl http://localhost:8080/health
```

**Check container status:**
```bash
docker ps | grep data-app-1
```

**Check logs for errors:**
```bash
./scripts/view-logs.sh
```

**Solution: Restart app**
```bash
docker restart data-app-1
```

---

### Tests Failing

**Problem: Service not running**

```bash
# Check health
curl http://localhost:8080/health

# Start if needed
./scripts/start-services.sh
```

**Problem: Database connection errors**

```bash
# Check database is running
docker ps | grep data-db-1

# Restart database
docker restart data-db-1
```

---

### Port Already in Use

**Find what's using the port:**
```bash
lsof -i :8080
```

**Kill it:**
```bash
kill -9 <PID>
```

**Or change port in config.yaml:**
```yaml
server:
  port: 8081  # Changed from 8080
```

---

### Can't Access Endpoints

**Problem: 404 errors**

Solution: Check Swagger for correct endpoints
```bash
open http://localhost:8080/swagger/index.html
```

**Problem: Connection refused**

```bash
# Verify service is running
docker ps

# Check logs
./scripts/view-logs.sh
```

---

### Metrics Not Showing

**Check metrics endpoint:**
```bash
curl http://localhost:8080/metrics
```

**Generate some traffic:**
```bash
./scripts/test-api.sh
```

**Check again:**
```bash
curl http://localhost:8080/metrics | grep "http_requests_total"
```

---

### Logs Not Appearing

**Check container is running:**
```bash
docker ps
```

**Try direct Docker logs:**
```bash
docker logs data-app-1
```

**Generate some logs:**
```bash
curl http://localhost:8080/users
curl http://localhost:8080/health
```

---

### New Code Changes Not Showing

**Rebuild the container:**
```bash
./scripts/stop-services.sh
./scripts/start-services.sh
# This rebuilds automatically
```

**Force rebuild:**
```bash
docker compose -f Context/Data/docker-compose.yml build --no-cache
./scripts/start-services.sh
```

---

### High CPU/Memory Usage

**Check container stats:**
```bash
docker stats
```

**Check what's using resources:**
```bash
docker stats --no-stream --format "table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}"
```

**Solution: Restart services**
```bash
./scripts/stop-services.sh
./scripts/start-services.sh
```

---

### Need a Fresh Start

**Complete reset (removes all data!):**
```bash
# Stop everything
./scripts/stop-services.sh

# Remove containers and volumes
docker compose -f Context/Data/docker-compose.yml down -v

# Clean Docker
docker system prune -f

# Start fresh
./scripts/start-services.sh
```

---

## Backend Development Patterns

### Error Handling

**DO:**
```go
// Return errors, don't panic
func GetUser(db *sql.DB, id int) (*User, error) {
    var user User
    err := db.QueryRow("SELECT * FROM users WHERE id = $1", id).Scan(...)
    if err == sql.ErrNoRows {
        return nil, fmt.Errorf("user not found: %d", id)
    }
    if err != nil {
        return nil, fmt.Errorf("database error: %w", err)
    }
    return &user, nil
}
```

**DON'T:**
```go
// Don't panic in library code
func GetUser(db *sql.DB, id int) *User {
    var user User
    err := db.QueryRow("SELECT * FROM users WHERE id = $1", id).Scan(...)
    if err != nil {
        panic(err) // ❌ BAD
    }
    return &user
}
```

---

### Logging Best Practices

**DO:**
```go
// Use structured fields
logger.WithFields(logrus.Fields{
    "ip":              "203.0.113.10",
    "status":          "blacklisted",
    "rejection_ratio": 0.07,
}).Warn("IP reputation degraded")
```

**DON'T:**
```go
// Don't use string concatenation
logger.Info("IP " + ip + " status is " + status) // ❌ BAD

// Don't log sensitive data
logger.Info("User password: " + password) // ❌ VERY BAD
```

---

### Database Best Practices

**DO:**
```go
// Use parameterized queries (prevent SQL injection)
db.Query("SELECT * FROM users WHERE username = $1", username)

// Close resources
defer rows.Close()
defer stmt.Close()

// Use transactions for multiple operations
tx, _ := db.Begin()
defer tx.Rollback()
// ... operations ...
tx.Commit()
```

**DON'T:**
```go
// Don't concatenate SQL strings
query := "SELECT * FROM users WHERE username = '" + username + "'" // ❌ SQL INJECTION
db.Query(query)
```

---

### Concurrency Patterns

**Worker Pool Pattern:**
```go
func processIPsWithWorkerPool(ips []string, numWorkers int) {
    jobs := make(chan string, len(ips))
    results := make(chan Result, len(ips))
    
    // Start workers
    for w := 0; w < numWorkers; w++ {
        go func() {
            for ip := range jobs {
                results <- processIP(ip)
            }
        }()
    }
    
    // Send jobs
    for _, ip := range ips {
        jobs <- ip
    }
    close(jobs)
    
    // Collect results
    for range ips {
        <-results
    }
}
```

---

### Middleware Pattern

```go
func loggingMiddleware(logger *logrus.Logger) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            start := time.Now()
            
            // Call next handler
            next.ServeHTTP(w, r)
            
            // Log request
            logger.WithFields(logrus.Fields{
                "method":   r.Method,
                "path":     r.URL.Path,
                "duration": time.Since(start),
            }).Info("HTTP request")
        })
    }
}

// Usage
router.Use(loggingMiddleware(logger))
```

---

## API Development

### RESTful Conventions

**Resource naming:**
- **Plural nouns**: `/users` not `/user`
- **Hierarchical**: `/users/{id}/posts`
- **Lowercase**: `/api/ips` not `/API/IPs`

**HTTP methods:**
- **GET**: Read (idempotent, safe)
- **POST**: Create
- **PUT**: Update (full replacement)
- **PATCH**: Update (partial)
- **DELETE**: Remove

**Status codes:**
- **200**: Success (GET, PUT, PATCH)
- **201**: Created (POST)
- **204**: No Content (DELETE)
- **400**: Bad Request (validation error)
- **404**: Not Found
- **500**: Internal Server Error

---

### Handler Pattern

```go
func createUserHandler(db *sql.DB, logger *logrus.Logger) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // 1. Parse request
        var user User
        if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
            logger.WithError(err).Error("Failed to parse request")
            http.Error(w, "Invalid request body", http.StatusBadRequest)
            return
        }
        
        // 2. Validate
        if user.Username == "" || user.Email == "" {
            http.Error(w, "Username and email required", http.StatusBadRequest)
            return
        }
        
        // 3. Business logic
        if err := database.CreateUser(db, &user); err != nil {
            logger.WithError(err).Error("Failed to create user")
            http.Error(w, "Failed to create user", http.StatusInternalServerError)
            return
        }
        
        // 4. Log success
        logger.WithFields(logrus.Fields{
            "user_id":  user.ID,
            "username": user.Username,
        }).Info("User created")
        
        // 5. Return response
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusCreated)
        json.NewEncoder(w).Encode(user)
    }
}
```

---

### Input Validation

```go
func validateEmail(email string) error {
    re := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
    if !re.MatchString(email) {
        return fmt.Errorf("invalid email format")
    }
    return nil
}

func validateUsername(username string) error {
    if len(username) < 3 {
        return fmt.Errorf("username must be at least 3 characters")
    }
    if len(username) > 50 {
        return fmt.Errorf("username must be less than 50 characters")
    }
    return nil
}
```

---

## Database Operations

### Connection Pool

```go
func Connect(dsn string) (*sql.DB, error) {
    db, err := sql.Open("postgres", dsn)
    if err != nil {
        return nil, err
    }
    
    // Configure connection pool
    db.SetMaxOpenConns(25)
    db.SetMaxIdleConns(5)
    db.SetConnMaxLifetime(5 * time.Minute)
    
    // Verify connection
    if err := db.Ping(); err != nil {
        return nil, err
    }
    
    return db, nil
}
```

---

### CRUD Patterns

**Query Many:**
```go
func GetUsers(db *sql.DB) ([]User, error) {
    rows, err := db.Query("SELECT id, username, email, created_at FROM users")
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var users []User
    for rows.Next() {
        var u User
        if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.CreatedAt); err != nil {
            return nil, err
        }
        users = append(users, u)
    }
    return users, rows.Err()
}
```

**Query One:**
```go
func GetUserByID(db *sql.DB, id int) (*User, error) {
    var user User
    err := db.QueryRow(
        "SELECT id, username, email, created_at FROM users WHERE id = $1",
        id,
    ).Scan(&user.ID, &user.Username, &user.Email, &user.CreatedAt)
    
    if err == sql.ErrNoRows {
        return nil, fmt.Errorf("user not found")
    }
    if err != nil {
        return nil, err
    }
    return &user, nil
}
```

**Insert:**
```go
func CreateUser(db *sql.DB, user *User) error {
    return db.QueryRow(
        "INSERT INTO users (username, email) VALUES ($1, $2) RETURNING id, created_at",
        user.Username, user.Email,
    ).Scan(&user.ID, &user.CreatedAt)
}
```

**Update:**
```go
func UpdateUser(db *sql.DB, user *User) error {
    _, err := db.Exec(
        "UPDATE users SET username = $1, email = $2 WHERE id = $3",
        user.Username, user.Email, user.ID,
    )
    return err
}
```

**Delete:**
```go
func DeleteUser(db *sql.DB, id int) error {
    _, err := db.Exec("DELETE FROM users WHERE id = $1", id)
    return err
}
```

---

### Transactions

```go
func transferFunds(db *sql.DB, fromID, toID int, amount float64) error {
    tx, err := db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()
    
    // Deduct from sender
    _, err = tx.Exec("UPDATE accounts SET balance = balance - $1 WHERE id = $2", amount, fromID)
    if err != nil {
        return err
    }
    
    // Add to receiver
    _, err = tx.Exec("UPDATE accounts SET balance = balance + $1 WHERE id = $2", amount, toID)
    if err != nil {
        return err
    }
    
    return tx.Commit()
}
```

---

## Best Practices

### Security

**✅ DO:**
- Use parameterized queries (prevent SQL injection)
- Validate all input
- Use environment variables for secrets
- Set connection timeouts
- Use HTTPS in production
- Implement rate limiting
- Log security events

**❌ DON'T:**
- Concatenate SQL strings
- Trust user input
- Hardcode credentials
- Expose sensitive data in errors
- Ignore authentication

---

### Performance

**Database:**
- Use connection pooling
- Add indexes for WHERE clauses
- Limit result sets
- Avoid N+1 queries
- Use EXPLAIN to analyze queries

**API:**
- Cache responses where appropriate
- Use compression (gzip)
- Implement pagination
- Set appropriate timeouts
- Monitor response times

---

### Code Quality

**Follow Go conventions:**
- Run `gofmt` before committing
- Use meaningful variable names
- Write clear comments
- Keep functions small and focused
- Handle all errors
- Write tests

**Documentation:**
- Add Swagger annotations to handlers
- Update README when adding features
- Document complex algorithms
- Keep documentation up-to-date

---

## Tips for Success

### 1. Use Swagger UI
It's the fastest way to test and understand your API. No need to write curl commands manually!

### 2. Keep Logs Open
Open logs in one terminal, test in another. You'll see exactly what's happening in real-time:
```bash
# Terminal 1
./scripts/view-logs.sh

# Terminal 2
curl http://localhost:8080/users
```

### 3. Don't Be Afraid to Break Things
You can always restart! Experimentation is learning.

### 4. Check the Code
When you see something interesting in Swagger or logs, find it in the code to understand how it works.

### 5. Use Docker Desktop
The GUI makes it easy to start/stop containers, view logs, and monitor resources.

---

## Next Steps

Once you're comfortable with the basics:

1. **Read Technical Reference**
   - Complete architecture
   - IP reputation details
   - Monitoring setup

2. **Explore Advanced Features**
   - Modify decision algorithm thresholds
   - Add custom metrics
   - Integrate with external services

3. **Build Something**
   - Add a new endpoint
   - Create a new feature
   - Optimize performance

---

**Happy coding! 🚀**

Last Updated: November 16, 2025

