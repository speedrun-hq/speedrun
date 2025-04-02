// SPDX-License-Identifier: MIT
pragma solidity 0.8.26;

import "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import "@openzeppelin/contracts/token/ERC20/utils/SafeERC20.sol";
import "./interfaces/IUniswapV3Router.sol";
import "./interfaces/IGateway.sol";
import "./interfaces/IZRC20.sol";
import "./interfaces/ISwap.sol";
import "./utils/PayloadUtils.sol";

/**
 * @title Router
 * @dev Routes CCTX and handles ZRC20 swaps on ZetaChain
 */
contract Router {
    using SafeERC20 for IERC20;

    // Gateway contract address
    address public gateway;
    // Swap module address
    address public swapModule;
    // Admin address
    address public immutable admin;

    // Mapping from chain ID to intent contract address
    mapping(uint256 => address) public intentContracts;

    // Mapping from token name to whether it exists
    mapping(string => bool) private _supportedTokens;
    // Mapping from ZRC20 address to token name
    mapping(address => string) public zrc20ToTokenName;
    // Mapping from token name and chain ID to asset address
    mapping(string => mapping(uint256 => address)) private _tokenAssets;
    // Mapping from token name and chain ID to ZRC20 address
    mapping(string => mapping(uint256 => address)) private _tokenZrc20s;
    // List of supported token names
    string[] public tokenNames;
    // List of chain IDs for each token
    mapping(string => uint256[]) private _tokenChainIds;

    // Event emitted when an intent contract is set
    event IntentContractSet(uint256 indexed chainId, address indexed intentContract);
    // Event emitted when a new token is added
    event TokenAdded(string indexed name);
    // Event emitted when a token association is added
    event TokenAssociationAdded(string indexed name, uint256 indexed chainId, address asset, address zrc20);
    // Event emitted when a token association is updated
    event TokenAssociationUpdated(string indexed name, uint256 indexed chainId, address asset, address zrc20);
    // Event emitted when a token association is removed
    event TokenAssociationRemoved(string indexed name, uint256 indexed chainId);
    // Event emitted when an intent settlement is forwarded
    event IntentSettlementForwarded(
        bytes indexed sender,
        uint256 indexed sourceChain,
        uint256 indexed targetChain,
        address zrc20,
        uint256 amount,
        uint256 tip
    );

    // Error for unauthorized access
    error Unauthorized(address caller);

    /**
     * @dev Constructor that sets the gateway and swap module addresses
     * @param _gateway The address of the gateway contract
     * @param _swapModule The address of the swap module contract
     */
    constructor(address _gateway, address _swapModule) {
        require(_gateway != address(0), "Invalid gateway address");
        require(_swapModule != address(0), "Invalid swap module address");

        gateway = _gateway;
        swapModule = _swapModule;
        admin = msg.sender;
    }

    /**
     * @dev Modifier to restrict access to the admin
     */
    modifier onlyAdmin() {
        if (msg.sender != admin) {
            revert Unauthorized(msg.sender);
        }
        _;
    }

    modifier onlyGateway() {
        require(msg.sender == gateway, "Only gateway can call this function");
        _;
    }

    /**
     * @dev Handles incoming messages from the gateway
     * @param context The message context containing sender and chain information
     * @param zrc20 The ZRC20 token address
     * @param amount The amount of tokens
     * @param message The encoded message containing intent payload
     */
    function onCall(
        IGateway.ZetaChainMessageContext calldata context,
        address zrc20,
        uint256 amount,
        bytes calldata message
    ) external onlyGateway {
        // Verify the call is coming from the intent contract for this chain
        require(intentContracts[context.chainID] == context.senderEVM, "Call must be from intent contract");

        // Decode intent payload
        PayloadUtils.IntentPayload memory intentPayload = PayloadUtils.decodeIntentPayload(message);

        // Get token association for target chain
        (address targetAsset, address targetZRC20, uint256 chainIdValue) = getTokenAssociation(zrc20, intentPayload.targetChain);

        // Get intent contract on target chain
        address intentContract = intentContracts[intentPayload.targetChain];
        require(intentContract != address(0), "Intent contract not set for target chain");

        // Get gas fee info from target ZRC20
        (address gasZRC20, uint256 gasFee) = IZRC20(targetZRC20).withdrawGasFeeWithGasLimit(100000);

        // Approve swap module to spend tokens
        IERC20(zrc20).approve(swapModule, amount);

        // Perform swap through swap module
        uint256 amountOut = ISwap(swapModule).swap(zrc20, targetZRC20, amount, gasZRC20, gasFee);

        // Calculate slippage difference and adjust tip accordingly
        uint256 slippageAndFeeCost = amount - amountOut;
        require(intentPayload.tip > slippageAndFeeCost, "Provided tip doesn't cover slippage and withdraw fee cost");
        uint256 tipAfterSwap = intentPayload.tip - slippageAndFeeCost;

        // Convert receiver from bytes to address
        address receiverAddress = PayloadUtils.bytesToAddress(intentPayload.receiver);

        // Encode settlement payload
        bytes memory settlementPayload = PayloadUtils.encodeSettlementPayload(
            intentPayload.intentId,
            intentPayload.amount,
            targetAsset,
            receiverAddress,
            tipAfterSwap
        );

        // Prepare call options
        IGateway.CallOptions memory callOptions = IGateway.CallOptions({
            gasLimit: 100000,
            isArbitraryCall: false
        });

        // Prepare revert options
        IGateway.RevertOptions memory revertOptions = IGateway.RevertOptions({
            revertAddress: address(0),
            callOnRevert: false,
            abortAddress: address(0),
            revertMessage: "",
            onRevertGasLimit: 0
        });

        // Approve gateway to spend tokens
        IERC20(targetZRC20).approve(gateway, amountOut);
        IERC20(gasZRC20).approve(gateway, gasFee);

        // Call gateway to withdraw and call intent contract
        IGateway(gateway).withdrawAndCall(
            abi.encodePacked(intentContract),
            amountOut,
            targetZRC20,
            settlementPayload,
            callOptions,
            revertOptions
        );

        emit IntentSettlementForwarded(
            context.sender,
            context.chainID,
            intentPayload.targetChain,
            zrc20,
            amount,
            tipAfterSwap
        );
    }

    /**
     * @dev Sets the intent contract address for a specific chain
     * @param chainId The chain ID to set the intent contract for
     * @param intentContract The address of the intent contract
     */
    function setIntentContract(uint256 chainId, address intentContract) public onlyAdmin {
        require(intentContract != address(0), "Invalid intent contract address");
        intentContracts[chainId] = intentContract;
        emit IntentContractSet(chainId, intentContract);
    }

    /**
     * @dev Gets the intent contract address for a specific chain
     * @param chainId The chain ID to get the intent contract for
     * @return The address of the intent contract
     */
    function getIntentContract(uint256 chainId) public view returns (address) {
        return intentContracts[chainId];
    }

    /**
     * @dev Adds a new supported token
     * @param name The name of the token (e.g., "USDC")
     */
    function addToken(string calldata name) public onlyAdmin {
        require(bytes(name).length > 0, "Token name cannot be empty");
        require(!_supportedTokens[name], "Token already exists");
        
        _supportedTokens[name] = true;
        tokenNames.push(name);
        emit TokenAdded(name);
    }

    /**
     * @dev Adds a new token association
     * @param name The name of the token
     * @param chainId The chain ID where the asset exists
     * @param asset The ERC20 address on the source chain
     * @param zrc20 The ZRC20 address on ZetaChain
     */
    function addTokenAssociation(
        string calldata name,
        uint256 chainId,
        address asset,
        address zrc20
    ) public onlyAdmin {
        require(_supportedTokens[name], "Token does not exist");
        require(asset != address(0), "Invalid asset address");
        require(zrc20 != address(0), "Invalid ZRC20 address");
        require(_tokenAssets[name][chainId] == address(0), "Association already exists");
        
        _tokenAssets[name][chainId] = asset;
        _tokenZrc20s[name][chainId] = zrc20;
        _tokenChainIds[name].push(chainId);
        zrc20ToTokenName[zrc20] = name;
        
        emit TokenAssociationAdded(name, chainId, asset, zrc20);
    }

    /**
     * @dev Updates an existing token association
     * @param name The name of the token
     * @param chainId The chain ID where the asset exists
     * @param asset The new ERC20 address on the source chain
     * @param zrc20 The new ZRC20 address on ZetaChain
     */
    function updateTokenAssociation(
        string calldata name,
        uint256 chainId,
        address asset,
        address zrc20
    ) public onlyAdmin {
        require(_supportedTokens[name], "Token does not exist");
        require(asset != address(0), "Invalid asset address");
        require(zrc20 != address(0), "Invalid ZRC20 address");
        require(_tokenAssets[name][chainId] != address(0), "Association does not exist");
        
        _tokenAssets[name][chainId] = asset;
        _tokenZrc20s[name][chainId] = zrc20;
        zrc20ToTokenName[zrc20] = name;
        
        emit TokenAssociationUpdated(name, chainId, asset, zrc20);
    }

    /**
     * @dev Removes a token association
     * @param name The name of the token
     * @param chainId The chain ID to remove the association for
     */
    function removeTokenAssociation(
        string calldata name,
        uint256 chainId
    ) public onlyAdmin {
        require(_supportedTokens[name], "Token does not exist");
        require(_tokenAssets[name][chainId] != address(0), "Association does not exist");
        
        delete _tokenAssets[name][chainId];
        delete _tokenZrc20s[name][chainId];
        
        // Remove chainId from the array
        uint256[] storage chainIds = _tokenChainIds[name];
        for (uint256 i = 0; i < chainIds.length; i++) {
            if (chainIds[i] == chainId) {
                chainIds[i] = chainIds[chainIds.length - 1];
                chainIds.pop();
                break;
            }
        }
        
        emit TokenAssociationRemoved(name, chainId);
    }

    /**
     * @dev Gets the token association for a specific chain
     * @param zrc20 The ZRC20 address on ZetaChain
     * @param chainId The chain ID to get the association for
     * @return asset The ERC20 address on the source chain
     * @return zrc20Addr The ZRC20 address on ZetaChain
     * @return chainIdValue The chain ID where the asset exists
     */
    function getTokenAssociation(
        address zrc20,
        uint256 chainId
    ) public view returns (
        address asset,
        address zrc20Addr,
        uint256 chainIdValue
    ) {
        string memory name = zrc20ToTokenName[zrc20];
        require(_supportedTokens[name], "Token does not exist");
        require(_tokenAssets[name][chainId] != address(0), "Association does not exist");
        
        return (_tokenAssets[name][chainId], _tokenZrc20s[name][chainId], chainId);
    }

    /**
     * @dev Gets all token associations for a specific token
     * @param name The name of the token
     * @return chainIds Array of chain IDs
     * @return assets Array of asset addresses
     * @return zrc20s Array of ZRC20 addresses
     */
    function getTokenAssociations(string calldata name) public view returns (
        uint256[] memory chainIds,
        address[] memory assets,
        address[] memory zrc20s
    ) {
        require(_supportedTokens[name], "Token does not exist");
        
        chainIds = _tokenChainIds[name];
        uint256 length = chainIds.length;
        
        assets = new address[](length);
        zrc20s = new address[](length);
        
        for (uint256 i = 0; i < length; i++) {
            uint256 chainId = chainIds[i];
            assets[i] = _tokenAssets[name][chainId];
            zrc20s[i] = _tokenZrc20s[name][chainId];
        }
    }

    /**
     * @dev Gets all supported token names
     * @return Array of token names
     */
    function getSupportedTokens() public view returns (string[] memory) {
        return tokenNames;
    }

    /**
     * @dev Checks if a token exists
     * @param name The name of the token
     * @return Whether the token exists
     */
    function isTokenSupported(string calldata name) public view returns (bool) {
        return _supportedTokens[name];
    }
} 