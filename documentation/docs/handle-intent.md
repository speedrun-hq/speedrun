# Handle an Intent

## Overview

_coming soon_

Currently, Speedrun only supports simple token transfers without any handling logic on the destination chain.

In the future, support for customizable callbacks on the destination chain will be introduced, allowing developers to define specific logic to be executed upon receipt.

Example: executing a trade on a destination chain DEX.

## Coming Soon: Intent Execution Interface

The upcoming intent execution feature will allow developers to implement a Solidity interface to handle intents on the destination chain. This interface will include two key functions:

```solidity
// Implemented by contract and called during fulfill
// Contains logic during intent execution
function onFulfill(
  bytes32 intentId,
  address asset,
  uint256 amount,
  bytes data
) external;

// Implemented by contract and called during settlement
// Can contain specific logic, like custom rewarding on the contract for the fulfiller
function onSettle(
  bytes32 intentId,
  address asset,
  uint256 amount,
  bytes data,
  bytes32 fulfillmentIndex
) external;
```

With these functions, developers will be able to execute custom logic when an intent is fulfilled and settled, enabling complex cross-chain operations like:

- Automated token swaps on destination DEXs
- Deposits into yield-generating protocols
- Dynamic NFT minting based on cross-chain data
- Cross-chain governance execution

## Example: Uniswap V2 Swap Implementation

Here's an example of how you might implement the `onFulfill` function to perform a token swap on Uniswap V2 using the cross-chain intent:

```solidity
// SPDX-License-Identifier: MIT
pragma solidity 0.8.26;

import "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import "@uniswap/v2-periphery/contracts/interfaces/IUniswapV2Router02.sol";

contract CrossChainSwapper {
    // Uniswap V2 Router address
    address public immutable uniswapRouter;

    constructor(address _uniswapRouter) {
        uniswapRouter = _uniswapRouter;
    }

    // Called by the protocol during intent fulfillment
    function onFulfill(
        bytes32 intentId,
        address asset,
        uint256 amount,
        bytes calldata data
    ) external {
        // Decode the swap parameters from the data field
        (
            address[] memory path,
            uint256 minAmountOut,
            uint256 deadline,
            address receiver
        ) = decodeSwapParams(data);

        // Ensure the first token in the path matches the received asset
        require(path[0] == asset, "Asset mismatch");

        // Transfer tokens from the sender to this contract
        IERC20(asset).transferFrom(msg.sender, address(this), amount);

        // Approve router to spend the tokens
        IERC20(asset).approve(uniswapRouter, amount);

        // Execute the swap on Uniswap
        IUniswapV2Router02(uniswapRouter).swapExactTokensForTokens(
            amount,
            minAmountOut,
            path,
            receiver,
            deadline
        );
    }

    // Helper function to decode swap parameters from the bytes data
    function decodeSwapParams(bytes memory data)
        internal
        pure
        returns (
            address[] memory path,
            uint256 minAmountOut,
            uint256 deadline,
            address receiver
        )
    {
        // Decode the packed data
        // Example format: path addresses + minAmountOut + deadline + receiver
        return abi.decode(data, (address[], uint256, uint256, address));
    }

    // onSettle implementation
    function onSettle(
        bytes32 intentId,
        address asset,
        uint256 amount,
        bytes calldata data,
        bytes32 fulfillmentIndex
    ) external {
        // Empty implementation
    }
}
```

In this example:

1. The `data` parameter contains encoded information about the swap: token path, minimum output amount, deadline, and the receiver address
2. The contract uses `transferFrom` to get the tokens from the sender (requires prior approval)
3. The `decodeSwapParams` function extracts all the parameters including the receiver from the encoded bytes
4. The contract then performs the swap using Uniswap V2's router
5. The swapped tokens are sent to the receiver address specified in the data
