// SPDX-License-Identifier: MIT
pragma solidity ^0.8.26;

import {Test, console2} from "forge-std/Test.sol";
import {Router} from "../src/router.sol";
import {ERC1967Proxy} from "@openzeppelin/contracts/proxy/ERC1967/ERC1967Proxy.sol";

contract RouterTest is Test {
    Router public router;
    Router public routerImplementation;
    address public owner;
    address public user1;
    address public user2;
    bytes32 public constant DEFAULT_ADMIN_ROLE = 0x00;

    event IntentContractSet(uint256 indexed chainId, address indexed intentContract);

    function setUp() public {
        owner = address(this);
        user1 = makeAddr("user1");
        user2 = makeAddr("user2");

        // Deploy implementation
        routerImplementation = new Router();

        // Prepare initialization data
        bytes memory initData = abi.encodeWithSelector(
            Router.initialize.selector
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

    // TODO: Add more tests for:
    // - route CCTX
    // - ZRC20 swap
    // - registry management
    // - onCall
    // - onRevert
    // - onAbort
} 