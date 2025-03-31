-- Create intents table
CREATE TABLE IF NOT EXISTS intents (
    id VARCHAR(36) PRIMARY KEY,
    source_chain VARCHAR(50) NOT NULL,
    destination_chain VARCHAR(50) NOT NULL,
    token VARCHAR(10) NOT NULL,
    amount VARCHAR(50) NOT NULL,
    recipient VARCHAR(42) NOT NULL,
    intent_fee VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

-- Create fulfillments table
CREATE TABLE IF NOT EXISTS fulfillments (
    id VARCHAR(36) PRIMARY KEY,
    intent_id VARCHAR(36) NOT NULL REFERENCES intents(id),
    fulfiller VARCHAR(42) NOT NULL,
    amount VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_intents_status ON intents(status);
CREATE INDEX IF NOT EXISTS idx_fulfillments_intent_id ON fulfillments(intent_id);
CREATE INDEX IF NOT EXISTS idx_fulfillments_status ON fulfillments(status); 