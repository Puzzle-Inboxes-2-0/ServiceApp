# IONOS IP Reservation - Quick Start Guide

## Setup (5 minutes)

### 1. Set Environment Variables

```bash
export IONOS_TOKEN="your_ionos_api_token_here"
```

### 2. Update Database

The database schema will be automatically created when you start the service. If you need to manually initialize:

```bash
cd /Users/Mounir/Task-Master/Codebase/golang-backend-service/Context/Data
docker-compose up -d
psql -h localhost -p 5432 -U postgres -d mydb -f init.sql
```

### 3. Start the Service

```bash
cd /Users/Mounir/Task-Master/Codebase/golang-backend-service
go run cmd/server/main.go
```

## Common Operations

### Reserve 11 Clean IPs

```bash
# Using the script
./scripts/reserve-ips.sh 11 us/ewr

# Or using curl
curl -X POST http://localhost:8080/api/v1/ips/reserve \
  -H "Content-Type: application/json" \
  -d '{"count": 11, "location": "us/ewr"}'
```

### List All Reserved IPs

```bash
./scripts/list-reserved-ips.sh

# Or
curl http://localhost:8080/api/v1/ips/reserved
```

### Check Quota

```bash
./scripts/check-quota.sh

# Or
curl http://localhost:8080/api/v1/ips/quota
```

### Mark IP as In-Use

```bash
curl -X PUT http://localhost:8080/api/v1/ips/reserved/1/status \
  -H "Content-Type: application/json" \
  -d '{"status": "in_use", "assigned_to": "smtp-service-1"}'
```

### Cleanup Unused IPs

```bash
./scripts/cleanup-unused-ips.sh

# Or
curl -X POST http://localhost:8080/api/v1/ips/cleanup
```

## API Endpoints at a Glance

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/ips/reserve` | Reserve clean IPs |
| GET | `/api/v1/ips/reserved` | List reserved IPs |
| GET | `/api/v1/ips/reserved/{id}` | Get specific IP |
| PUT | `/api/v1/ips/reserved/{id}/status` | Update IP status |
| POST | `/api/v1/ips/reserved/{id}/recheck` | Recheck blacklist |
| DELETE | `/api/v1/ips/reserved/{id}` | Delete IP |
| GET | `/api/v1/ips/quota` | Check quota |
| POST | `/api/v1/ips/cleanup` | Cleanup unused blocks |
| GET | `/api/v1/ips/statistics` | Get statistics |

## IP Status Values

- **reserved**: IP is reserved but not in use
- **in_use**: IP is actively being used
- **released**: IP has been released back to pool  
- **quarantined**: IP has been flagged for issues

## Database Queries

### Find Available Clean IPs

```sql
SELECT ip_address, location, reserved_at 
FROM reserved_ips 
WHERE status = 'reserved' 
  AND is_blacklisted = false 
ORDER BY reserved_at DESC;
```

### Check Blacklist History

```sql
SELECT ri.ip_address, rh.checked_at, rh.was_blacklisted, rh.blacklists_found
FROM reserved_ip_blacklist_history rh
JOIN reserved_ips ri ON ri.id = rh.reserved_ip_id
WHERE ri.ip_address = '66.179.255.235'
ORDER BY rh.checked_at DESC;
```

### View Reservation Success Rate

```sql
SELECT * FROM reservation_success_metrics 
ORDER BY date DESC 
LIMIT 7;
```

## Troubleshooting

### Service Not Starting
```bash
# Check if IONOS_TOKEN is set
echo $IONOS_TOKEN

# Check database connection
psql -h localhost -p 5432 -U postgres -d mydb -c "SELECT 1"

# Check logs
tail -f logs/app.log
```

### IPs Always Blacklisted
```bash
# Try different location
./scripts/reserve-ips.sh 5 us/las

# Check which blacklists are failing
curl http://localhost:8080/api/v1/ips/reserved/{id} | jq '.blacklist_details'
```

### Quota Exhausted
```bash
# Cleanup unused blocks
./scripts/cleanup-unused-ips.sh

# Check quota
./scripts/check-quota.sh
```

## Production Checklist

- [ ] Set `IONOS_TOKEN` in production environment
- [ ] Configure database with production credentials
- [ ] Set up monitoring alerts for quota usage
- [ ] Set up alerts for blacklisted IPs
- [ ] Schedule daily blacklist rechecks
- [ ] Schedule weekly cleanup jobs
- [ ] Configure backup strategy for reservation data
- [ ] Test failover scenarios (IONOS API down)
- [ ] Document IP rotation procedures
- [ ] Set up logging aggregation

## Next Steps

1. Read full documentation: [IP_RESERVATION_SYSTEM.md](./IP_RESERVATION_SYSTEM.md)
2. Set up monitoring dashboards
3. Integrate with SMTP sending service
4. Configure automated IP rotation
5. Set up alerting rules

