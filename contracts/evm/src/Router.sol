// SPDX-License-Identifier: MIT
pragma solidity ^0.8.26;

import "@openzeppelin/contracts-upgradeable/proxy/utils/Initializable.sol";
import "@openzeppelin/contracts-upgradeable/proxy/utils/UUPSUpgradeable.sol";
import "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";
import "@openzeppelin/contracts-upgradeable/access/AccessControlUpgradeable.sol";

/**
 * @title Router
 * @dev Routes CCTX and handles ZRC20 swaps on ZetaChain
 */
contract Router is Initializable, UUPSUpgradeable, OwnableUpgradeable, AccessControlUpgradeable {
    // Mapping from chain ID to intent contract address
    mapping(uint256 => address) public intentContracts;

    // Event emitted when an intent contract is set
    event IntentContractSet(uint256 indexed chainId, address indexed intentContract);

    /// @custom:oz-upgrades-unsafe-allow constructor
    constructor() {
        _disableInitializers();
    }

    function initialize() public initializer {
        __Ownable_init(msg.sender);
        __UUPSUpgradeable_init();
        __AccessControl_init();

        // Grant the default admin role to the deployer
        _grantRole(DEFAULT_ADMIN_ROLE, msg.sender);
    }

    function _authorizeUpgrade(address newImplementation) internal override onlyOwner {}

    /**
     * @dev Sets the intent contract address for a specific chain
     * @param chainId The chain ID to set the intent contract for
     * @param intentContract The address of the intent contract
     */
    function setIntentContract(uint256 chainId, address intentContract) external onlyRole(DEFAULT_ADMIN_ROLE) {
        require(intentContract != address(0), "Invalid intent contract address");
        intentContracts[chainId] = intentContract;
        emit IntentContractSet(chainId, intentContract);
    }

    /**
     * @dev Gets the intent contract address for a specific chain
     * @param chainId The chain ID to get the intent contract for
     * @return The address of the intent contract
     */
    function getIntentContract(uint256 chainId) external view returns (address) {
        return intentContracts[chainId];
    }

    // TODO: Add routing functions
    // - route CCTX
    // - ZRC20 swap
    // - registry management
    // - onCall
    // - onRevert
    // - onAbort
} 