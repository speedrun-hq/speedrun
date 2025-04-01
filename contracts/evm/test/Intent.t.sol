// SPDX-License-Identifier: MIT
pragma solidity ^0.8.26;

import {Test, console2} from "forge-std/Test.sol";
import {Intent} from "../src/intent.sol";

contract IntentTest is Test {
    Intent public intent;
    address public owner;
    address public user1;
    address public user2;

    function setUp() public {
        owner = address(this);
        user1 = makeAddr("user1");
        user2 = makeAddr("user2");

        // Deploy Intent contract
        intent = new Intent();
        intent.initialize();
    }

    function test_Initialization() public {
        assertEq(intent.owner(), owner);
    }

    // TODO: Add more tests for:
    // - initiate
    // - fulfill
    // - complete
    // - onCall
    // - onRevert
} 