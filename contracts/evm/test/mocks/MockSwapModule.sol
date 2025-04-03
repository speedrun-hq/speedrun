// SPDX-License-Identifier: MIT
pragma solidity 0.8.26;

import "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import "../../src/interfaces/ISwap.sol";

contract MockSwapModule is ISwap {
    // Control parameters to simulate different swap behaviors
    uint256 public slippage = 0; // 0 = no slippage (1:1 swap), higher means more slippage
    
    function setSlippage(uint256 _slippage) external {
        slippage = _slippage;
    }
    
    function swap(
        address tokenIn,
        address tokenOut,
        uint256 amountIn,
        address gasZRC20,
        uint256 gasFee
    ) external override returns (uint256 amountOut) {
        // Transfer tokens from sender to this contract
        IERC20(tokenIn).transferFrom(msg.sender, address(this), amountIn);
        
        // Calculate amount out based on slippage settings
        uint256 slippageCost = (amountIn * slippage) / 10000; // slippage in basis points (e.g., 100 = 1%)
        amountOut = amountIn - slippageCost - gasFee;
        
        // Transfer tokens back to the sender
        IERC20(tokenOut).transfer(msg.sender, amountOut);
        IERC20(gasZRC20).transfer(msg.sender, gasFee);
        
        return amountOut;
    }
} 