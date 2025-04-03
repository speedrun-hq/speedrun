-- Create intents table
CREATE TABLE IF NOT EXISTS intents (
    id VARCHAR(66) PRIMARY KEY,
    source_chain BIGINT NOT NULL,
    destination_chain BIGINT NOT NULL,
    token VARCHAR(42) NOT NULL,
    amount VARCHAR(78) NOT NULL,
    recipient VARCHAR(42) NOT NULL,
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

-- Create last_processed_blocks table
CREATE TABLE IF NOT EXISTS last_processed_blocks (
    chain_id BIGINT PRIMARY KEY,
    block_number BIGINT NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_intents_status ON intents(status);
CREATE INDEX IF NOT EXISTS idx_fulfillments_id ON fulfillments(id);