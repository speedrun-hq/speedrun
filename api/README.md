# ZetaFast API

The ZetaFast API is a service that facilitates intent-based cross chain management and fulfillment. It provides a RESTful interface for creating, retrieving, and managing intent-based cross-chain transfers.

## Overview

ZetaFast API is designed to work with the ZetaChain ecosystem, allowing users to create intents for cross-chain transfers and track their fulfillment status. The API handles the following key functionalities:

- Creating intents for cross-chain transfers
- Retrieving intent details
- Listing all intents
- Monitoring intent fulfillment events
- Managing fulfillment status

## Architecture

The API is built with a clean architecture approach, separating concerns into distinct layers:

```
api/
├── config/       # Configuration management
├── db/           # Database interactions
├── handlers/     # HTTP request handlers
├── models/       # Data models and types
├── services/     # Business logic
├── test/         # Test utilities and mocks
└── utils/        # Utility functions
```

### Key Components

- **Handlers**: HTTP request handlers using the Gin framework
- **Services**: Business logic for intent management and event processing
- **Models**: Data structures for intents, events, and responses
- **Database**: PostgreSQL database for persistent storage
- **Ethereum Client**: Integration with Ethereum networks for event monitoring

## Getting Started

### Prerequisites

- Go 1.20 or higher
- PostgreSQL 12 or higher
- Access to Ethereum RPC endpoints

### Installation

1. Clone the repository:
   ```
   git clone https://github.com/zeta-chain/zetafast.git
   cd zetafast/api
   ```

2. Install dependencies:
   ```
   go mod download
   ```

3. Copy the example environment file and configure it:
   ```
   cp .env.example .env
   # Edit .env with your configuration
   ```

4. Build the application:
   ```
   go build -o zetafast
   ```

### Configuration

The API is configured using environment variables. See `.env.example` for all available options:

- `PORT`: The port the API server will listen on
- `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`: Database connection details
- `ETH_RPC_URL`: Ethereum RPC endpoint URL
- `CONTRACT_ADDRESS`: The address of the deployed contract
- `INTENT_INITIATED_EVENT_ABI`: The ABI for the IntentInitiated event

## API Endpoints

### Intents

#### Create Intent

```
POST /api/intents
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

Response:
```json
{
  "id": "0x1234567890123456789012345678901234567890123456789012345678901234",
  "source_chain": "zeta",
  "destination_chain": "42161",
  "token": "0x1234567890123456789012345678901234567890",
  "amount": "1000000000000000000",
  "recipient": "0x0987654321098765432109876543210987654321",
  "intent_fee": "100000000000000000",
  "status": "pending",
  "created_at": "2023-04-02T12:00:00Z",
  "updated_at": "2023-04-02T12:00:00Z"
}
```

#### Get Intent

```
GET /api/intents/:id
```

Response:
```json
{
  "id": "0x1234567890123456789012345678901234567890123456789012345678901234",
  "source_chain": "zeta",
  "destination_chain": "42161",
  "token": "0x1234567890123456789012345678901234567890",
  "amount": "1000000000000000000",
  "recipient": "0x0987654321098765432109876543210987654321",
  "intent_fee": "100000000000000000",
  "status": "pending",
  "created_at": "2023-04-02T12:00:00Z",
  "updated_at": "2023-04-02T12:00:00Z"
}
```

#### List Intents

```
GET /api/intents
```

Response:
```json
[
  {
    "id": "0x1234567890123456789012345678901234567890123456789012345678901234",
    "source_chain": "zeta",
    "destination_chain": "42161",
    "token": "0x1234567890123456789012345678901234567890",
    "amount": "1000000000000000000",
    "recipient": "0x0987654321098765432109876543210987654321",
    "intent_fee": "100000000000000000",
    "status": "pending",
    "created_at": "2023-04-02T12:00:00Z",
    "updated_at": "2023-04-02T12:00:00Z"
  }
]
```

## Event Monitoring

The API automatically monitors the blockchain for intent-related events:

- **IntentInitiated**: When a new intent is created on the blockchain
- **Fulfillment**: When an intent is fulfilled on the target chain

The service processes these events and updates the database accordingly.

## Development

### Running Tests

```
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