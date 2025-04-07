# API Schema

This document describes the API endpoints for Speedrun, a system for intent-based cross-chain USDC transfers between Base and Arbitrum networks.

## Create Intent

Creates a new intent for USDC transfers between Base and Arbitrum networks.

### Request

```json
{
  "source_chain": "base",
  "target_chain": "arbitrum",
  "token": "USDC",
  "amount": "100.00",
  "recipient": "0x1234567890123456789012345678901234567890",
  "intent_fee": "0.01"
}
```

### Response

```json
{
  "message": "Intent created successfully",
  "intent": {
    "id": "1",
    "source_chain": "base",
    "target_chain": "arbitrum",
    "token": "USDC",
    "amount": "100.00",
    "recipient": "0x1234567890123456789012345678901234567890",
    "intent_fee": "0.01",
    "status": "pending",
    "created_at": "2024-03-20T10:00:00Z",
    "updated_at": "2024-03-20T10:00:00Z"
  }
}
```

## List Intents

Lists all intents with optional pagination.

### Request

```json
{
  "page": 1,
  "limit": 10
}
```

### Response

```json
{
  "intents": [
    {
      "id": "1",
      "source_chain": "base",
      "target_chain": "arbitrum",
      "token": "USDC",
      "amount": "100.00",
      "recipient": "0x1234567890123456789012345678901234567890",
      "intent_fee": "0.01",
      "status": "pending",
      "created_at": "2024-03-20T10:00:00Z",
      "updated_at": "2024-03-20T10:00:00Z"
    }
  ],
  "total": 1,
  "page": 1,
  "limit": 10
}
```

## Chain Support

The following chains are currently supported:

- `base` - Base Mainnet
- `arbitrum` - Arbitrum Mainnet

## Error Responses

All endpoints may return the following error responses:

```json
{
  "error": "Error message description"
}
```

Common HTTP status codes:
- 400: Bad Request
- 404: Not Found
- 500: Internal Server Error 