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

    // Mapping to track fulfillments
    mapping(bytes32 => address) public fulfillments;

    // Struct to track settlement status
    struct Settlement {
        bool settled;
        bool fulfilled;
        uint256 paidTip;
        address fulfiller;
    }

    // Mapping to track settlements
    mapping(bytes32 => Settlement) public settlements;

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

    // Event emitted when an intent is fulfilled
    event IntentFulfilled(
        bytes32 indexed intentId,
        address indexed asset,
        uint256 amount,
        address indexed receiver
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

        // Create revert options
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

    /**
     * @dev Fulfills an intent by transferring tokens to the receiver
     * @param intentId The ID of the intent to fulfill
     * @param asset The ERC20 token address
     * @param amount Amount to transfer
     * @param receiver Receiver address
     */
    function fulfill(
        bytes32 intentId,
        address asset,
        uint256 amount,
        address receiver
    ) external {
        // Compute the fulfillment index
        bytes32 fulfillmentIndex = PayloadUtils.computeFulfillmentIndex(
            intentId,
            asset,
            amount,
            receiver
        );

        // Check if intent is already fulfilled with these parameters
        require(fulfillments[fulfillmentIndex] == address(0), "Intent already fulfilled with these parameters");

        // Transfer tokens from this contract to the receiver
        IERC20(asset).transfer(receiver, amount);

        // Register the fulfillment
        fulfillments[fulfillmentIndex] = msg.sender;

        // Emit event
        emit IntentFulfilled(
            intentId,
            asset,
            amount,
            receiver
        );
    }

    /**
     * @dev Internal function to settle an intent
     * @param intentId The ID of the intent to settle
     * @param asset The ERC20 token address
     * @param amount Amount to transfer
     * @param receiver Receiver address
     * @param tip Tip for the fulfiller
     */
    function _settle(
        bytes32 intentId,
        address asset,
        uint256 amount,
        address receiver,
        uint256 tip
    ) internal {
        // Compute the fulfillment index
        bytes32 fulfillmentIndex = PayloadUtils.computeFulfillmentIndex(
            intentId,
            asset,
            amount,
            receiver
        );

        // Get the fulfiller if it exists
        address fulfiller = fulfillments[fulfillmentIndex];
        bool fulfilled = fulfiller != address(0);

        // Create settlement record
        Settlement storage settlement = settlements[fulfillmentIndex];
        settlement.settled = true;
        settlement.fulfilled = fulfilled;
        settlement.fulfiller = fulfiller;

        // If there's a fulfiller, transfer the tip
        if (fulfilled) {
            IERC20(asset).transfer(fulfiller, amount + tip);
            settlement.paidTip = tip;
        }
    }

    // TODO: Add intent management functions
    // - complete
    // - onCall
    // - onRevert
} 