// SPDX-License-Identifier: MIT
pragma solidity ^0.8.26;

interface IGateway {
    struct RevertOptions {
        address revertAddress;
        bool callOnRevert;
        address abortAddress;
        bytes revertMessage;
        uint256 onRevertGasLimit;
    }

    /**
     * @notice Deposits ERC20 tokens to the custody or connector contract and calls an omnichain smart contract.
     * @param receiver Address of the receiver.
     * @param amount Amount of tokens to deposit.
     * @param asset Address of the ERC20 token.
     * @param payload Calldata to pass to the call.
     * @param revertOptions Revert options.
     */
    function depositAndCall(
        address receiver,
        uint256 amount,
        address asset,
        bytes calldata payload,
        RevertOptions calldata revertOptions
    ) external;
} 