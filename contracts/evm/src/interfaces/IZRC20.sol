// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

interface IZRC20 {
    function withdrawGasFeeWithGasLimit(uint256 gasLimit) external view returns (address gasZRC20, uint256 gasFee);
} 