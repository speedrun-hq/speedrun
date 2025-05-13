# Initiate an Intent

## Overview

The following function should be called on the intent contract to initiate a new intent for token transfer:

```solidity
function initiate(
  address asset,
  uint256 amount,
  uint256 targetChain,
  bytes calldata receiver,
  uint256 tip,
  uint256 salt
) external
```

Creates a new intent for cross-chain transfer:

- `asset`: Address of the token to transfer
- `amount`: Amount of tokens to transfer
- `targetChain`: Chain ID of the destination chain
- `receiver`: Address of the receiver on the target chain (in bytes format)
- `tip`: Amount of tokens to pay as fee for the cross-chain transfer. This value is fully customizable, fulfillers choose the intent to fulfill based on this tip.
- `salt`: Optional salt that can be used to make the intent ID non-predictable

## Example

Transferring 1000 USDC from Arbitrum to Base.

```solidity
interface IIntent {
  function initiate(
    address asset,
    uint256 amount,
    uint256 targetChain,
    bytes calldata receiver,
    uint256 tip,
    uint256 salt
  ) external;
}

contract IntentCreator {
  function createIntent() external {
    address arbitrumIntent = 0xD6B0E2a8D115cCA2823c5F80F8416644F3970dD2;

    IIntent(arbitrumIntent).initiate(
      0xaf88d065e77c8cc2239327c5edb3a432268e5831,   // USDC on Arbitrum
      1_000e6,                                      // Amount: 1000 tokens
      8453,                                         // Destination chain ID (here: Base)
      abi.encodePacked(msg.sender),                 // Receiver (as bytes)
      3e6,                                          // Tip for fulfiller
      42                                            // Fixed salt value
    );
  }
}
```

## Fetch the Intent

After initiating an intent, you can fetch its details using the Speedrun API. The API provides several endpoints to retrieve intent information.

### Get Intent by ID

To fetch a specific intent by its ID:

```bash
GET /api/v1/intents/:id
```

Example request:
```bash
curl -X GET "https://api.speedrun.exchange/api/v1/intents/0x1234567890123456789012345678901234567890123456789012345678901234"
```

Response:
```json
{
  "id": "0x1234567890123456789012345678901234567890123456789012345678901234",
  "source_chain": "42161",
  "destination_chain": "8453",
  "token": "0xaf88d065e77c8cc2239327c5edb3a432268e5831",
  "amount": "1000000000",
  "recipient": "0x0987654321098765432109876543210987654321",
  "intent_fee": "3000000",
  "status": "pending",
  "created_at": "2024-03-14T12:00:00Z",
  "updated_at": "2024-03-14T12:00:00Z"
}
```

### List Intents

To list all intents with pagination:

```bash
GET /api/v1/intents?page=1&page_size=10&status=pending
```

Query Parameters:
- `page`: Page number (default: 1)
- `page_size`: Number of items per page (default: 10)
- `status`: Filter by intent status (optional)

Example request:
```bash
curl -X GET "https://api.speedrun.exchange/api/v1/intents?page=1&page_size=10&status=pending"
```

### Get Intents by Sender

To fetch all intents created by a specific sender:

```bash
GET /api/v1/intents/sender/:sender
```

Example request:
```bash
curl -X GET "https://api.speedrun.exchange/api/v1/intents/sender/0x1234567890123456789012345678901234567890"
```

### Get Intents by Recipient

To fetch all intents where a specific address is the recipient:

```bash
GET /api/v1/intents/recipient/:recipient
```

Example request:
```bash
curl -X GET "https://api.speedrun.exchange/api/v1/intents/recipient/0x0987654321098765432109876543210987654321"
```

### Response Fields

The intent response includes the following fields:

- `id`: Unique identifier of the intent
- `source_chain`: Chain ID where the intent was created
- `destination_chain`: Chain ID where the transfer will be fulfilled
- `token`: Address of the token to be transferred
- `amount`: Amount of tokens to transfer (in smallest unit)
- `recipient`: Address of the recipient on the destination chain
- `intent_fee`: Fee amount for the fulfiller (in smallest unit)
- `status`: Current status of the intent (pending, fulfilled, settled, etc.)
- `created_at`: Timestamp when the intent was created
- `updated_at`: Timestamp when the intent was last updated

### Error Handling

The API returns appropriate HTTP status codes and error messages:

- `200 OK`: Request successful
- `400 Bad Request`: Invalid parameters
- `404 Not Found`: Intent not found
- `500 Internal Server Error`: Server-side error

Example error response:
```json
{
  "error": "Intent not found",
  "code": "NOT_FOUND"
}
```
