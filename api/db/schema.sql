-- Create intents table
CREATE TABLE IF NOT EXISTS intents (
    id VARCHAR(66) PRIMARY KEY,
    source_chain BIGINT NOT NULL,
    destination_chain BIGINT NOT NULL,
    token VARCHAR(42) NOT NULL,
    amount VARCHAR(78) NOT NULL,
    recipient VARCHAR(42) NOT NULL,
    sender VARCHAR(42) NOT NULL,
    intent_fee VARCHAR(78) NOT NULL,
    status VARCHAR(20) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create fulfillments table
CREATE TABLE IF NOT EXISTS fulfillments (
    id VARCHAR(66) PRIMARY KEY,
    asset VARCHAR(42) NOT NULL,
    amount VARCHAR(78) NOT NULL,
    receiver VARCHAR(42) NOT NULL,
    tx_hash VARCHAR(66) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create settlements table
CREATE TABLE IF NOT EXISTS settlements (
    id VARCHAR(66) PRIMARY KEY,
    asset VARCHAR(42) NOT NULL,
    amount VARCHAR(78) NOT NULL,
    receiver VARCHAR(42) NOT NULL,
    fulfilled BOOLEAN NOT NULL,
    fulfiller VARCHAR(42) NOT NULL,
    actual_amount VARCHAR(78) NOT NULL,
    paid_tip VARCHAR(78) NOT NULL,
    tx_hash VARCHAR(66) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create last_processed_blocks table
CREATE TABLE IF NOT EXISTS last_processed_blocks (
    chain_id BIGINT PRIMARY KEY,
    block_number BIGINT NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_intents_status ON intents(status);
CREATE INDEX IF NOT EXISTS idx_fulfillments_id ON fulfillments(id);
CREATE INDEX IF NOT EXISTS idx_settlements_id ON settlements(id);

-- Create composite indexes for improved query performance
CREATE INDEX IF NOT EXISTS idx_intents_status_created_at ON intents(status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_intents_sender ON intents(sender);
CREATE INDEX IF NOT EXISTS idx_intents_recipient ON intents(recipient);
CREATE INDEX IF NOT EXISTS idx_intents_sender_status ON intents(sender, status);
CREATE INDEX IF NOT EXISTS idx_intents_recipient_status ON intents(recipient, status);
CREATE INDEX IF NOT EXISTS idx_intents_created_at ON intents(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_fulfillments_created_at ON fulfillments(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_settlements_created_at ON settlements(created_at DESC);

-- Create views for analytics and reporting

-- 1. Intent Lifecycle View: Track the full lifecycle of intents
CREATE OR REPLACE VIEW intent_lifecycle_view AS
SELECT
    i.id,
    i.source_chain,
    i.destination_chain,
    i.token,
    i.amount,
    i.recipient,
    i.sender,
    i.intent_fee,
    i.status,
    i.created_at as intent_created_at,
    f.created_at as fulfillment_time,
    s.created_at as settlement_time,
    s.fulfilled as settlement_fulfilled,
    s.fulfiller,
    s.actual_amount,
    s.paid_tip,
    CASE
        WHEN s.created_at IS NOT NULL AND f.created_at IS NOT NULL 
        THEN EXTRACT(EPOCH FROM (s.created_at - i.created_at))
        ELSE NULL
    END as total_processing_time_seconds,
    CASE
        WHEN f.created_at IS NOT NULL 
        THEN EXTRACT(EPOCH FROM (f.created_at - i.created_at))
        ELSE NULL
    END as time_to_fulfillment_seconds
FROM 
    intents i
LEFT JOIN 
    fulfillments f ON i.id = f.id
LEFT JOIN 
    settlements s ON i.id = s.id
ORDER BY 
    i.created_at DESC;

-- 2. User Activity View: Track user metrics for both senders and receivers
CREATE OR REPLACE VIEW user_activity_view AS
SELECT
    sender as address,
    'sender' as role,
    COUNT(*) as transaction_count,
    SUM(CAST(amount as NUMERIC)) as total_amount,
    MIN(created_at) as first_activity,
    MAX(created_at) as last_activity
FROM
    intents
GROUP BY
    sender, role
UNION ALL
SELECT
    recipient as address,
    'receiver' as role,
    COUNT(*) as transaction_count,
    SUM(CAST(amount as NUMERIC)) as total_amount,
    MIN(created_at) as first_activity,
    MAX(created_at) as last_activity
FROM
    intents
GROUP BY
    recipient, role;

-- 3. Chain Activity View: Track activity across different chains
CREATE OR REPLACE VIEW chain_activity_view AS
SELECT
    source_chain,
    destination_chain,
    COUNT(*) as transaction_count,
    SUM(CAST(amount as NUMERIC)) as total_volume,
    AVG(CAST(intent_fee as NUMERIC)) as avg_fee,
    MIN(created_at) as first_transaction,
    MAX(created_at) as last_transaction
FROM
    intents
GROUP BY
    source_chain, destination_chain
ORDER BY
    transaction_count DESC;

-- 4. Settlement Performance View: Monitor performance by fulfiller
CREATE OR REPLACE VIEW settlement_performance_view AS
SELECT
    fulfiller,
    COUNT(*) as settlement_count,
    SUM(CASE WHEN fulfilled THEN 1 ELSE 0 END) as successful_settlements,
    AVG(CAST(paid_tip as NUMERIC)) as avg_tip_paid,
    SUM(CAST(paid_tip as NUMERIC)) as total_tips_earned,
    MIN(created_at) as first_settlement,
    MAX(created_at) as last_settlement
FROM
    settlements
GROUP BY
    fulfiller
ORDER BY
    settlement_count DESC;

-- 5. Leaderboard View: Simplify leaderboard calculations
CREATE OR REPLACE VIEW leaderboard_view AS
SELECT
    sender as address,
    source_chain as chain_id,
    COUNT(*) as total_transfers,
    SUM(CAST(amount as NUMERIC)) as total_volume,
    AVG(EXTRACT(EPOCH FROM (updated_at - created_at))) as avg_completion_time_seconds,
    MIN(EXTRACT(EPOCH FROM (updated_at - created_at))) as fastest_completion_time_seconds,
    MAX(updated_at) as last_transfer_time
FROM
    intents
WHERE
    status = 'settled'
GROUP BY
    sender, source_chain
ORDER BY
    total_volume DESC;