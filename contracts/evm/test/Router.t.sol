// SPDX-License-Identifier: MIT
pragma solidity ^0.8.26;

import {Test, console2} from "forge-std/Test.sol";
import {Router} from "../src/router.sol";
import {ERC1967Proxy} from "@openzeppelin/contracts/proxy/ERC1967/ERC1967Proxy.sol";
import {MockGateway} from "./mocks/MockGateway.sol";
import {MockUniswapV3Factory} from "./mocks/MockUniswapV3Factory.sol";
import {MockUniswapV3Router} from "./mocks/MockUniswapV3Router.sol";
import {MockWETH} from "./mocks/MockWETH.sol";

contract RouterTest is Test {
    Router public router;
    Router public routerImplementation;
    MockGateway public gateway;
    MockUniswapV3Factory public factory;
    MockUniswapV3Router public swapRouter;
    MockWETH public weth;
    address public owner;
    address public user1;
    address public user2;
    bytes32 public constant DEFAULT_ADMIN_ROLE = 0x00;

    event IntentContractSet(uint256 indexed chainId, address indexed intentContract);
    event TokenAdded(string indexed name);
    event TokenAssociationAdded(string indexed name, uint256 indexed chainId, address asset, address zrc20);
    event TokenAssociationUpdated(string indexed name, uint256 indexed chainId, address asset, address zrc20);
    event TokenAssociationRemoved(string indexed name, uint256 indexed chainId);

    function setUp() public {
        owner = address(this);
        user1 = makeAddr("user1");
        user2 = makeAddr("user2");

        // Deploy mock contracts
        gateway = new MockGateway();
        factory = new MockUniswapV3Factory();
        swapRouter = new MockUniswapV3Router();
        weth = new MockWETH();

        // Deploy implementation
        routerImplementation = new Router();

        // Prepare initialization data
        bytes memory initData = abi.encodeWithSelector(
            Router.initialize.selector,
            address(gateway),
            address(factory),
            address(swapRouter),
            address(weth)
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

    // TODO: Add more tests for:
    // - route CCTX
    // - ZRC20 swap
    // - registry management
    // - onCall
    // - onRevert
    // - onAbort
} 