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

_More detailed documentation coming soon_
