// SPDX-License-Identifier: MIT
pragma solidity ^0.8.26;

import {Script, console2} from "forge-std/Script.sol";
import {Intent} from "../src/intent.sol";

contract IntentScript is Script {
    function setUp() public {}

    function run() public {
        uint256 deployerPrivateKey = vm.envUint("PRIVATE_KEY");
        vm.startBroadcast(deployerPrivateKey);

        // Deploy Intent contract
        Intent intent = new Intent();
        intent.initialize();

        console2.log("Intent deployed to:", address(intent));

        vm.stopBroadcast();
    }
} 