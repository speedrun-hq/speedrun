// SPDX-License-Identifier: MIT
pragma solidity ^0.8.26;

import {Script, console2} from "forge-std/Script.sol";
import {Router} from "../src/router.sol";

contract RouterScript is Script {
    function setUp() public {}

    function run() public {
        uint256 deployerPrivateKey = vm.envUint("PRIVATE_KEY");
        vm.startBroadcast(deployerPrivateKey);

        // Deploy Router contract
        Router router = new Router();
        router.initialize();

        console2.log("Router deployed to:", address(router));

        vm.stopBroadcast();
    }
} 