# Aerodrome

The Aerodrome Module enables near-instant cross-chain token swaps using the [Aerodrome](https://aerodrome.finance) decentralized exchange on Base.

## Overview

Aerodrome is a popular liquidity protocol on Base that offers both stable and volatile pools for swapping tokens. This module allows users to:

1. Initiate a token transfer from any supported source chain
2. Automatically swap the received tokens on Aerodrome when they arrive on Base
3. Receive the swapped tokens at a specified address

## Addresses

- **Module on Base**: `0x30e13787a90De8Ab4831c35e1d2c64783144ab7a`
- **Initiator on Arbitrum**: `0xbAFeFC473e886557A7Bc8a283EdF4Cf47a3E17f9`

## Interface

The Aerodrome module exposes the following function for initiating cross-chain swaps:

```solidity
/**
 * @dev Initiates a cross-chain swap on Aerodrome
 * @param asset The source token address
 * @param amount Amount to swap
 * @param tip Tip for the fulfiller
 * @param salt Salt for intent ID generation
 * @param gasLimit Gas limit for the target chain transaction
 * @param path Array of token addresses for the swap path
 * @param stableFlags Array of booleans indicating if pools are stable or volatile
 * @param minAmountOut Minimum output amount
 * @param deadline Transaction deadline
 * @param receiver Address that will receive the swapped tokens
 * @return intentId The generated intent ID
 */
function initiateAerodromeSwap(
    address asset,
    uint256 amount,
    uint256 tip,
    uint256 salt,
    uint256 gasLimit,
    address[] calldata path,
    bool[] calldata stableFlags,
    uint256 minAmountOut,
    uint256 deadline,
    address receiver
) external returns (bytes32)
```

### Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `asset` | `address` | The source token address to be transferred and swapped |
| `amount` | `uint256` | Amount of the source token to transfer |
| `tip` | `uint256` | Additional incentive for fulfillers |
| `salt` | `uint256` | Random value to ensure uniqueness of the intent ID |
| `gasLimit` | `uint256` | Gas limit for the swap transaction on Base |
| `path` | `address[]` | Array of token addresses representing the swap path on Aerodrome |
| `stableFlags` | `bool[]` | Array of boolean flags indicating whether each pool in the path is stable (`true`) or volatile (`false`) |
| `minAmountOut` | `uint256` | Minimum amount of output tokens to receive, protecting against slippage |
| `deadline` | `uint256` | Timestamp after which the swap transaction will revert |
| `receiver` | `address` | Address that will receive the swapped tokens |

## How It Works

1. **Source Chain**: Call `initiateAerodromeSwap` on the source chain intent contract
2. **Intent Creation**: The system generates a unique intent ID and emits an intent event
3. **Fulfillment**: A fulfiller picks up the intent and releases the equivalent tokens on Base
4. **Swap Execution**: The Aerodrome module's `onFulfill` function executes the swap on Aerodrome
5. **Token Delivery**: The output tokens from the swap are sent to the specified receiver address

## Supported Tokens

The Aerodrome module supports the following token as source:

- `USDC` (`0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913`)

## Notes and Limitations

1. **Gas Estimation**: Set an appropriate `gasLimit` to ensure the swap can be executed successfully on Base
2. **Slippage Protection**: Set a reasonable `minAmountOut` to protect against price movements
3. **Path Length**: The swap path can include multiple tokens for complex routes
4. **StableFlags Length**: The `stableFlags` array must have exactly one less element than the `path` array

## Example Usage

The following example demonstrates how to use the Aerodrome module to perform a near-instant cross-chain swap from USDC on Arbitrum to WETH on Base. This allows users to bridge tokens and swap them in a single transaction, significantly improving UX compared to traditional bridge-then-swap workflows.

```solidity
// SPDX-License-Identifier: MIT
pragma solidity 0.8.26;

import "@openzeppelin/contracts/token/ERC20/IERC20.sol";

/**
 * @title IAerodromeInitiator
 * @dev Interface for the AerodromeInitiator contract
 */
interface IAerodromeInitiator {
    function initiateAerodromeSwap(
        address asset,
        uint256 amount,
        uint256 tip,
        uint256 salt,
        uint256 gasLimit,
        address[] calldata path,
        bool[] calldata stableFlags,
        uint256 minAmountOut,
        uint256 deadline,
        address receiver
    ) external returns (bytes32);
}

/**
 * @title AerodromeSwapper
 * @dev Example contract for swapping USDC from Arbitrum to WETH on Base via Aerodrome
 */
contract AerodromeSwapper {
    // Contract addresses
    address public constant AERODROME_INITIATOR = 0xbAFeFC473e886557A7Bc8a283EdF4Cf47a3E17f9;

    // Token addresses
    address public constant USDC_BASE = 0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913;
    address public constant USDC_ARB = 0xaf88d065e77c8cC2239327C5EDb3A432268e5831;
    address public constant WETH = 0x4200000000000000000000000000000000000006;

    /**
     * @dev Initiates a cross-chain swap from USDC on Arbitrum to WETH on Base
     * @param amount Amount of USDC to swap
     * @param tip Tip for the fulfiller
     * @param minAmountOut Minimum amount of WETH to receive
     * @param receiver Address that will receive the WETH on Base
     * @return intentId The generated intent ID
     */
    function initiateSwap(
        uint256 amount,
        uint256 tip,
        uint256 minAmountOut,
        address receiver
    ) external returns (bytes32) {
        // Set default parameters
        uint256 salt = uint256(keccak256(abi.encodePacked(block.timestamp, msg.sender)));
        uint256 gasLimit = 600000;
        uint256 deadline = type(uint256).max;
        
        // Initialize swap path
        address[] memory path = new address[](2);
        path[0] = USDC_BASE;
        path[1] = WETH;

        // Initialize stable flags
        bool[] memory stableFlags = new bool[](1);
        stableFlags[0] = false; // USDC/WETH pool is not stable

        // Approve tokens for AerodromeInitiator
        IERC20(USDC_ARB).approve(AERODROME_INITIATOR, amount + tip);

        // Initiate the swap via the AerodromeInitiator contract
        return IAerodromeInitiator(AERODROME_INITIATOR).initiateAerodromeSwap(
            USDC_ARB,
            amount,
            tip,
            salt,
            gasLimit,
            path,
            stableFlags,
            minAmountOut,
            deadline,
            receiver
        );
    }
}
```

