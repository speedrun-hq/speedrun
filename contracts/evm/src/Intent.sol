// SPDX-License-Identifier: MIT
pragma solidity ^0.8.26;

import "@openzeppelin/contracts-upgradeable/proxy/utils/Initializable.sol";
import "@openzeppelin/contracts-upgradeable/proxy/utils/UUPSUpgradeable.sol";
import "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";
import "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import "./interfaces/IGateway.sol";
import "./utils/PayloadUtils.sol";

/**
 * @title Intent
 * @dev Handles intent-based transfers across chains
 */
contract Intent is Initializable, UUPSUpgradeable, OwnableUpgradeable {
    // Counter for generating unique intent IDs
    uint256 private _intentCounter;

    // Gateway contract address
    address public gateway;

    // Router contract address on ZetaChain
    address public router;

    // Event emitted when a new intent is created
    event IntentInitiated(
        bytes32 indexed intentId,
        address indexed asset,
        uint256 amount,
        uint256 targetChain,
        bytes receiver,
        uint256 tip,
        uint256 salt
    );

    /// @custom:oz-upgrades-unsafe-allow constructor
    constructor() {
        _disableInitializers();
    }

    function initialize(address _gateway, address _router) public initializer {
        __Ownable_init(msg.sender);
        __UUPSUpgradeable_init();
        
        gateway = _gateway;
        router = _router;
    }

    function _authorizeUpgrade(address newImplementation) internal override onlyOwner {}

    /**
     * @dev Initiates a new intent for cross-chain transfer
     * @param asset The ERC20 token address
     * @param amount Amount to receive on target chain
     * @param targetChain Target chain ID
     * @param receiver Receiver address in bytes format
     * @param tip Tip for the fulfiller
     * @param salt Salt for intent ID generation
     * @return intentId The generated intent ID
     */
    function initiate(
        address asset,
        uint256 amount,
        uint256 targetChain,
        bytes calldata receiver,
        uint256 tip,
        uint256 salt
    ) external returns (bytes32) {
        // Calculate total amount to transfer (amount + tip)
        uint256 totalAmount = amount + tip;

        // Transfer ERC20 tokens from sender to this contract
        IERC20(asset).transferFrom(msg.sender, address(this), totalAmount);

        // Approve gateway to spend the tokens
        IERC20(asset).approve(gateway, totalAmount);

        // Generate intent ID using keccak256 hash of counter and salt
        bytes32 intentId = keccak256(abi.encodePacked(_intentCounter, salt));
        
        // Increment counter
        _intentCounter++;

        // Create payload for crosschain transaction
        bytes memory payload = PayloadUtils.encodeIntentPayload(
            intentId,
            amount,
            tip,
            targetChain,
            receiver
        );

        // Create empty revert options
        IGateway.RevertOptions memory revertOptions = IGateway.RevertOptions({
            revertAddress: msg.sender, // in case of revert, the funds are directly sent back to the sender
            callOnRevert: false,
            abortAddress: address(0),
            revertMessage: "",
            onRevertGasLimit: 0
        });

        // Call gateway to initiate crosschain transaction
        IGateway(gateway).depositAndCall(
            router, // receiver is the router on ZetaChain
            totalAmount, // transfer amount + tip
            asset,
            payload,
            revertOptions
        );

        // Emit event
        emit IntentInitiated(
            intentId,
            asset,
            amount,
            targetChain,
            receiver,
            tip,
            salt
        );

        return intentId;
    }

    // TODO: Add intent management functions
    // - fulfill
    // - complete
    // - onCall
    // - onRevert
} 