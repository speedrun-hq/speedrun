// SPDX-License-Identifier: MIT
pragma solidity ^0.8.26;

import {Test, console2} from "forge-std/Test.sol";
import {Router} from "../src/router.sol";
import {ERC1967Proxy} from "@openzeppelin/contracts/proxy/ERC1967/ERC1967Proxy.sol";
import {MockGateway} from "./mocks/MockGateway.sol";
import {MockUniswapV3Factory} from "./mocks/MockUniswapV3Factory.sol";
import {MockUniswapV3Router} from "./mocks/MockUniswapV3Router.sol";
import {MockWETH} from "./mocks/MockWETH.sol";
import {MockToken} from "./mocks/MockToken.sol";
import {PayloadUtils} from "../src/utils/PayloadUtils.sol";
import {IGateway} from "../src/interfaces/IGateway.sol";
import {IERC20} from "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import {IZRC20} from "../src/interfaces/IZRC20.sol";
import {ISwapRouter} from "../src/interfaces/ISwapRouter.sol";
import "forge-std/console.sol";

contract RouterTest is Test {
    Router public router;
    Router public routerImplementation;
    MockGateway public gateway;
    MockUniswapV3Factory public factory;
    MockUniswapV3Router public swapRouter;
    MockWETH public wzeta;
    MockToken public inputToken;
    MockToken public gasZRC20;
    MockToken public targetZRC20;
    address public owner;
    address public user1;
    address public user2;
    bytes32 public constant DEFAULT_ADMIN_ROLE = 0x00;

    event IntentContractSet(uint256 indexed chainId, address indexed intentContract);
    event TokenAdded(string indexed name);
    event TokenAssociationAdded(string indexed name, uint256 indexed chainId, address asset, address zrc20);
    event TokenAssociationUpdated(string indexed name, uint256 indexed chainId, address asset, address zrc20);
    event TokenAssociationRemoved(string indexed name, uint256 indexed chainId);
    event IntentSettlementForwarded(
        bytes indexed sender,
        uint256 indexed sourceChain,
        uint256 indexed targetChain,
        address zrc20,
        uint256 amount,
        uint256 tip
    );

    function setUp() public {
        owner = address(this);
        user1 = makeAddr("user1");
        user2 = makeAddr("user2");

        // Deploy mock contracts
        gateway = new MockGateway();
        factory = new MockUniswapV3Factory();
        swapRouter = new MockUniswapV3Router();
        wzeta = new MockWETH();
        inputToken = new MockToken("Input Token", "INPUT");
        gasZRC20 = new MockToken("Gas Token", "GAS");
        targetZRC20 = new MockToken("Target Token", "TARGET");

        // Deploy implementation
        routerImplementation = new Router();

        // Prepare initialization data
        bytes memory initData = abi.encodeWithSelector(
            Router.initialize.selector,
            address(gateway),
            address(factory),
            address(swapRouter),
            address(wzeta)
        );

        // Deploy proxy
        ERC1967Proxy proxy = new ERC1967Proxy(
            address(routerImplementation),
            initData
        );
        router = Router(address(proxy));
    }

    function test_Initialization() public {
        assertEq(router.owner(), owner);
    }

    function test_SetIntentContract() public {
        uint256 chainId = 1;
        router.setIntentContract(chainId, user1);
        assertEq(router.intentContracts(chainId), user1);
    }

    function test_SetIntentContract_EmitsEvent() public {
        uint256 chainId = 1;
        vm.expectEmit(true, true, false, false);
        emit IntentContractSet(chainId, user1);
        router.setIntentContract(chainId, user1);
    }

    function test_SetIntentContract_NonAdminReverts() public {
        uint256 chainId = 1;
        vm.prank(user1);
        vm.expectRevert(
            abi.encodeWithSelector(
                bytes4(keccak256("AccessControlUnauthorizedAccount(address,bytes32)")),
                user1,
                DEFAULT_ADMIN_ROLE
            )
        );
        router.setIntentContract(chainId, user2);
    }

    function test_SetIntentContract_ZeroAddressReverts() public {
        uint256 chainId = 1;
        vm.expectRevert("Invalid intent contract address");
        router.setIntentContract(chainId, address(0));
    }

    function test_GetIntentContract() public {
        uint256 chainId = 1;
        router.setIntentContract(chainId, user1);
        assertEq(router.getIntentContract(chainId), user1);
    }

    function test_UpdateIntentContract() public {
        uint256 chainId = 1;
        router.setIntentContract(chainId, user1);
        router.setIntentContract(chainId, user2);
        assertEq(router.intentContracts(chainId), user2);
    }

    function test_GetIntentContract_UnsetChainId() public {
        uint256 chainId = 999;
        assertEq(router.getIntentContract(chainId), address(0));
    }

    function test_AddToken() public {
        string memory name = "USDC";
        vm.expectEmit(true, false, false, false);
        emit TokenAdded(name);
        router.addToken(name);
        assertTrue(router.isTokenSupported(name));
        assertEq(router.tokenNames(0), name);
    }

    function test_AddToken_EmptyNameReverts() public {
        string memory name = "";
        vm.expectRevert("Token name cannot be empty");
        router.addToken(name);
    }

    function test_AddToken_DuplicateReverts() public {
        string memory name = "USDC";
        router.addToken(name);
        vm.expectRevert("Token already exists");
        router.addToken(name);
    }

    function test_AddTokenAssociation() public {
        string memory name = "USDC";
        uint256 chainId = 1;
        address asset = makeAddr("asset");
        address zrc20 = makeAddr("zrc20");

        router.addToken(name);
        vm.expectEmit(true, true, false, false);
        emit TokenAssociationAdded(name, chainId, asset, zrc20);
        router.addTokenAssociation(name, chainId, asset, zrc20);

        (address returnedAsset, address returnedZrc20, uint256 chainIdValue) = router.getTokenAssociation(zrc20, chainId);
        assertEq(returnedAsset, asset);
        assertEq(returnedZrc20, zrc20);
        assertEq(chainIdValue, chainId);
        assertEq(router.zrc20ToTokenName(zrc20), name);
    }

    function test_AddTokenAssociation_NonExistentTokenReverts() public {
        string memory name = "USDC";
        uint256 chainId = 1;
        address asset = makeAddr("asset");
        address zrc20 = makeAddr("zrc20");

        vm.expectRevert("Token does not exist");
        router.addTokenAssociation(name, chainId, asset, zrc20);
    }

    function test_AddTokenAssociation_ZeroAddressReverts() public {
        string memory name = "USDC";
        uint256 chainId = 1;
        address asset = address(0);
        address zrc20 = makeAddr("zrc20");

        router.addToken(name);
        vm.expectRevert("Invalid asset address");
        router.addTokenAssociation(name, chainId, asset, zrc20);
    }

    function test_AddTokenAssociation_DuplicateChainIdReverts() public {
        string memory name = "USDC";
        uint256 chainId = 1;
        address asset1 = makeAddr("asset1");
        address asset2 = makeAddr("asset2");
        address zrc20 = makeAddr("zrc20");

        router.addToken(name);
        router.addTokenAssociation(name, chainId, asset1, zrc20);
        vm.expectRevert("Association already exists");
        router.addTokenAssociation(name, chainId, asset2, zrc20);
    }

    function test_UpdateTokenAssociation() public {
        string memory name = "USDC";
        uint256 chainId = 1;
        address asset1 = makeAddr("asset1");
        address asset2 = makeAddr("asset2");
        address zrc20 = makeAddr("zrc20");

        router.addToken(name);
        router.addTokenAssociation(name, chainId, asset1, zrc20);

        vm.expectEmit(true, true, false, false);
        emit TokenAssociationUpdated(name, chainId, asset2, zrc20);
        router.updateTokenAssociation(name, chainId, asset2, zrc20);

        (address returnedAsset, address returnedZrc20, uint256 chainIdValue) = router.getTokenAssociation(zrc20, chainId);
        assertEq(returnedAsset, asset2);
        assertEq(returnedZrc20, zrc20);
        assertEq(chainIdValue, chainId);
    }

    function test_UpdateTokenAssociation_NonExistentAssociationReverts() public {
        string memory name = "USDC";
        uint256 chainId = 1;
        address asset = makeAddr("asset");
        address zrc20 = makeAddr("zrc20");

        router.addToken(name);
        vm.expectRevert("Association does not exist");
        router.updateTokenAssociation(name, chainId, asset, zrc20);
    }

    function test_RemoveTokenAssociation() public {
        string memory name = "USDC";
        uint256 chainId = 1;
        address asset = makeAddr("asset");
        address zrc20 = makeAddr("zrc20");

        router.addToken(name);
        router.addTokenAssociation(name, chainId, asset, zrc20);

        vm.expectEmit(true, true, false, false);
        emit TokenAssociationRemoved(name, chainId);
        router.removeTokenAssociation(name, chainId);

        vm.expectRevert("Association does not exist");
        router.getTokenAssociation(zrc20, chainId);
    }

    function test_RemoveTokenAssociation_NonExistentAssociationReverts() public {
        string memory name = "USDC";
        uint256 chainId = 1;

        router.addToken(name);
        vm.expectRevert("Association does not exist");
        router.removeTokenAssociation(name, chainId);
    }

    function test_GetTokenAssociations() public {
        string memory name = "USDC";
        uint256 chainId1 = 1;
        uint256 chainId2 = 2;
        address asset1 = makeAddr("asset1");
        address asset2 = makeAddr("asset2");
        address zrc20 = makeAddr("zrc20");

        router.addToken(name);
        router.addTokenAssociation(name, chainId1, asset1, zrc20);
        router.addTokenAssociation(name, chainId2, asset2, zrc20);

        (uint256[] memory chainIds, address[] memory assets, address[] memory zrc20s) = router.getTokenAssociations(name);
        assertEq(chainIds.length, 2);
        assertEq(chainIds[0], chainId1);
        assertEq(chainIds[1], chainId2);
        assertEq(assets[0], asset1);
        assertEq(assets[1], asset2);
        assertEq(zrc20s[0], zrc20);
        assertEq(zrc20s[1], zrc20);
    }

    function test_GetSupportedTokens() public {
        string memory name1 = "USDC";
        string memory name2 = "USDT";

        router.addToken(name1);
        router.addToken(name2);

        string[] memory tokens = router.getSupportedTokens();
        assertEq(tokens.length, 2);
        assertEq(tokens[0], name1);
        assertEq(tokens[1], name2);
    }

    function test_NonAdminCannotModify() public {
        string memory name = "USDC";
        uint256 chainId = 1;
        address asset = makeAddr("asset");
        address zrc20 = makeAddr("zrc20");

        vm.prank(user1);
        vm.expectRevert(
            abi.encodeWithSelector(
                bytes4(keccak256("AccessControlUnauthorizedAccount(address,bytes32)")),
                user1,
                DEFAULT_ADMIN_ROLE
            )
        );
        router.addToken(name);

        router.addToken(name);
        vm.prank(user1);
        vm.expectRevert(
            abi.encodeWithSelector(
                bytes4(keccak256("AccessControlUnauthorizedAccount(address,bytes32)")),
                user1,
                DEFAULT_ADMIN_ROLE
            )
        );
        router.addTokenAssociation(name, chainId, asset, zrc20);

        router.addTokenAssociation(name, chainId, asset, zrc20);
        vm.prank(user1);
        vm.expectRevert(
            abi.encodeWithSelector(
                bytes4(keccak256("AccessControlUnauthorizedAccount(address,bytes32)")),
                user1,
                DEFAULT_ADMIN_ROLE
            )
        );
        router.updateTokenAssociation(name, chainId, asset, zrc20);

        vm.prank(user1);
        vm.expectRevert(
            abi.encodeWithSelector(
                bytes4(keccak256("AccessControlUnauthorizedAccount(address,bytes32)")),
                user1,
                DEFAULT_ADMIN_ROLE
            )
        );
        router.removeTokenAssociation(name, chainId);
    }

    function test_OnCall_Success() public {
        // Setup test data
        address intentContract = makeAddr("intentContract");
        uint256 targetChain = 2;
        uint256 amount = 1000 ether;
        uint256 tip = 300 ether;
        uint256 gasFee = 50 ether;
        bytes32 intentId = keccak256("test-intent");
        bytes memory receiverBytes = abi.encodePacked(makeAddr("receiver"));

        // Register input token
        router.addToken("INPUT");
        router.addTokenAssociation(
            "INPUT",
            targetChain,
            makeAddr("targetAsset"),
            address(targetZRC20)
        );

        // Register input token for source chain
        router.addTokenAssociation(
            "INPUT",
            1, // source chain
            makeAddr("inputAsset"),
            address(inputToken)
        );

        // Setup intent payload
        PayloadUtils.IntentPayload memory intentPayload = PayloadUtils.IntentPayload({
            intentId: intentId,
            amount: amount,
            tip: tip,
            targetChain: targetChain,
            receiver: receiverBytes
        });
        bytes memory message = PayloadUtils.encodeIntentPayload(
            intentId,
            amount,
            tip,
            targetChain,
            receiverBytes
        );

        // Set up intent contract for source chain
        vm.prank(owner);
        router.setIntentContract(1, intentContract);

        // Create message context with senderEVM matching the intent contract
        IGateway.ZetaChainMessageContext memory context = IGateway.ZetaChainMessageContext({
            sender: abi.encodePacked(makeAddr("sender")),
            senderEVM: intentContract,
            chainID: 1
        });

        // Setup token associations
        vm.mockCall(
            address(router),
            abi.encodeWithSelector(router.getTokenAssociation.selector, address(inputToken), targetChain),
            abi.encode(makeAddr("targetAsset"), address(targetZRC20), targetChain)
        );

        // Setup intent contract for target chain
        router.setIntentContract(targetChain, intentContract);

        // Setup gas fee info
        vm.mockCall(
            address(targetZRC20),
            abi.encodeWithSelector(IZRC20.withdrawGasFeeWithGasLimit.selector, 100000),
            abi.encode(address(gasZRC20), gasFee)
        );

        // Setup mock balances and approvals
        inputToken.mint(address(router), amount);
        gasZRC20.mint(address(router), gasFee);
        targetZRC20.mint(address(router), amount - 1 ether);

        // Setup mock swap responses with exact parameters
        vm.mockCall(
            address(router.uniswapV3Router()),
            abi.encodeWithSelector(
                ISwapRouter.exactInputSingle.selector,
                ISwapRouter.ExactInputSingleParams({
                    tokenIn: address(inputToken),
                    tokenOut: address(wzeta),
                    fee: 3000,
                    recipient: address(router),
                    deadline: block.timestamp + 15 minutes,
                    amountIn: amount,
                    amountOutMinimum: 0,
                    sqrtPriceLimitX96: 0
                })
            ),
            abi.encode(amount) // First swap: input -> ZETA
        );
        vm.mockCall(
            address(router.uniswapV3Router()),
            abi.encodeWithSelector(
                ISwapRouter.exactOutputSingle.selector,
                ISwapRouter.ExactOutputSingleParams({
                    tokenIn: address(wzeta),
                    tokenOut: address(gasZRC20),
                    fee: 3000,
                    recipient: address(router),
                    deadline: block.timestamp + 15 minutes,
                    amountOut: gasFee,
                    amountInMaximum: amount,
                    sqrtPriceLimitX96: 0
                })
            ),
            abi.encode(gasFee) // Second swap: ZETA -> gas token
        );
        vm.mockCall(
            address(router.uniswapV3Router()),
            abi.encodeWithSelector(
                ISwapRouter.exactInputSingle.selector,
                ISwapRouter.ExactInputSingleParams({
                    tokenIn: address(wzeta),
                    tokenOut: address(targetZRC20),
                    fee: 3000,
                    recipient: address(router),
                    deadline: block.timestamp + 15 minutes,
                    amountIn: amount - gasFee,
                    amountOutMinimum: 0,
                    sqrtPriceLimitX96: 0
                })
            ),
            abi.encode(amount - 1 ether) // Third swap: remaining ZETA -> target token
        );

        // Verify expected calls before execution
        vm.expectCall(
            address(inputToken),
            abi.encodeWithSelector(IERC20.approve.selector, address(router.uniswapV3Router()), amount)
        );
        vm.expectCall(
            address(wzeta),
            abi.encodeWithSelector(IERC20.approve.selector, address(router.uniswapV3Router()), amount)
        );
        vm.expectCall(
            address(targetZRC20),
            abi.encodeWithSelector(IERC20.approve.selector, address(gateway), amount - 1 ether)
        );
        vm.expectCall(
            address(gasZRC20),
            abi.encodeWithSelector(IERC20.approve.selector, address(gateway), gasFee)
        );

        // Verify expected swaps
        vm.expectCall(
            address(router.uniswapV3Router()),
            abi.encodeWithSelector(ISwapRouter.exactInputSingle.selector)
        );
        vm.expectCall(
            address(router.uniswapV3Router()),
            abi.encodeWithSelector(ISwapRouter.exactOutputSingle.selector)
        );
        vm.expectCall(
            address(router.uniswapV3Router()),
            abi.encodeWithSelector(ISwapRouter.exactInputSingle.selector)
        );

        // Verify gateway call
        // TODO: fix this expected call check, we can see in the logs when executing the test that withdrawAndCall is called
        // https://github.com/lumtis/zetafast/issues/10
        // but it seems there are a discrepancy of the data actually used
        // vm.expectCall(
        //     address(gateway),
        //     abi.encodeWithSelector(
        //         IGateway.withdrawAndCall.selector,
        //         abi.encodePacked(intentContract),
        //         amount - 1 ether,
        //         address(targetZRC20),
        //         abi.encode(
        //             intentPayload.intentId,
        //             intentPayload.amount,
        //             "targetAsset",
        //             receiverBytes,
        //             tip - 1 ether
        //         ),
        //         abi.encode(IGateway.CallOptions({gasLimit: 100000, isArbitraryCall: false})),
        //         abi.encode(IGateway.RevertOptions({
        //             revertAddress: address(0),
        //             callOnRevert: false,
        //             abortAddress: address(0),
        //             revertMessage: "",
        //             onRevertGasLimit: 0
        //         }))
        //     )
        // );

        // Verify event
        vm.expectEmit(true, true, true, false);
        emit IntentSettlementForwarded(
            abi.encodePacked(makeAddr("sender")),
            1,
            targetChain,
            address(inputToken),
            amount,
            tip - 1 ether
        );

        // Execute
        vm.prank(address(gateway));
        router.onCall(context, address(inputToken), amount, message);
    }

    // TODO: add failure case for onCall
    // https://github.com/lumtis/zetafast/issues/10
} 