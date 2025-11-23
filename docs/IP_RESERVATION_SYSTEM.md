# IONOS IP Reservation System

## Overview

The IONOS IP Reservation System is a microservice integrated into the golang-backend-service that automates the reservation, validation, and management of IP addresses from IONOS Cloud. It ensures that all reserved IPs are clean (not blacklisted) before being stored in the database for use in email sending operations.

## Features

### Core Functionality
- **Automated IP Reservation**: Reserve IP blocks from IONOS Cloud API
- **Blacklist Validation**: Check IPs against 10+ DNS-based blacklists (DNSBL)
- **Lifecycle Management**: Track IP status (reserved, in_use, released, quarantined)
- **Quota Management**: Monitor and manage IONOS quota usage
- **Auto-Cleanup**: Remove unused single-IP blocks while protecting 11-IP blocks
- **Historical Tracking**: Maintain history of blacklist checks and reservation attempts

### Database Tables

#### `reserved_ips`
Stores all reserved IP addresses with their current status and metadata.

**Key Fields:**
- `ip_address` (INET): The reserved IP address
- `reservation_block_id` (VARCHAR): IONOS block ID
- `uid` (VARCHAR): Unique identifier for the reservation
- `location` (VARCHAR): Datacenter location (e.g., us/ewr, us/las)
- `status` (VARCHAR): Current status (reserved, in_use, released, quarantined)
- `is_blacklisted` (BOOLEAN): Current blacklist status
- `blacklist_details` (JSONB): List of blacklists the IP is on
- `assigned_to` (VARCHAR): Service or user using this IP
- `usage_count` (INTEGER): Number of times IP has been used

#### `ip_reservation_attempts`
Tracks all reservation attempts for auditing and analytics.

**Key Fields:**
- `attempt_uid` (VARCHAR): Unique attempt identifier
- `success` (BOOLEAN): Whether the reservation succeeded
- `was_blacklisted` (BOOLEAN): Whether the IP was blacklisted
- `action_taken` (VARCHAR): Action taken (kept, deleted, quarantined)

#### `reserved_ip_blacklist_history`
Maintains a history of all blacklist checks performed on reserved IPs.

#### `ionos_quota_snapshots`
Records quota usage over time for capacity planning.

## API Endpoints

### Reserve IPs
```http
POST /api/v1/ips/reserve
Content-Type: application/json

{
  "count": 11,
  "location": "us/ewr"
}
```

**Response:**
```json
{
  "success_count": 11,
  "failure_count": 0,
  "blacklisted_count": 3,
  "reserved_ips": [
    {
      "id": 1,
      "ip_address": "66.179.255.235",
      "uid": "f5e6d2e6",
      "status": "reserved",
      "is_blacklisted": false,
      "location": "us/ewr"
    }
  ]
}
```

### List Reserved IPs
```http
GET /api/v1/ips/reserved?status=reserved&blacklisted=false&location=us/ewr
```

### Get Reserved IP
```http
GET /api/v1/ips/reserved/{id}
```

### Update IP Status
```http
PUT /api/v1/ips/reserved/{id}/status
Content-Type: application/json

{
  "status": "in_use",
  "assigned_to": "smtp-service-1"
}
```

**Valid Statuses:**
- `reserved`: IP is reserved but not in use
- `in_use`: IP is actively being used
- `released`: IP has been released back to pool
- `quarantined`: IP has been flagged for issues

### Recheck Blacklist
```http
POST /api/v1/ips/reserved/{id}/recheck
```

Forces a fresh blacklist check and updates the IP's status.

### Delete Reserved IP
```http
DELETE /api/v1/ips/reserved/{id}
```

Deletes the IP from database and optionally from IONOS.

### Check Quota
```http
GET /api/v1/ips/quota
```

**Response:**
```json
{
  "total_blocks": 25,
  "protected_blocks": 3,
  "single_ip_blocks": 22,
  "estimated_limit": 50,
  "remaining": 25
}
```

### Cleanup Unused Blocks
```http
POST /api/v1/ips/cleanup
```

Removes all single-IP blocks not currently in use. **Never touches 11-IP blocks.**

### Get Statistics
```http
GET /api/v1/ips/statistics
```

## Configuration

### Environment Variables

