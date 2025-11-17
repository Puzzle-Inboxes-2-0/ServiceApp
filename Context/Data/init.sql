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

