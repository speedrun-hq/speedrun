// SPDX-License-Identifier: MIT
pragma solidity ^0.8.26;

import {Script, console2} from "forge-std/Script.sol";
import {Intent} from "../src/intent.sol";

contract IntentScript is Script {
    function setUp() public {}

    function run() public {
        uint256 deployerPrivateKey = vm.envUint("PRIVATE_KEY");
        vm.startBroadcast(deployerPrivateKey);

        // Get gateway and router addresses from environment variables
        address gateway = vm.envAddress("GATEWAY_ADDRESS");
        address router = vm.envAddress("ROUTER_ADDRESS");

        // Deploy Intent contract
        Intent intent = new Intent();
        intent.initialize(gateway, router);

        console2.log("Intent deployed to:", address(intent));
        console2.log("Gateway address:", gateway);
        console2.log("Router address:", router);

        vm.stopBroadcast();
    }
} 