```bash
# Required
export IONOS_TOKEN="your_ionos_api_token"

# Optional (with defaults)
export IONOS_API_URL="https://api.ionos.com/cloudapi/v6"
export IONOS_DEFAULT_LOCATION="us/ewr"
export IONOS_DEFAULT_SIZE="1"
export IONOS_MAX_QUOTA="50"
export IONOS_RESERVATION_TIMEOUT="30s"
```

### config.yaml

```yaml
ionos:
  token: ${IONOS_TOKEN:}
  api_url: ${IONOS_API_URL:https://api.ionos.com/cloudapi/v6}
  default_location: ${IONOS_DEFAULT_LOCATION:us/ewr}
  default_reservation_size: ${IONOS_DEFAULT_SIZE:1}
  max_quota: ${IONOS_MAX_QUOTA:50}
  reservation_timeout: ${IONOS_RESERVATION_TIMEOUT:30s}
```

## DNSBL Configuration

### Monitored Blacklists

The system checks against the following DNS-based blacklists:

1. **Spamhaus** (zen.spamhaus.org)
2. **Barracuda** (b.barracudacentral.org)
3. **Spamcop** (bl.spamcop.net)
4. **Abuseat CBL** (cbl.abuseat.org)
5. **SpamRats** (dyna, noptr, spam)
6. **Manitu** (ix.dnsbl.manitu.net)
7. **SORBS** (dnsbl.sorbs.net)
8. **SURRIEL** (psbl.surriel.com)
9. **UBL** (ubl.unsubscore.com)
10. **DroneBL** (dnsbl.dronebl.org)

### Ignored Blacklists

Per user requirements, the following blacklists are **ignored** (not checked or results discarded):

- **UCEPROTECT** (all levels) - User accepts these as clean
- **Invaluement** - Known for false positives

## Usage Scripts

### Reserve IPs
```bash
cd /Users/Mounir/Task-Master/Codebase/golang-backend-service
./scripts/reserve-ips.sh 11 us/ewr
```

### Check Quota
```bash
./scripts/check-quota.sh
```

### List Reserved IPs
```bash
./scripts/list-reserved-ips.sh
./scripts/list-reserved-ips.sh reserved     # Filter by status
./scripts/list-reserved-ips.sh reserved false  # Non-blacklisted reserved IPs
```

### Cleanup Unused Blocks
```bash
./scripts/cleanup-unused-ips.sh
```

## Operational Considerations

### Second-Order Consequences

1. **Quota Management**
   - IONOS has a limit on IP blocks (typically 50 per account)
   - Single-IP blocks count against quota
   - 11-IP blocks (protected) also count
   - Monitor quota regularly to avoid hitting limits

2. **Blacklist Timing**
   - IPs can become blacklisted after reservation
   - Implement periodic rechecking (daily/weekly)
   - Consider TTL for clean IPs (e.g., recheck every 7 days)

3. **Cost Implications**
   - Each reserved IP block has a cost
   - Unused IPs should be released promptly
   - Balance between pool size and cost

### Third-Order Consequences

1. **Integration with SMTP Service**
   - Reserved IPs should be automatically provisioned to sending infrastructure
   - When an IP is blacklisted, immediately rotate to a clean IP
   - Track which emails were sent from which IPs for troubleshooting

2. **Monitoring and Alerting**
   - Alert when quota usage > 80%
   - Alert when blacklist rate > 20%
   - Alert when an in-use IP becomes blacklisted
   - Track success rate trends

3. **Historical Analytics**
   - Analyze which subnets/locations have higher blacklist rates
   - Identify patterns in blacklist timing
   - Optimize reservation strategies based on data

### Fourth-Order Consequences

1. **Multi-Region Strategy**
   - Different regions may have different blacklist profiles
   - Newark (us/ewr) and Las Vegas (us/las) have different IP ranges
   - Consider region-specific IP pools

2. **Automated Remediation**
   - Auto-release IPs that remain blacklisted after X days
   - Auto-reserve new IPs when pool drops below threshold
   - Auto-quarantine IPs with repeated blacklist issues

3. **Compliance and Audit**
   - All reservation attempts are logged
   - Full blacklist check history maintained
   - IP usage can be traced to specific sends
   - Supports compliance requirements for email operations

