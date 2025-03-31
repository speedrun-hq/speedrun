# ZetaFast API Schema Documentation

## Overview
This document outlines the API schema for the ZetaFast service, which handles intent creation and fulfillment for USDC transfers between Ethereum, BASE, and ZetaChain networks.

## Base URL
```
http://localhost:8080/api/v1
```

## Authentication
All API endpoints require authentication. Include your API key in the request header:
```
Authorization: Bearer <your-api-key>
```

## Endpoints

### Health Check
```http
GET /health
```

Response:
```json
{
    "status": "ok"
}
```

### Intents

#### Create Intent
```http
POST /intents
```

Request Body:
```json
{
    "source_chain": "ethereum",
    "source_address": "0x...",
    "target_chain": "zetachain",
    "target_address": "0x...",
    "amount": "1000000"  // Amount in USDC (6 decimals)
}
```

Response:
```json
{
    "message": "Intent created successfully",
    "intent": {
        "id": "uuid",
        "source_chain": "ethereum",
        "source_address": "0x...",
        "target_chain": "zetachain",
        "target_address": "0x...",
        "amount": "1000000",
        "token": "USDC",
        "status": "pending",
        "created_at": "timestamp",
        "updated_at": "timestamp"
    }
}
```

#### Get Intent
```http
GET /intents/:id
```

Response:
```json
{
    "intent": {
        "id": "uuid",
        "source_chain": "ethereum",
        "source_address": "0x...",
        "target_chain": "zetachain",
        "target_address": "0x...",
        "amount": "1000000",
        "token": "USDC",
        "status": "pending",
        "created_at": "timestamp",
        "updated_at": "timestamp"
    }
}
```

#### List Intents
```http
GET /intents
```

Query Parameters:
- `status`: Filter by intent status (optional)
- `source_chain`: Filter by source chain (optional)
- `target_chain`: Filter by target chain (optional)
- `page`: Page number for pagination (default: 1)
- `limit`: Number of items per page (default: 10)

Response:
```json
{
    "intents": [
        {
            "id": "uuid",
            "source_chain": "ethereum",
            "source_address": "0x...",
            "target_chain": "zetachain",
            "target_address": "0x...",
            "amount": "1000000",
            "token": "USDC",
            "status": "pending",
            "created_at": "timestamp",
            "updated_at": "timestamp"
        }
    ],
    "total": 1,
    "page": 1,
    "limit": 10
}
```

### Fulfillments

#### Create Fulfillment
```http
POST /fulfillments
```

Request Body:
```json
{
    "intent_id": "uuid",
    "fulfiller": "0x...",
    "amount": "1000000"  // Amount in USDC (6 decimals)
}
```

Response:
```json
{
    "message": "Fulfillment created successfully",
    "fulfillment": {
        "id": "uuid",
        "intent_id": "uuid",
        "fulfiller": "0x...",
        "amount": "1000000",
        "token": "USDC",
        "status": "pending",
        "created_at": "timestamp",
        "updated_at": "timestamp"
    }
}
```

#### Get Fulfillment
```http
GET /fulfillments/:id
```

Response:
```json
{
    "fulfillment": {
        "id": "uuid",
        "intent_id": "uuid",
        "fulfiller": "0x...",
        "amount": "1000000",
        "token": "USDC",
        "status": "pending",
        "created_at": "timestamp",
        "updated_at": "timestamp"
    }
}
```

## Error Responses
All endpoints may return the following error responses:

### 400 Bad Request
```json
{
    "error": "Error message describing the validation failure"
}
```

### 401 Unauthorized
```json
{
    "error": "Invalid or missing API key"
}
```

### 404 Not Found
```json
{
    "error": "Resource not found"
}
```

### 500 Internal Server Error
```json
{
    "error": "Internal server error message"
}
```

## Data Types

### Chain Types
- `ethereum` - Ethereum Mainnet
- `base` - BASE Mainnet
- `zetachain` - ZetaChain Mainnet

### Token Type
- `USDC` (6 decimal places)

### Status Types
- `pending`
- `completed`
- `failed`
- `cancelled`

## Amount Format
All amounts are specified in the smallest unit of USDC (6 decimal places). For example:
- 1 USDC = 1000000
- 0.5 USDC = 500000
- 0.01 USDC = 10000


## Rate Limiting
API requests are limited to 100 requests per minute per API key.

## Pagination
List endpoints support pagination using the following query parameters:
- `page`: The page number (1-based)
- `limit`: The number of items per page (default: 10, max: 100)

## Versioning
The API is versioned through the URL path. The current version is v1. 