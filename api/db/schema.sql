-- Create intents table
CREATE TABLE IF NOT EXISTS intents (
    id VARCHAR(66) PRIMARY KEY,
    source_chain VARCHAR(10) NOT NULL,
    destination_chain VARCHAR(10) NOT NULL,
    token VARCHAR(10) NOT NULL,
    amount VARCHAR(78) NOT NULL,
    recipient VARCHAR(42) NOT NULL,
    intent_fee VARCHAR(78) NOT NULL,
    status VARCHAR(20) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create fulfillments table
CREATE TABLE IF NOT EXISTS fulfillments (
    id SERIAL PRIMARY KEY,
    intent_id VARCHAR(66) NOT NULL,
    tx_hash VARCHAR(66) NOT NULL,
    status VARCHAR(20) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (intent_id) REFERENCES intents(id)
);

-- Table to store last processed block numbers
CREATE TABLE IF NOT EXISTS last_processed_blocks (
    chain_id VARCHAR(10) PRIMARY KEY,
    block_number BIGINT NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_intents_status ON intents(status);
CREATE INDEX IF NOT EXISTS idx_fulfillments_intent_id ON fulfillments(intent_id);
CREATE INDEX IF NOT EXISTS idx_fulfillments_status ON fulfillments(status); 