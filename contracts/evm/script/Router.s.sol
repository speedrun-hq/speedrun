// SPDX-License-Identifier: MIT
pragma solidity ^0.8.26;

import "forge-std/Script.sol";
import "../src/Router.sol";
import "../test/mocks/MockGateway.sol";

contract RouterScript is Script {
    function setUp() public {}

    function run() public {
        uint256 deployerPrivateKey = vm.envUint("PRIVATE_KEY");
        vm.startBroadcast(deployerPrivateKey);

        // Deploy mock gateway
        MockGateway gateway = new MockGateway();

        // Deploy router
        Router router = new Router();
        router.initialize(
            address(gateway),
            address(0), // Mock Uniswap V3 Factory
            address(0), // Mock Uniswap V3 Router
            address(0)  // Mock WETH
        );

        vm.stopBroadcast();
    }
} 