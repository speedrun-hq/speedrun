// SPDX-License-Identifier: MIT
pragma solidity ^0.8.26;

interface IUniswapV3Factory {
    event PoolCreated(
        address indexed token0,
        address indexed token1,
        uint24 indexed fee,
        int24 tickSpacing,
        address pool,
        uint160 sqrtPriceX96
    );

    event EnableFeeAmount(uint24 indexed fee, int24 indexed tickSpacing);

    function owner() external view returns (address);
    function feeAmountTickSpacing(uint24 fee) external view returns (int24);
    function getPool(
        address tokenA,
        address tokenB,
        uint24 fee
    ) external view returns (address pool);
    function createPool(
        address tokenA,
        address tokenB,
        uint24 fee
    ) external returns (address pool);
    function enableFeeAmount(uint24 fee, int24 tickSpacing) external;
} 