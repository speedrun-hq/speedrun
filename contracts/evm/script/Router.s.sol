// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import {Script, console2} from "forge-std/Script.sol";
import {Router} from "../src/Router.sol";
import {ERC1967Proxy} from "@openzeppelin/contracts/proxy/ERC1967/ERC1967Proxy.sol";

contract RouterScript is Script {
    function setUp() public {}

    function run() public {
        uint256 deployerPrivateKey = vm.envUint("PRIVATE_KEY");
        vm.startBroadcast(deployerPrivateKey);

        // Get environment variables
        address gateway = vm.envAddress("GATEWAY_ADDRESS");
        address wzeta = vm.envAddress("WZETA_ADDRESS");
        address swapModule = vm.envAddress("SWAP_MODULE_ADDRESS");

        // Deploy implementation
        Router implementation = new Router();

        // Prepare initialization data
        bytes memory initData = abi.encodeWithSelector(
            Router.initialize.selector,
            gateway,
            wzeta,
            swapModule
        );

        // Deploy proxy
        ERC1967Proxy proxy = new ERC1967Proxy(
            address(implementation),
            initData
        );

        Router router = Router(address(proxy));

        console2.log("Router deployed to:", address(router));
        console2.log("Implementation at:", address(implementation));
        console2.log("Proxy at:", address(proxy));
        console2.log("Initialized with:");
        console2.log("- Gateway:", gateway);
        console2.log("- WZETA:", wzeta);
        console2.log("- Swap Module:", swapModule);
        vm.stopBroadcast();
    }
} 