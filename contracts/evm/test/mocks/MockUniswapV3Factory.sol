// SPDX-License-Identifier: MIT
pragma solidity ^0.8.26;

import "../../src/interfaces/IUniswapV3Factory.sol";

contract MockUniswapV3Factory is IUniswapV3Factory {
    address public owner;
    mapping(uint24 => int24) public feeAmountTickSpacing;
    mapping(address => mapping(address => mapping(uint24 => address))) public getPool;

    constructor() {
        owner = msg.sender;
        feeAmountTickSpacing[3000] = 60;
    }

    function createPool(
        address tokenA,
        address tokenB,
        uint24 fee
    ) external override returns (address pool) {
        require(tokenA != tokenB, "Same token");
        (address token0, address token1) = tokenA < tokenB ? (tokenA, tokenB) : (tokenB, tokenA);
        require(token0 != address(0), "Zero address");
        require(getPool[token0][token1][fee] == address(0), "Pool exists");
        pool = address(0); // Mock implementation
        getPool[token0][token1][fee] = pool;
        emit PoolCreated(token0, token1, fee, 60, pool, 0); // tickSpacing = 60, sqrtPriceX96 = 0
    }

    function enableFeeAmount(uint24 fee, int24 tickSpacing) external override {
        require(fee > 0, "Fee too low");
        require(tickSpacing > 0, "Tick spacing too low");
        require(feeAmountTickSpacing[fee] == 0, "Fee already enabled");
        feeAmountTickSpacing[fee] = tickSpacing;
        emit EnableFeeAmount(fee, tickSpacing);
    }
} 