4. **Disaster Recovery**
   - What happens if IONOS API is unavailable?
   - Fallback to existing clean IPs
   - Queue reservation requests for retry
   - Alert operations team

## Best Practices

### IP Lifecycle

1. **Reservation** → Check blacklists → Store in database
2. **Provisioning** → Update status to "in_use", assign to service
3. **Monitoring** → Periodic blacklist rechecks
4. **Rotation** → Release and replace if blacklisted
5. **Cleanup** → Delete from IONOS when no longer needed

### Maintenance Tasks

**Daily:**
- Check quota usage
- Review newly blacklisted IPs
- Monitor success rates

**Weekly:**
- Recheck all in-use IPs
- Cleanup released IPs older than 7 days
- Review reservation statistics

**Monthly:**
- Analyze blacklist patterns
- Optimize location strategy
- Review and adjust quota limits

### Safety Measures

1. **Protected Blocks**: 11-IP blocks are NEVER deleted
2. **In-Use Protection**: IPs marked as "in_use" are not deleted during cleanup
3. **Concurrent Safety**: Database constraints prevent duplicate reservations
4. **Graceful Degradation**: Service continues if IONOS API is unavailable

## Troubleshooting

### Problem: High Blacklist Rate

**Symptoms**: > 30% of reserved IPs are blacklisted

**Solutions**:
- Try different location (us/ewr vs us/las)
- Check if entire subnet is blacklisted
- Consider requesting new IP ranges from IONOS

### Problem: Quota Exhausted

**Symptoms**: Cannot reserve new IPs, quota at limit

**Solutions**:
- Run cleanup: `./scripts/cleanup-unused-ips.sh`
- Release old IPs: Update status to "released"
- Review and delete old reservations
- Contact IONOS to increase quota

### Problem: IPs Becoming Blacklisted After Use

**Symptoms**: Clean IPs become blacklisted during usage

**Solutions**:
- Review sending patterns and volume
- Check email content quality
- Implement proper SPF/DKIM/DMARC
- Monitor recipient complaints
- Consider warming up IPs gradually

### Problem: Slow Reservation Process

**Symptoms**: Reserving 11 IPs takes > 5 minutes

**Solutions**:
- This is normal due to:
  - IONOS API rate limits
  - Blacklist checks (10+ DNS queries per IP)
  - Multiple attempts for blacklisted IPs
- Consider running reservations during off-peak hours
- Reserve in smaller batches

## Integration Example

```go
// Example: Get a clean IP for sending
func GetCleanIPForSending() (string, error) {
    // List available clean IPs
    ips, err := database.ListReservedIPs(
        stringPtr("reserved"),
        boolPtr(false),  // not blacklisted
        nil,
    )
    if err != nil {
        return "", err
    }
    
    if len(ips) == 0 {
        return "", errors.New("no clean IPs available")
    }
    
    // Use the first available IP
    ip := ips[0]
    
    // Update status to in_use
    err = database.UpdateReservedIPStatus(
        ip.ID,
        "in_use",
        stringPtr("smtp-sender-1"),
    )
    if err != nil {
        return "", err
    }
    
    return ip.IPAddress, nil
}
```

## Monitoring Metrics

Key metrics exposed on `/metrics` endpoint:

- `ionos_reservations_total{status="success|failure"}`
- `ionos_blacklist_checks_total{result="clean|dirty"}`
- `ionos_quota_usage{type="total|remaining"}`
- `ionos_ip_status_count{status="reserved|in_use|released|quarantined"}`
- `ionos_reservation_duration_seconds`

## Related Systems

- **IP Reputation System**: Tracks SMTP failures and reputation
- **SMTP Sending Service**: Uses reserved IPs for email delivery
- **Monitoring Dashboard**: Visualizes IP health and metrics
- **Alert System**: Notifies on quota limits and blacklist issues

## Support

For issues or questions:
1. Check logs: `docker-compose logs app`
2. Review reservation attempts: `SELECT * FROM ip_reservation_attempts ORDER BY attempted_at DESC LIMIT 10;`
3. Check blacklist history: `SELECT * FROM reserved_ip_blacklist_history WHERE was_blacklisted = true;`
4. Monitor quota: `./scripts/check-quota.sh`

