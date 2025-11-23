-- Initialize the database schema

CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create index on username for faster lookups
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);

-- Create index on email for faster lookups
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

-- Insert sample data
INSERT INTO users (username, email) VALUES
    ('john_doe', 'john.doe@example.com'),
    ('jane_smith', 'jane.smith@example.com')
ON CONFLICT (username) DO NOTHING;

-- ========================================
-- IP Reputation & SMTP Failure Tracking
-- ========================================

-- Track individual SMTP delivery failures
CREATE TABLE IF NOT EXISTS smtp_failures (
    id SERIAL PRIMARY KEY,
    sending_ip VARCHAR(45) NOT NULL,
    recipient_email VARCHAR(255) NOT NULL,
    recipient_domain VARCHAR(255) NOT NULL,
    smtp_code INTEGER,
    enhanced_code VARCHAR(20),  -- Increased from VARCHAR(10) to handle longer codes
    reason TEXT,
    mx_server VARCHAR(255),
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    event_id VARCHAR(255) UNIQUE NOT NULL,  -- CRITICAL: UNIQUE constraint prevents duplicate webhook events
    attempt_number INTEGER DEFAULT 1
);

-- Indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_smtp_failures_ip_timestamp ON smtp_failures(sending_ip, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_smtp_failures_domain_timestamp ON smtp_failures(recipient_domain, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_smtp_failures_enhanced_code ON smtp_failures(enhanced_code);
CREATE INDEX IF NOT EXISTS idx_smtp_failures_timestamp ON smtp_failures(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_smtp_failures_event_id ON smtp_failures(event_id);  -- Fast lookup for deduplication

-- Store aggregated IP reputation metrics
CREATE TABLE IF NOT EXISTS ip_reputation_metrics (
    id SERIAL PRIMARY KEY,
    ip VARCHAR(45) UNIQUE NOT NULL,
    window_start TIMESTAMP WITH TIME ZONE NOT NULL,
    window_end TIMESTAMP WITH TIME ZONE NOT NULL,
    total_sent INTEGER DEFAULT 0,
    total_rejected INTEGER DEFAULT 0,
    rejection_ratio DECIMAL(5,4) DEFAULT 0.0000,
    unique_domains_rejected INTEGER DEFAULT 0,
    distinct_rejection_reasons JSONB DEFAULT '{}',
    major_providers_rejecting JSONB DEFAULT '[]',
    status VARCHAR(20) DEFAULT 'healthy',
    last_updated TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    metadata JSONB DEFAULT '{}'
);

-- Indexes for IP reputation queries
CREATE INDEX IF NOT EXISTS idx_ip_reputation_ip ON ip_reputation_metrics(ip);
CREATE INDEX IF NOT EXISTS idx_ip_reputation_status ON ip_reputation_metrics(status);
CREATE INDEX IF NOT EXISTS idx_ip_reputation_updated ON ip_reputation_metrics(last_updated DESC);

-- Track DNSBL (DNS Blacklist) check results
CREATE TABLE IF NOT EXISTS dnsbl_checks (
    id SERIAL PRIMARY KEY,
    ip VARCHAR(45) NOT NULL,
    checked_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    listed BOOLEAN DEFAULT FALSE,
    listings JSONB DEFAULT '[]',
    check_duration_ms INTEGER,
    metadata JSONB DEFAULT '{}'
);

-- Indexes for DNSBL checks
CREATE INDEX IF NOT EXISTS idx_dnsbl_checks_ip ON dnsbl_checks(ip);
CREATE INDEX IF NOT EXISTS idx_dnsbl_checks_timestamp ON dnsbl_checks(checked_at DESC);
CREATE INDEX IF NOT EXISTS idx_dnsbl_checks_listed ON dnsbl_checks(listed);

-- Track IP actions and status changes
CREATE TABLE IF NOT EXISTS ip_actions (
    id SERIAL PRIMARY KEY,
    ip VARCHAR(45) NOT NULL,
    action VARCHAR(50) NOT NULL,
    previous_status VARCHAR(20),
    new_status VARCHAR(20),
    reason TEXT,
    triggered_by VARCHAR(100) DEFAULT 'automated',
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Index for IP actions
CREATE INDEX IF NOT EXISTS idx_ip_actions_ip ON ip_actions(ip);
CREATE INDEX IF NOT EXISTS idx_ip_actions_timestamp ON ip_actions(created_at DESC);

-- Log initialization
DO $$
BEGIN
    RAISE NOTICE 'Database initialized successfully with IP reputation tracking';
END $$;

-- ========================================
-- IONOS IP Reservation Management
-- ========================================

-- Track reserved IPs from IONOS with their status and metadata
CREATE TABLE IF NOT EXISTS reserved_ips (
    id SERIAL PRIMARY KEY,
    ip_address INET NOT NULL UNIQUE,
    reservation_block_id VARCHAR(255),  -- IONOS block ID
    uid VARCHAR(255) UNIQUE NOT NULL,   -- Our unique identifier
    location VARCHAR(100) DEFAULT 'us/ewr',  -- IONOS datacenter location
    
    -- Status tracking
    status VARCHAR(50) DEFAULT 'reserved',  -- reserved, in_use, released, quarantined
    is_blacklisted BOOLEAN DEFAULT FALSE,
    blacklist_details JSONB DEFAULT '[]',  -- List of blacklists the IP is on
    
    -- Lifecycle timestamps
    reserved_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_checked_at TIMESTAMP WITH TIME ZONE,
    released_at TIMESTAMP WITH TIME ZONE,
    
    -- Usage tracking
    assigned_to VARCHAR(255),  -- Service or user using this IP
    usage_count INTEGER DEFAULT 0,  -- How many times this IP has been used
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    notes TEXT,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for reserved_ips
CREATE INDEX IF NOT EXISTS idx_reserved_ips_status ON reserved_ips(status);
CREATE INDEX IF NOT EXISTS idx_reserved_ips_blacklisted ON reserved_ips(is_blacklisted);
CREATE INDEX IF NOT EXISTS idx_reserved_ips_location ON reserved_ips(location);
CREATE INDEX IF NOT EXISTS idx_reserved_ips_reserved_at ON reserved_ips(reserved_at DESC);
CREATE INDEX IF NOT EXISTS idx_reserved_ips_uid ON reserved_ips(uid);
CREATE INDEX IF NOT EXISTS idx_reserved_ips_block_id ON reserved_ips(reservation_block_id);

-- Track blacklist check history for reserved IPs
CREATE TABLE IF NOT EXISTS reserved_ip_blacklist_history (
    id SERIAL PRIMARY KEY,
    reserved_ip_id INTEGER NOT NULL REFERENCES reserved_ips(id) ON DELETE CASCADE,
    ip_address INET NOT NULL,
    checked_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    was_blacklisted BOOLEAN DEFAULT FALSE,
    blacklists_found JSONB DEFAULT '[]',  -- List of blacklists IP was found on
    check_duration_ms INTEGER,
    metadata JSONB DEFAULT '{}'
);

-- Indexes for blacklist history
CREATE INDEX IF NOT EXISTS idx_reserved_ip_bl_history_ip_id ON reserved_ip_blacklist_history(reserved_ip_id);
CREATE INDEX IF NOT EXISTS idx_reserved_ip_bl_history_checked_at ON reserved_ip_blacklist_history(checked_at DESC);
CREATE INDEX IF NOT EXISTS idx_reserved_ip_bl_history_blacklisted ON reserved_ip_blacklist_history(was_blacklisted);

-- Track IP reservation attempts and failures
CREATE TABLE IF NOT EXISTS ip_reservation_attempts (
    id SERIAL PRIMARY KEY,
    attempt_uid VARCHAR(255) UNIQUE NOT NULL,
    block_id VARCHAR(255),  -- IONOS block ID (may be null if reservation failed)
    ip_address INET,  -- May be null if reservation failed before IP was assigned
    location VARCHAR(100),
    
    -- Result
    success BOOLEAN DEFAULT FALSE,
    failure_reason TEXT,
    was_blacklisted BOOLEAN,
    blacklists_found JSONB DEFAULT '[]',
    
    -- Timing
    attempted_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,
    duration_ms INTEGER,
    
    -- Action taken
    action_taken VARCHAR(100),  -- kept, deleted, quarantined
    metadata JSONB DEFAULT '{}'
);

-- Indexes for reservation attempts
CREATE INDEX IF NOT EXISTS idx_ip_reservation_attempts_success ON ip_reservation_attempts(success);
CREATE INDEX IF NOT EXISTS idx_ip_reservation_attempts_attempted_at ON ip_reservation_attempts(attempted_at DESC);
CREATE INDEX IF NOT EXISTS idx_ip_reservation_attempts_location ON ip_reservation_attempts(location);

-- Track IONOS API quota usage
CREATE TABLE IF NOT EXISTS ionos_quota_snapshots (
    id SERIAL PRIMARY KEY,
    total_blocks INTEGER NOT NULL,
    estimated_limit INTEGER DEFAULT 50,
    remaining INTEGER,
    protected_blocks INTEGER DEFAULT 0,  -- Number of 11-IP blocks to never touch
    single_ip_blocks INTEGER DEFAULT 0,
    location VARCHAR(100),
    snapshot_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    metadata JSONB DEFAULT '{}'
);

-- Index for quota snapshots
CREATE INDEX IF NOT EXISTS idx_ionos_quota_snapshots_at ON ionos_quota_snapshots(snapshot_at DESC);

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Trigger to auto-update updated_at on reserved_ips
DROP TRIGGER IF EXISTS update_reserved_ips_updated_at ON reserved_ips;
CREATE TRIGGER update_reserved_ips_updated_at
    BEFORE UPDATE ON reserved_ips
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- View for quick IP status overview
CREATE OR REPLACE VIEW reserved_ips_summary AS
SELECT 
    status,
    COUNT(*) as count,
    COUNT(*) FILTER (WHERE is_blacklisted = TRUE) as blacklisted_count,
    location
FROM reserved_ips
GROUP BY status, location;

-- View for reservation success rate
CREATE OR REPLACE VIEW reservation_success_metrics AS
SELECT 
    DATE(attempted_at) as date,
    location,
    COUNT(*) as total_attempts,
    COUNT(*) FILTER (WHERE success = TRUE) as successful,
    COUNT(*) FILTER (WHERE was_blacklisted = TRUE) as blacklisted,
    ROUND(100.0 * COUNT(*) FILTER (WHERE success = TRUE) / NULLIF(COUNT(*), 0), 2) as success_rate
FROM ip_reservation_attempts
GROUP BY DATE(attempted_at), location
ORDER BY date DESC;

-- Notification
DO $$
BEGIN
    RAISE NOTICE 'IONOS IP Reservation tables initialized successfully';
END $$;

