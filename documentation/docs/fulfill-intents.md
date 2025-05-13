# Fulfill Intents

This page will explain how to fulfill intents on Speedrun.

## Overview

To fulfill intents on Speedrun, fulfillers need to detect intents on the source chain and then call the fulfill function on the target chain.

### Detecting Intents

Intents can be detected by parsing events from the intent contract on the connected source chain. When a user initiates an intent, the following event is emitted:

```solidity
event IntentInitiated(
    bytes32 indexed intentId,
    address indexed asset,
    uint256 amount,
    uint256 targetChain,
    bytes receiver,
    uint256 tip,
    uint256 salt
);
```

By monitoring these events, fulfillers can identify new intents that need to be fulfilled.

### Fulfilling Intents

Once an intent is detected, fulfillers must call the `fulfill` function on the connected target chain:

```solidity
function fulfill(
    bytes32 intentId,
    address asset,
    uint256 amount,
    address receiver
)
```

This function transfers the specified tokens to the receiver on the target chain, completing the cross-chain operation.

## More Advanced and Optimized Fulfillment

While the above represents the general fulfillment flow, Speedrun provides additional tools to simplify the process:

1. **Intent API**: A dedicated API to retrieve active intents without needing to parse blockchain events directly
2. **Minimal Fulfiller Process**: A reference implementation that handles the core fulfillment logic

These tools help developers get started with fulfilling intents with minimal setup requirements.

## Querying Fulfillments via API

Speedrun provides a comprehensive API to query fulfillment events and track the status of fulfilled intents.

### Get Fulfillment by ID

To fetch details of a specific fulfillment:

```bash
GET /api/v1/fulfillments/:id
```

Example request:
```bash
curl -X GET "https://api.speedrun.exchange/api/v1/fulfillments/0x1234567890123456789012345678901234567890123456789012345678901234"
```

Response:
```json
{
  "id": "0x1234567890123456789012345678901234567890123456789012345678901234",
  "asset": "0xaf88d065e77c8cc2239327c5edb3a432268e5831",
  "amount": "1000000000",
  "receiver": "0x0987654321098765432109876543210987654321",
  "tx_hash": "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
  "created_at": "2024-03-14T12:00:00Z",
  "updated_at": "2024-03-14T12:00:00Z"
}
```

### List Fulfillments

To list all fulfillments with pagination:

```bash
GET /api/v1/fulfillments?page=1&page_size=10
```

Query Parameters:
- `page`: Page number (default: 1)
- `page_size`: Number of items per page (default: 10)

Example request:
```bash
curl -X GET "https://api.speedrun.exchange/api/v1/fulfillments?page=1&page_size=10"
```

### Response Fields

The fulfillment response includes the following fields:

- `id`: Unique identifier of the fulfillment (matches the intent ID)
- `asset`: Address of the token that was transferred
- `amount`: Amount of tokens transferred (in smallest unit)
- `receiver`: Address of the recipient on the destination chain
- `tx_hash`: Transaction hash of the fulfillment transaction
- `created_at`: Timestamp when the fulfillment was created
- `updated_at`: Timestamp when the fulfillment was last updated

### Tracking Fulfillment Status

To track the status of a fulfillment, you can:

1. Query the intent status using the intent ID (which matches the fulfillment ID)
2. Monitor the settlement status through the settlements endpoint

Example intent status check:
```bash
curl -X GET "https://api.speedrun.exchange/api/v1/intents/0x1234567890123456789012345678901234567890123456789012345678901234"
```

The intent status will reflect the fulfillment state:
- `pending`: Intent created but not yet fulfilled
- `fulfilled`: Intent has been fulfilled but not yet settled
- `settled`: Intent has been fully settled
- `failed`: Fulfillment failed or was rejected

### Error Handling

The API returns appropriate HTTP status codes and error messages:

- `200 OK`: Request successful
- `400 Bad Request`: Invalid parameters
- `404 Not Found`: Fulfillment not found
- `500 Internal Server Error`: Server-side error

Example error response:
```json
{
  "error": "Fulfillment not found",
  "code": "NOT_FOUND"
}
```

### Best Practices

1. **Polling Frequency**: When monitoring fulfillments, use a reasonable polling interval (e.g., every 30 seconds) to avoid rate limiting
2. **Error Handling**: Implement proper error handling and retry logic for failed API calls
3. **Status Tracking**: Use the intent status endpoint to track the complete lifecycle of a fulfillment
4. **Transaction Verification**: Always verify the transaction hash returned in the fulfillment response on the blockchain
