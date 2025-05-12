# API Reference

This document provides detailed information about the Speedrun API endpoints, request/response formats, and authentication.

## Base URL

```
https://api.speedrun.exchange/api/v1
```

## Endpoints

### Intents

#### Create Intent

```http
POST /intents
```

Request body:
```json
{
  "asset": "0xaf88d065e77c8cc2239327c5edb3a432268e5831",
  "amount": "1000000000",
  "targetChain": 8453,
  "receiver": "0x742d35Cc6634C0532925a3b844Bc454e4438f44e",
  "tip": "3000000"
}
```

Response:
```json
{
  "id": "int_123456789",
  "status": "pending",
  "asset": "0xaf88d065e77c8cc2239327c5edb3a432268e5831",
  "amount": "1000000000",
  "targetChain": 8453,
  "receiver": "0x742d35Cc6634C0532925a3b844Bc454e4438f44e",
  "tip": "3000000",
  "created_at": "2024-03-20T10:00:00Z"
}
```

#### Get Intent

```http
GET /intents/{intent_id}
```

Response:
```json
{
  "id": "int_123456789",
  "status": "fulfilled",
  "asset": "0xaf88d065e77c8cc2239327c5edb3a432268e5831",
  "amount": "1000000000",
  "targetChain": 8453,
  "receiver": "0x742d35Cc6634C0532925a3b844Bc454e4438f44e",
  "tip": "3000000",
  "created_at": "2024-03-20T10:00:00Z",
  "fulfilled_at": "2024-03-20T10:00:05Z",
  "fulfillment_tx": "0x123...",
  "settled_at": "2024-03-20T10:00:10Z",
  "settlement_tx": "0x456..."
}
```

#### List Intents

```http
GET /intents
```

Query parameters:
- `page`: Page number (default: 1)
- `limit`: Items per page (default: 10, max: 100)
- `status`: Filter by status (pending, fulfilled, settled, failed)
- `sender`: Filter by sender address
- `receiver`: Filter by receiver address

Response:
```json
{
  "data": [
    {
      "id": "int_123456789",
      "status": "fulfilled",
      "asset": "0xaf88d065e77c8cc2239327c5edb3a432268e5831",
      "amount": "1000000000",
      "targetChain": 8453,
      "receiver": "0x742d35Cc6634C0532925a3b844Bc454e4438f44e",
      "tip": "3000000",
      "created_at": "2024-03-20T10:00:00Z",
      "fulfilled_at": "2024-03-20T10:00:05Z",
      "fulfillment_tx": "0x123..."
    }
  ],
  "pagination": {
    "total": 100,
    "page": 1,
    "limit": 10,
    "pages": 10
  }
}
```

### Fulfillments

#### Get Fulfillment

```http
GET /fulfillments/{fulfillment_id}
```

Response:
```json
{
  "id": "ful_123456789",
  "intent_id": "int_123456789",
  "status": "completed",
  "asset": "0xaf88d065e77c8cc2239327c5edb3a432268e5831",
  "amount": "1000000000",
  "receiver": "0x742d35Cc6634C0532925a3b844Bc454e4438f44e",
  "tx_hash": "0x123...",
  "created_at": "2024-03-20T10:00:05Z",
  "updated_at": "2024-03-20T10:00:10Z"
}
```

#### List Fulfillments

```http
GET /fulfillments
```

Query parameters:
- `page`: Page number (default: 1)
- `limit`: Items per page (default: 10, max: 100)
- `status`: Filter by status (pending, completed, failed)
- `intent_id`: Filter by intent ID

Response:
```json
{
  "data": [
    {
      "id": "ful_123456789",
      "intent_id": "int_123456789",
      "status": "completed",
      "asset": "0xaf88d065e77c8cc2239327c5edb3a432268e5831",
      "amount": "1000000000",
      "receiver": "0x742d35Cc6634C0532925a3b844Bc454e4438f44e",
      "tx_hash": "0x123...",
      "created_at": "2024-03-20T10:00:05Z",
      "updated_at": "2024-03-20T10:00:10Z"
    }
  ],
  "pagination": {
    "total": 100,
    "page": 1,
    "limit": 10,
    "pages": 10
  }
}
```

### Settlements

#### Get Settlement

```http
GET /settlements/{settlement_id}
```

Response:
```json
{
  "id": "set_123456789",
  "intent_id": "int_123456789",
  "fulfillment_id": "ful_123456789",
  "status": "completed",
  "tx_hash": "0x456...",
  "created_at": "2024-03-20T10:00:10Z",
  "updated_at": "2024-03-20T10:00:15Z"
}
```

#### List Settlements

```http
GET /settlements
```

Query parameters:
- `page`: Page number (default: 1)
- `limit`: Items per page (default: 10, max: 100)
- `status`: Filter by status (pending, completed, failed)
- `intent_id`: Filter by intent ID
- `fulfillment_id`: Filter by fulfillment ID

Response:
```json
{
  "data": [
    {
      "id": "set_123456789",
      "intent_id": "int_123456789",
      "fulfillment_id": "ful_123456789",
      "status": "completed",
      "tx_hash": "0x456...",
      "created_at": "2024-03-20T10:00:10Z",
      "updated_at": "2024-03-20T10:00:15Z"
    }
  ],
  "pagination": {
    "total": 100,
    "page": 1,
    "limit": 10,
    "pages": 10
  }
}
```

## Error Handling

All errors follow this format:

```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "Human readable error message",
    "details": {
      // Additional error details if available
    }
  }
}
```

Common error codes:
- `INVALID_REQUEST`: The request was malformed
- `UNAUTHORIZED`: Invalid or missing API key
- `RATE_LIMITED`: Too many requests
- `NOT_FOUND`: Resource not found
- `INTERNAL_ERROR`: Server error

## Webhooks

Speedrun can send webhook notifications for various events. Configure webhooks in your dashboard.

### Webhook Events

- `intent.created`: New intent created
- `intent.fulfilled`: Intent fulfilled
- `intent.settled`: Intent settled

### Webhook Payload

```json
{
  "event": "intent.fulfilled",
  "timestamp": "2024-03-20T10:00:05Z",
  "data": {
    "id": "int_123456789",
    "status": "fulfilled",
    "asset": "0xaf88d065e77c8cc2239327c5edb3a432268e5831",
    "amount": "1000000000",
    "targetChain": 8453,
    "receiver": "0x742d35Cc6634C0532925a3b844Bc454e4438f44e",
    "tip": "3000000",
    "created_at": "2024-03-20T10:00:00Z",
    "fulfilled_at": "2024-03-20T10:00:05Z",
    "fulfillment_tx": "0x123..."
  }
}
```

## Support

For API support:
- [GitHub Issues](https://github.com/speedrun-hq/speedrun/issues) 