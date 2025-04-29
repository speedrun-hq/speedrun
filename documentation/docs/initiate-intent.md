# Initiate an Intent

## Overview

The following function should be called on the intent contract to initiate a new intent:

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

_Coming soon_
