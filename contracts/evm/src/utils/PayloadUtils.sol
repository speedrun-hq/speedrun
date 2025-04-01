// SPDX-License-Identifier: MIT
pragma solidity ^0.8.26;

/**
 * @title PayloadUtils
 * @dev Utility functions for encoding and decoding payloads for cross-chain transactions
 */
library PayloadUtils {
    struct IntentPayload {
        bytes32 intentId;
        uint256 amount;
        uint256 tip;
        uint256 targetChain;
        bytes receiver;
    }

    /**
     * @dev Encodes intent data into a payload for cross-chain transaction
     */
    function encodeIntentPayload(
        bytes32 intentId,
        uint256 amount,
        uint256 tip,
        uint256 targetChain,
        bytes calldata receiver
    ) internal pure returns (bytes memory) {
        return abi.encode(
            intentId,
            amount,
            tip,
            targetChain,
            receiver
        );
    }

    /**
     * @dev Decodes payload back into intent data
     */
    function decodeIntentPayload(bytes memory payload) internal pure returns (IntentPayload memory) {
        (
            bytes32 intentId,
            uint256 amount,
            uint256 tip,
            uint256 targetChain,
            bytes memory receiver
        ) = abi.decode(payload, (bytes32, uint256, uint256, uint256, bytes));

        return IntentPayload({
            intentId: intentId,
            amount: amount,
            tip: tip,
            targetChain: targetChain,
            receiver: receiver
        });
    }
} 