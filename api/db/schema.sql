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