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

contract IntentCaller {
    function createIntent(address intentContract) external {
        IIntent(intentContract).initiate(
            0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE, // Token address (placeholder)
            1_000e18,                                   // Amount: 1000 tokens
            7000,                                       // Destination chain ID
            abi.encodePacked(msg.sender),               // Receiver (as bytes)
            10e18,                                      // Tip for fulfiller
            42                                          // Fixed salt value
        );
    }
}
```
