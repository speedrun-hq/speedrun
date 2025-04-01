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

    /**
     * @dev Struct for settlement payload
     */
    struct SettlementPayload {
        bytes32 intentId;
        uint256 amount;
        address asset;
        address receiver;
        uint256 tip;
    }

    /**
     * @dev Encodes settlement data into a payload
     */
    function encodeSettlementPayload(
        bytes32 intentId,
        uint256 amount,
        address asset,
        address receiver,
        uint256 tip
    ) internal pure returns (bytes memory) {
        return abi.encode(
            intentId,
            amount,
            asset,
            receiver,
            tip
        );
    }

    /**
     * @dev Decodes settlement payload back into data
     */
    function decodeSettlementPayload(bytes memory payload) internal pure returns (SettlementPayload memory) {
        (
            bytes32 intentId,
            uint256 amount,
            address asset,
            address receiver,
            uint256 tip
        ) = abi.decode(payload, (bytes32, uint256, address, address, uint256));

        return SettlementPayload({
            intentId: intentId,
            amount: amount,
            asset: asset,
            receiver: receiver,
            tip: tip
        });
    }

    /**
     * @dev Computes a unique index for a fulfillment
     * @param intentId The ID of the intent
     * @param asset The ERC20 token address
     * @param amount Amount to transfer
     * @param receiver Receiver address
     * @return The computed fulfillment index
     */
    function computeFulfillmentIndex(
        bytes32 intentId,
        address asset,
        uint256 amount,
        address receiver
    ) internal pure returns (bytes32) {
        return keccak256(abi.encodePacked(
            intentId,
            asset,
            amount,
            receiver
        ));
    }

    /**
     * @dev Converts bytes to address
     * @param data The bytes to convert
     * @return The converted address
     */
    function bytesToAddress(bytes memory data) internal pure returns (address) {
        require(data.length >= 20, "Invalid address length");
        address addr;
        assembly {
            addr := and(mload(add(data, 20)), 0xffffffffffffffffffffffffffffffffffffffff)
        }
        return addr;
    }
} 