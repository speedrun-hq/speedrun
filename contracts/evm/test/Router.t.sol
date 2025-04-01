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

    // TODO: Add more tests for:
    // - route CCTX
    // - ZRC20 swap
    // - registry management
    // - onCall
    // - onRevert
    // - onAbort
} 