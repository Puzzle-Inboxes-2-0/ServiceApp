# 🚀 HOW TO RUN THE IP REPUTATION SYSTEM

## ⚡ FASTEST WAY (One Command)

```bash
./START_EVERYTHING.sh
```

**This will:**
1. Check if Docker is running (start it if not)
2. Start the backend (Go app + PostgreSQL)
3. Start the web server for the test dashboard
4. Open the test dashboard in your browser

---

## 🛑 TO STOP EVERYTHING

```bash
./STOP_EVERYTHING.sh
```

---

## 📋 MANUAL STEP-BY-STEP (If you want to do it yourself)

### Step 1: Make sure Docker Desktop is running

```bash
open -a Docker
# Wait 30 seconds for it to start
```

### Step 2: Start the backend

```bash
cd /Users/Mounir/Task-Master/Codebase/golang-backend-service
docker compose -f Context/Data/docker-compose.yml up --build -d
```

**Wait about 10 seconds**, then verify it's working:

```bash
curl http://127.0.0.1:8080/health
```

You should see: `{"status":"healthy","timestamp":"..."}`

### Step 3: Start web server for dashboard (in a new terminal)

```bash
cd /Users/Mounir/Task-Master/Codebase/golang-backend-service/web
python3 -m http.server 8888
```

### Step 4: Open test dashboard

```bash
open http://localhost:8888/test-dashboard.html
```

### Step 5: Run tests

In the dashboard that opens:
1. Click **"Run All Tests"** button
2. Watch the results appear with pass/fail indicators

---

## 🧪 COMMAND-LINE TESTING

### Run the full test suite

```bash
./scripts/test-ip-reputation.sh
```

### Test a specific IP

```bash
curl -X POST http://127.0.0.1:8080/api/testing/simulate-failures \
  -H "Content-Type: application/json" \
  -d '{
    "ip": "203.0.113.100",
    "total_sent": 500,
    "failures": [
      {"code": "5.7.1", "domain": "gmail.com", "count": 30}
    ]
  }' | jq
```

### Check any IP reputation

```bash
curl http://127.0.0.1:8080/api/ips/203.0.113.100/reputation | jq
```

### View all IPs

```bash
curl http://127.0.0.1:8080/api/dashboard/ip-health | jq
```

---

## 🌐 AVAILABLE SERVICES

Once started, these URLs are available:

| Service | URL | Description |
|---------|-----|-------------|
| **Test Dashboard** | http://localhost:8888/test-dashboard.html | Visual testing interface |
| **Backend API** | http://localhost:8080 | Main API |
| **Health Check** | http://localhost:8080/health | Service health |
| **Swagger UI** | http://localhost:8080/swagger/index.html | Interactive API docs |
| **Prometheus** | http://localhost:9090 | Metrics |
| **Grafana** | http://localhost:3000 | Dashboards (admin/admin) |

---

## 🐛 TROUBLESHOOTING

### "Docker is not running"

```bash
open -a Docker
# Wait 30 seconds, then try again
```

### "Connection refused" on port 8080

Wait 10 seconds for the backend to fully start, then try again:

```bash
curl http://127.0.0.1:8080/health
```

### Dashboard shows "Failed to fetch"

Make sure you're accessing it through the web server (not as a file):
- ✅ CORRECT: `http://localhost:8888/test-dashboard.html`
- ❌ WRONG: `file:///Users/...`

If web server isn't running:

```bash
cd /Users/Mounir/Task-Master/Codebase/golang-backend-service/web
python3 -m http.server 8888
```

### Check if containers are running

```bash
docker ps
```

You should see:
- `data-app-1` (Go backend)
- `data-db-1` (PostgreSQL)
- Plus monitoring stack (prometheus, grafana, etc.)

### View logs

```bash
docker logs -f data-app-1
```

---

## 🔄 FRESH START (Clean Everything)

If you want to start completely fresh (wipe database):

```bash
./STOP_EVERYTHING.sh
docker compose -f Context/Data/docker-compose.yml down -v  # -v removes volumes
./START_EVERYTHING.sh
```

---

## 📚 MORE DOCUMENTATION

- **Quick Answers**: `QUICK_ANSWERS.md`
- **Complete Analysis**: `docs/CRITICAL_FINDINGS_AND_ANSWERS.md`
- **Error Code Reference**: `docs/SMTP_ERROR_CODES_REFERENCE.md`
- **Architecture Diagrams**: `docs/SYSTEM_ARCHITECTURE_DIAGRAM.md`
- **Full README**: `README.md`


