# Initiate an Intent for Smart Contract Calls

## Overview

Speedrun not only supports simple token transfers but also intents for custom smart contract calls on the target chain.

**Example of applications:**
- executing a trade on a destination chain DEX
- depositing into lending or yield protocols
- participating in governance votes cross-chain
- minting NFTs on other chains
- automated portfolio rebalancing across chains
- interacting with GameFi applications on different chains

## Implement an Intent Target Contract

To support smart contract call intents, the developer must implement the `IntentTarget` interface:

```solidity
/**
 * @title IntentTarget
 * @dev Interface for contracts that want to support intent calls
 * Contains logic during intent execution
 */
interface IntentTarget {
    /**
     * @dev Called during intent fulfillment to execute custom logic
     * @param intentId The ID of the intent
     * @param asset The ERC20 token address
     * @param amount Amount transferred
     * @param data Custom data for execution
     */
    function onFulfill(
        bytes32 intentId,
        address asset,
        uint256 amount,
        bytes calldata data
    ) external;

    /**
     * @dev Called during intent settlement to execute custom logic
     * @param intentId The ID of the intent
     * @param asset The ERC20 token address
     * @param amount Amount transferred
     * @param data Custom data for execution
     * @param fulfillmentIndex The fulfillment index for this intent
     * @param isFulfilled Whether the intent was fulfilled before settlement
     * @param tipAmount Tip amount for this intent, can be used to redistribute if not fulfilled
     * // Will generally implement logic to determine where to refund the tip if the intent was not fulfilled
     * // Can also contain specific logic, like custom rewarding on the contract for the fulfiller
     */
    function onSettle(
        bytes32 intentId,
        address asset,
        uint256 amount,
        bytes calldata data,
        bytes32 fulfillmentIndex,
        bool isFulfilled,
        uint256 tipAmount
    ) external;
}

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
    // It performs a Uniswap v2 swap
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
    // if the intent was not fulfilled: it decodes the swap receiver and sends the tip to this address
    function onSettle(
        bytes32 intentId,
        address asset,
        uint256 amount,
        bytes calldata data,
        bytes32 fulfillmentIndex,
        bool isFulfilled,
        uint256 tipAmount
    ) external {
        if (!isFulfilled) {
            (
                address[] memory path,
                uint256 minAmountOut,
                uint256 deadline,
                address receiver
            ) = decodeSwapParams(data);

            IERC20(asset).transfer(receiver, tipAmount);
        }
    }
}
```

In this example:

1. The `data` parameter contains encoded information about the swap: token path, minimum output amount, deadline, and the receiver address
2. The contract uses `transferFrom` to get the tokens from the sender (requires prior approval)
3. The `decodeSwapParams` function extracts all the parameters including the receiver from the encoded bytes
4. The contract then performs the swap using Uniswap V2's router
5. The swapped tokens are sent to the receiver address specified in the data
6. During settlement, if the swap was not fulfilled, the tip is refunded to the swap receiver

## Initiate a Contract Call Intent

To initiate a contract call intent from a source chain, you must use the `initiateCall` function on the corresponding chain's intent contract. This function allows users to specify a target contract on the destination chain that will execute custom logic when the intent is fulfilled.

### Interface

```solidity
/**
 * @dev Initiates a new intent for cross-chain transfer with contract call
 * @param asset The ERC20 token address
 * @param amount Amount to receive on target chain
 * @param targetChain Target chain ID
 * @param receiver Receiver address in bytes format (must implement IntentTarget)
 * @param tip Tip for the fulfiller
 * @param salt Salt for intent ID generation
 * @param data Custom data to be passed to the receiver contract
 * @return intentId The generated intent ID
 */
function initiateCall(
    address asset,
    uint256 amount,
    uint256 targetChain,
    bytes calldata receiver,
    uint256 tip,
    uint256 salt,
    bytes calldata data
) external returns (bytes32);
```

### Parameters

- **asset**: The ERC20 token address you want to transfer
- **amount**: Amount of tokens to be received on the target chain
- **targetChain**: The chain ID of the destination chain
- **receiver**: The address of the contract that will receive the tokens and execute custom logic (encoded as bytes)
- **tip**: Additional incentive for fulfillers
- **salt**: A random value to ensure uniqueness of the intent ID
- **data**: Custom encoded data that will be passed to the receiver contract's `onFulfill` function

### Example: Initiating a Uniswap V2 Swap Intent

Here's an example of how to create an intent that will perform a Uniswap V2 swap on the destination chain:

```solidity
// SPDX-License-Identifier: MIT
pragma solidity 0.8.26;

import "@openzeppelin/contracts/token/ERC20/IERC20.sol";

interface IIntent {
    function initiateCall(
        address asset,
        uint256 amount,
        uint256 targetChain,
        bytes calldata receiver,
        uint256 tip,
        uint256 salt,
        bytes calldata data
    ) external returns (bytes32);
}

contract SwapInitiator {
    function initiateUniswapSwap(
        address intentContract,
        address sourceToken,
        uint256 amount,
        uint256 targetChain,
        address swapperContract, // Address of the CrossChainSwapper contract
        address[] memory swapPath,
        uint256 minAmountOut,
        uint256 deadline,
        address receiver,
        uint256 tip,
        uint256 salt
    ) external returns (bytes32) {
        // Encode the swap parameters that will be passed to the target contract
        bytes memory swapData = abi.encode(
            swapPath,
            minAmountOut,
            deadline,
            receiver
        );
        
        // Approve the intent contract to spend tokens
        IERC20(sourceToken).approve(intentContract, amount);
        
        // Initiate the cross-chain swap intent
        return IIntent(intentContract).initiateCall(
            sourceToken,
            amount,
            targetChain,
            abi.encodePacked(swapperContract), // Convert address to bytes
            tip,
            salt,
            swapData
        );
    }
}
```

### Integration Steps

1. **Deploy a Target Contract**: First, deploy a contract that implements the `IntentTarget` interface on your destination chain (like the `CrossChainSwapper` example).

2. **Approve Tokens**: Before initiating the intent, ensure you've approved the Intent contract to spend your tokens.

3. **Encode Custom Data**: Prepare any data needed for your destination chain operation.

4. **Call initiateCall**: Trigger the cross-chain intent by calling the `initiateCall` function.

5. **Track Intent Status**: You can track the status of your intent using the returned intent ID.
