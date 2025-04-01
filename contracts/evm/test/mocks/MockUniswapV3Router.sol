// SPDX-License-Identifier: MIT
pragma solidity ^0.8.26;

import "../../src/interfaces/ISwapRouter.sol";

contract MockUniswapV3Router is ISwapRouter {
    function exactInputSingle(ExactInputSingleParams calldata params) external payable override returns (uint256 amountOut) {
        // Mock implementation - return input amount
        return params.amountIn;
    }

    function exactInput(ExactInputParams calldata params) external payable override returns (uint256 amountOut) {
        // Mock implementation - return input amount
        return params.amountIn;
    }
} 