# IONOS IP Reservation - Quick Start (Token Configured ✅)

Your IONOS token has been configured! Here's how to start using the service:

## Your Token (Stored Securely)

Your IONOS API token is stored in:
- `scripts/setup-env.sh` (NOT committed to git)

**Token**: `eyJ0eXAiOiJKV1QiLCJraWQiOiIxNWJjZWNjMC1iYTg4LTRlMWItYWFhYy0zMWIxMDQ3MTgyNDEiLCJhbGciOiJSUzI1NiJ9`

## Option 1: Use the Quick Start Script (Easiest) 🚀

```bash
cd /Users/Mounir/Task-Master/Codebase/golang-backend-service

# This script will:
# 1. Load your IONOS token
# 2. Check database connection
# 3. Start the service
./START_SERVICE.sh
```

## Option 2: Manual Setup

### Step 1: Load Environment Variables

```bash
cd /Users/Mounir/Task-Master/Codebase/golang-backend-service

# Load the token and database config
source scripts/setup-env.sh
```

### Step 2: Start the Service

```bash
go run cmd/server/main.go
```

You should see:
```
INFO[0000] Starting GoLang Backend Service
INFO[0000] IONOS service initialized
INFO[0000] Service is ready
```

### Step 3: Test in Another Terminal

Open a new terminal and run:

```bash
# Check quota
./scripts/check-quota.sh

# Reserve 3 test IPs (this will actually reserve from IONOS!)
./scripts/reserve-ips.sh 3 us/ewr

# List reserved IPs
./scripts/list-reserved-ips.sh
```

## What Happens When You Reserve IPs

1. **IONOS API Call**: Creates a new IP block in your IONOS account
2. **Blacklist Check**: Checks the IP against 10+ DNS blacklists (takes ~7 seconds)
3. **Decision**:
   - ✅ **Clean IP**: Stored in database with status "reserved"
   - ❌ **Blacklisted IP**: Deleted from IONOS immediately, tries again
4. **Database Record**: IP stored with UID, location, timestamp

## API Endpoints Available

Once the service is running on `http://localhost:8080`:

```bash
# Reserve IPs
curl -X POST http://localhost:8080/api/v1/ips/reserve \
  -H "Content-Type: application/json" \
  -d '{"count": 5, "location": "us/ewr"}'

# List reserved IPs
curl http://localhost:8080/api/v1/ips/reserved

# Check quota
curl http://localhost:8080/api/v1/ips/quota

# Get statistics
curl http://localhost:8080/api/v1/ips/statistics
```

## Important Notes

### Security ⚠️

- ✅ Token is stored in `scripts/setup-env.sh` (NOT committed to git)
- ✅ `.gitignore` configured to exclude sensitive files
- ⚠️ **Never commit the token to git!**
- 🔒 Keep `scripts/setup-env.sh` secure and private

### Database

If you haven't started the database yet:

```bash
cd Context/Data
docker-compose up -d

# Wait a few seconds for it to initialize, then check:
psql -h localhost -p 5432 -U postgres -d mydb -c "SELECT 1"
# Password: changeme123
```

The database schema (with IP tables) will be created automatically.

### Cost Warning 💰

**Each IP reservation costs money!** 
- Single IP blocks are charged by IONOS
- Use the cleanup script to remove unused IPs:
  ```bash
  ./scripts/cleanup-unused-ips.sh
  ```
- Check quota before reserving large batches

## Troubleshooting

### "IONOS service not initialized"
- Make sure you ran `source scripts/setup-env.sh` first
- Verify token is set: `echo $IONOS_TOKEN`

### "Cannot connect to database"
- Start database: `cd Context/Data && docker-compose up -d`
- Check it's running: `docker ps`
- Test connection: `psql -h localhost -p 5432 -U postgres -d mydb`

### "Failed to reserve IP block: 401"
- Your IONOS token may be expired or invalid
- Get a new token from IONOS Cloud dashboard
- Update `scripts/setup-env.sh`

### "Failed to reserve IP block: 422"
- You've hit your IONOS quota limit
- Run cleanup: `./scripts/cleanup-unused-ips.sh`
- Check quota: `./scripts/check-quota.sh`
- Or contact IONOS to increase quota

## Next Steps

1. **Test with 1 IP first**: `./scripts/reserve-ips.sh 1 us/ewr`
2. **Verify it worked**: `./scripts/list-reserved-ips.sh`
3. **Check the database**: 
   ```sql
   psql -h localhost -p 5432 -U postgres -d mydb
   SELECT * FROM reserved_ips;
   ```
4. **Reserve your full batch**: `./scripts/reserve-ips.sh 11 us/ewr`
5. **Monitor usage**: `curl http://localhost:8080/api/v1/ips/statistics | jq`

## Complete Example Session

```bash
# Terminal 1: Start the service
cd /Users/Mounir/Task-Master/Codebase/golang-backend-service
./START_SERVICE.sh

# Terminal 2: Use the service
cd /Users/Mounir/Task-Master/Codebase/golang-backend-service

# Check quota before starting
./scripts/check-quota.sh

# Reserve 11 clean IPs (may take 2-5 minutes)
./scripts/reserve-ips.sh 11 us/ewr

# List what we got
./scripts/list-reserved-ips.sh

# Check statistics
curl http://localhost:8080/api/v1/ips/statistics | jq

# Mark an IP as in-use
curl -X PUT http://localhost:8080/api/v1/ips/reserved/1/status \
  -H "Content-Type: application/json" \
  -d '{"status": "in_use", "assigned_to": "smtp-service-1"}'

# Cleanup unused IPs later
./scripts/cleanup-unused-ips.sh
```

## Status

- ✅ Token configured
- ✅ Scripts ready
- ✅ Service ready to start
- 🎯 **Ready to reserve IPs!**

---

**Remember**: The service is production-ready but will make real API calls to IONOS and reserve actual IPs (which cost money). Start with small tests (1-3 IPs) to verify everything works before reserving larger batches.

