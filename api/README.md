# SPEEDRUN API

The SPEEDRUN API is a service that facilitates intent-based cross-chain management and fulfillment. It provides a RESTful interface for creating, retrieving, and managing intent-based cross-chain transfers.

## Overview

SPEEDRUN API is designed to work with multiple EVM chains, allowing users to create intents for cross-chain transfers and track their fulfillment status. The API handles the following key functionalities:

- Creating and managing cross-chain transfer intents
- Retrieving intent details and status
- Listing intents with pagination support
- Monitoring intent fulfillment events
- Tracking settlements and fulfillments
- Real-time event processing and reconciliation

## Architecture

The API is built with a clean architecture approach, separating concerns into distinct layers:

```
api/
├── config/       # Configuration management
├── db/           # Database interactions
├── handlers/     # HTTP request handlers
├── models/       # Data models and types
├── services/     # Business logic
├── utils/        # Utility functions
└── main.go       # Application entry point
```

### Key Components

- **Handlers**: HTTP request handlers using the Gin framework
  - Intent management endpoints
  - Fulfillment tracking
  - Health monitoring
  - Request validation and error handling

- **Services**: Business logic for intent management and event processing
  - Intent service for managing transfer intents
  - Fulfillment service for tracking fulfillments
  - Settlement service for managing settlements
  - Event catchup service for processing missed events

- **Models**: Data structures for intents, events, and responses
  - Intent model for transfer requests
  - Fulfillment model for tracking fulfillments
  - Settlement model for settlement records
  - Response models for API endpoints

- **Database**: PostgreSQL database for persistent storage
  - Optimized queries with indexes
  - Transaction history tracking
  - Analytics views for monitoring
  - Block tracking for event processing

- **Event Processing**: Real-time blockchain event monitoring
  - Multi-chain support
  - Automatic catchup for missed events
  - Robust error handling and recovery
  - Health monitoring and diagnostics

## Getting Started

### Prerequisites

- Go 1.20 or higher
- PostgreSQL 12 or higher
- Access to supported chain RPC endpoints:
  - Ethereum Mainnet
  - Arbitrum
  - Base
  - Polygon
  - BSC
  - Avalanche
  - ZetaChain

### Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/speedrun-hq/speedrun
   cd speedrun/api
   ```

2. Install dependencies:
   ```bash
   go mod download
   ```

3. Copy the example environment file and configure it:
   ```bash
   cp .env.example .env
   # Edit .env with your configuration
   ```

4. Build the application:
   ```bash
   go build -o speedrun
   ```

### Configuration

The API is configured using environment variables:

- `PORT`: The port the API server will listen on
- `DATABASE_URL`: PostgreSQL connection string
- `SUPPORTED_CHAINS`: Comma-separated list of supported chain IDs
- Chain-specific configurations:
  - `{CHAIN}_RPC_URL`: RPC endpoint URL
  - `{CHAIN}_INTENT_ADDR`: Contract address
  - `{CHAIN}_BLOCK_INTERVAL`: Block processing interval
  - `{CHAIN}_MAX_RETRIES`: Maximum retry attempts
  - `{CHAIN}_RETRY_DELAY`: Delay between retries
  - `{CHAIN}_CONFIRMATIONS`: Required block confirmations

## API Endpoints

### Intents

#### Create Intent
```
POST /api/v1/intents
```

Request body:
```json
{
  "source_chain": "zeta",
  "destination_chain": "42161",
  "token": "0x1234567890123456789012345678901234567890",
  "amount": "1000000000000000000",
  "recipient": "0x0987654321098765432109876543210987654321",
  "intent_fee": "100000000000000000"
}
```

#### Get Intent
```
GET /api/v1/intents/:id
```

#### List Intents
```
GET /api/v1/intents?page=1&page_size=10&status=pending
```

#### Get Intents by Sender
```
GET /api/v1/intents/sender/:sender
```

#### Get Intents by Recipient
```
GET /api/v1/intents/recipient/:recipient
```

### Fulfillments

#### Create Fulfillment
```
POST /api/v1/fulfillments
```

#### Get Fulfillment
```
GET /api/v1/fulfillments/:id
```

#### List Fulfillments
```
GET /api/v1/fulfillments?page=1&page_size=10
```

### Health Check
```
GET /health
```

### Prometheus Metrics
```
GET /metrics
```
Standard Prometheus metrics endpoint for monitoring intent service health, event processing, and performance.

### Metrics Summary (JSON)
```
GET /api/v1/metrics
```
Human-readable JSON endpoint showing detailed metrics for all chains.

## Event Monitoring

The API automatically monitors the blockchain for intent-related events:

- **IntentInitiated**: When a new intent is created on the blockchain
- **Fulfillment**: When an intent is fulfilled on the target chain
- **Settlement**: When a fulfillment is settled

The service processes these events and updates the database accordingly, with automatic catchup for any missed events.

## Monitoring and Metrics

The API exposes comprehensive Prometheus metrics for monitoring intent service health and performance:

- **Service Health**: Monitor service uptime, active goroutines, and subscription health
- **Event Processing**: Track event processing rates, errors, and duplicate detection  
- **Connection Monitoring**: Monitor WebSocket reconnections and subscription stability
- **Performance Metrics**: Track processing latency and resource usage

For detailed information about available metrics, Prometheus configuration, and recommended alerts, see [`PROMETHEUS_METRICS.md`](PROMETHEUS_METRICS.md).

## Development

### Running Tests
```bash
go test ./...
```

### Adding New Features

1. Define models in the `models` package
2. Implement business logic in the `services` package
3. Create HTTP handlers in the `handlers` package
4. Add tests for your implementation
5. Update the README if necessary

## License

This project is licensed under the MIT License - see the LICENSE file for details.
