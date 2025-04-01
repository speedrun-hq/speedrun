// SPDX-License-Identifier: MIT
pragma solidity ^0.8.26;

import {Test, console2} from "forge-std/Test.sol";
import {Intent} from "../src/intent.sol";
import {MockGateway} from "./mocks/MockGateway.sol";
import {MockERC20} from "./mocks/MockERC20.sol";
import {PayloadUtils} from "../src/utils/PayloadUtils.sol";
import {IGateway} from "../src/interfaces/IGateway.sol";
import {ERC1967Proxy} from "@openzeppelin/contracts/proxy/ERC1967/ERC1967Proxy.sol";
import {IERC20Errors} from "@openzeppelin/contracts/interfaces/draft-IERC6093.sol";

contract IntentTest is Test {
    Intent public intent;
    Intent public intentImplementation;
    MockGateway public gateway;
    MockERC20 public token;
    address public owner;
    address public user1;
    address public user2;
    address public router;

    // Define the event to match the one in Intent contract
    event IntentInitiated(
        bytes32 indexed intentId,
        address indexed asset,
        uint256 amount,
        uint256 targetChain,
        bytes receiver,
        uint256 tip,
        uint256 salt
    );

    // Define the event for intent fulfillment
    event IntentFulfilled(
        bytes32 indexed intentId,
        address indexed asset,
        uint256 amount,
        address indexed receiver
    );

    function setUp() public {
        owner = address(this);
        user1 = makeAddr("user1");
        user2 = makeAddr("user2");
        router = makeAddr("router");

        // Deploy mock contracts
        gateway = new MockGateway();
        token = new MockERC20("Test Token", "TEST");

        // Deploy implementation
        intentImplementation = new Intent();

        // Prepare initialization data
        bytes memory initData = abi.encodeWithSelector(
            Intent.initialize.selector,
            address(gateway),
            router
        );

        // Deploy proxy
        ERC1967Proxy proxy = new ERC1967Proxy(
            address(intentImplementation),
            initData
        );
        intent = Intent(address(proxy));

        // Setup test tokens
        token.mint(user1, 1000 ether);
        vm.prank(user1);
        token.approve(address(intent), 1000 ether);
    }

    function test_Initialization() public {
        assertEq(intent.owner(), owner);
        assertEq(intent.gateway(), address(gateway));
        assertEq(intent.router(), router);
    }

    function test_Initiate() public {
        // Test parameters
        uint256 amount = 100 ether;
        uint256 tip = 10 ether;
        uint256 targetChain = 1;
        bytes memory receiver = abi.encodePacked(user2);
        uint256 salt = 123;

        // Expect the IntentInitiated event
        vm.expectEmit(true, true, false, false);
        emit IntentInitiated(
            keccak256(abi.encodePacked(uint256(0), salt)), // First intent ID
            address(token),
            amount,
            targetChain,
            receiver,
            tip,
            salt
        );

        // Call initiate
        vm.prank(user1);
        bytes32 intentId = intent.initiate(
            address(token),
            amount,
            targetChain,
            receiver,
            tip,
            salt
        );

        // Verify intent ID
        assertEq(intentId, keccak256(abi.encodePacked(uint256(0), salt)));

        // Verify gateway received the correct amount
        assertEq(token.balanceOf(address(gateway)), amount + tip);

        // Verify gateway call data
        (address callReceiver, uint256 callAmount, address callAsset, bytes memory callPayload, IGateway.RevertOptions memory callRevertOptions) = gateway.lastCall();
        assertEq(callReceiver, router);
        assertEq(callAmount, amount + tip);
        assertEq(callAsset, address(token));

        // Verify payload
        PayloadUtils.IntentPayload memory payload = PayloadUtils.decodeIntentPayload(callPayload);
        assertEq(payload.intentId, intentId);
        assertEq(payload.amount, amount);
        assertEq(payload.tip, tip);
        assertEq(payload.targetChain, targetChain);
        assertEq(keccak256(payload.receiver), keccak256(receiver));
    }

    function test_InitiateInsufficientBalance() public {
        // Test parameters
        uint256 amount = 1000 ether; // More than user1's balance
        uint256 tip = 10 ether;
        uint256 targetChain = 1;
        bytes memory receiver = abi.encodePacked(user2);
        uint256 salt = 123;

        // Call initiate and expect revert with ERC20InsufficientAllowance error
        vm.prank(user1);
        vm.expectRevert(
            abi.encodeWithSelector(
                IERC20Errors.ERC20InsufficientAllowance.selector,
                address(intent),
                1000 ether,
                1010 ether
            )
        );
        intent.initiate(
            address(token),
            amount,
            targetChain,
            receiver,
            tip,
            salt
        );
    }

    function test_Fulfill() public {
        // First create an intent
        uint256 amount = 100 ether;
        uint256 tip = 10 ether;
        uint256 targetChain = 1;
        bytes memory receiver = abi.encodePacked(user2);
        uint256 salt = 123;

        vm.prank(user1);
        bytes32 intentId = intent.initiate(
            address(token),
            amount,
            targetChain,
            receiver,
            tip,
            salt
        );

        // Transfer tokens to the intent contract for fulfillment
        token.mint(address(intent), amount);

        // Expect the IntentFulfilled event
        vm.expectEmit(true, true, false, true);
        emit IntentFulfilled(
            intentId,
            address(token),
            amount,
            user2
        );

        // Call fulfill
        vm.prank(user1);
        intent.fulfill(
            intentId,
            address(token),
            amount,
            user2
        );

        // Verify fulfillment was registered
        bytes32 fulfillmentIndex = PayloadUtils.computeFulfillmentIndex(
            intentId,
            address(token),
            amount,
            user2
        );
        assertEq(intent.fulfillments(fulfillmentIndex), user1);

        // Verify tokens were transferred
        assertEq(token.balanceOf(user2), amount);
    }

    function test_FulfillAlreadyFulfilled() public {
        // First create and fulfill an intent
        uint256 amount = 100 ether;
        uint256 tip = 10 ether;
        uint256 targetChain = 1;
        bytes memory receiver = abi.encodePacked(user2);
        uint256 salt = 123;

        vm.prank(user1);
        bytes32 intentId = intent.initiate(
            address(token),
            amount,
            targetChain,
            receiver,
            tip,
            salt
        );

        // Transfer tokens to the intent contract for fulfillment
        token.mint(address(intent), amount);

        // First fulfillment
        vm.prank(user1);
        intent.fulfill(
            intentId,
            address(token),
            amount,
            user2
        );

        // Try to fulfill again with same parameters and expect revert
        vm.prank(user1);
        vm.expectRevert("Intent already fulfilled with these parameters");
        intent.fulfill(
            intentId,
            address(token),
            amount,
            user2
        );
    }

    function test_FulfillWithDifferentParameters() public {
        // First create and fulfill an intent
        uint256 amount = 100 ether;
        uint256 tip = 10 ether;
        uint256 targetChain = 1;
        bytes memory receiver = abi.encodePacked(user2);
        uint256 salt = 123;

        vm.prank(user1);
        bytes32 intentId = intent.initiate(
            address(token),
            amount,
            targetChain,
            receiver,
            tip,
            salt
        );

        // Transfer tokens to the intent contract for both fulfillments
        token.mint(address(intent), amount + (amount + 1 ether));

        // First fulfillment
        vm.prank(user1);
        intent.fulfill(
            intentId,
            address(token),
            amount,
            user2
        );

        // Try to fulfill with different amount
        vm.prank(user1);
        intent.fulfill(
            intentId,
            address(token),
            amount + 1 ether,
            user2
        );

        // Verify both fulfillments were registered
        bytes32 fulfillmentIndex1 = PayloadUtils.computeFulfillmentIndex(
            intentId,
            address(token),
            amount,
            user2
        );
        bytes32 fulfillmentIndex2 = PayloadUtils.computeFulfillmentIndex(
            intentId,
            address(token),
            amount + 1 ether,
            user2
        );
        assertEq(intent.fulfillments(fulfillmentIndex1), user1);
        assertEq(intent.fulfillments(fulfillmentIndex2), user1);

        // Verify tokens were transferred for both fulfillments
        assertEq(token.balanceOf(user2), amount + (amount + 1 ether));
    }

    function test_OnCallWithFulfillment() public {
        // First create and fulfill an intent
        uint256 amount = 100 ether;
        uint256 tip = 10 ether;
        uint256 targetChain = 1;
        bytes memory receiver = abi.encodePacked(user2);
        uint256 salt = 123;

        // Store initial balance
        uint256 initialBalance = token.balanceOf(user1);

        vm.prank(user1);
        bytes32 intentId = intent.initiate(
            address(token),
            amount,
            targetChain,
            receiver,
            tip,
            salt
        );

        // Account for tokens spent during initiate
        initialBalance -= (amount + tip);

        // Transfer tokens to the intent contract for fulfillment
        token.mint(address(intent), amount);

        // Fulfill the intent
        vm.prank(user1);
        intent.fulfill(
            intentId,
            address(token),
            amount,
            user2
        );

        // Prepare settlement payload
        bytes memory settlementPayload = PayloadUtils.encodeSettlementPayload(
            intentId,
            amount,
            address(token),
            user2,
            tip
        );

        // Transfer tokens to gateway for settlement
        token.mint(address(gateway), amount + tip);
        vm.prank(address(gateway));
        token.approve(address(intent), amount + tip);

        // Call onCall through gateway
        vm.prank(address(gateway));
        intent.onCall(
            Intent.MessageContext({
                sender: router
            }),
            settlementPayload
        );

        // Verify settlement record
        bytes32 fulfillmentIndex = PayloadUtils.computeFulfillmentIndex(
            intentId,
            address(token),
            amount,
            user2
        );
        (bool settled, bool fulfilled, uint256 paidTip, address fulfiller) = intent.settlements(fulfillmentIndex);
        assertTrue(settled);
        assertTrue(fulfilled);
        assertEq(paidTip, tip);
        assertEq(fulfiller, user1);

        // Verify tokens were transferred to fulfiller (amount + tip)
        assertEq(token.balanceOf(user1), initialBalance + amount + tip);
    }

    function test_OnCallWithoutFulfillment() public {
        // Create an intent
        uint256 amount = 100 ether;
        uint256 tip = 10 ether;
        uint256 targetChain = 1;
        bytes memory receiver = abi.encodePacked(user2);
        uint256 salt = 123;

        vm.prank(user1);
        bytes32 intentId = intent.initiate(
            address(token),
            amount,
            targetChain,
            receiver,
            tip,
            salt
        );

        // Prepare settlement payload
        bytes memory settlementPayload = PayloadUtils.encodeSettlementPayload(
            intentId,
            amount,
            address(token),
            user2,
            tip
        );

        // Transfer tokens to gateway for settlement
        token.mint(address(gateway), amount + tip);
        vm.prank(address(gateway));
        token.approve(address(intent), amount + tip);

        // Call onCall through gateway
        vm.prank(address(gateway));
        intent.onCall(
            Intent.MessageContext({
                sender: router
            }),
            settlementPayload
        );

        // Verify settlement record
        bytes32 fulfillmentIndex = PayloadUtils.computeFulfillmentIndex(
            intentId,
            address(token),
            amount,
            user2
        );
        (bool settled, bool fulfilled, uint256 paidTip, address fulfiller) = intent.settlements(fulfillmentIndex);
        assertTrue(settled);
        assertFalse(fulfilled);
        assertEq(paidTip, 0);
        assertEq(fulfiller, address(0));

        // Verify tokens were transferred to receiver (amount + tip)
        assertEq(token.balanceOf(user2), amount + tip);
    }

    function test_OnCallInvalidSender() public {
        // Create an intent
        uint256 amount = 100 ether;
        uint256 tip = 10 ether;
        uint256 targetChain = 1;
        bytes memory receiver = abi.encodePacked(user2);
        uint256 salt = 123;

        vm.prank(user1);
        bytes32 intentId = intent.initiate(
            address(token),
            amount,
            targetChain,
            receiver,
            tip,
            salt
        );

        // Prepare settlement payload
        bytes memory settlementPayload = PayloadUtils.encodeSettlementPayload(
            intentId,
            amount,
            address(token),
            user2,
            tip
        );

        // Transfer tokens to gateway for settlement
        token.mint(address(gateway), amount + tip);
        vm.prank(address(gateway));
        token.approve(address(intent), amount + tip);

        // Call onCall through gateway with invalid sender
        vm.prank(address(gateway));
        vm.expectRevert("Invalid sender");
        intent.onCall(
            Intent.MessageContext({
                sender: address(0x123) // Invalid sender
            }),
            settlementPayload
        );
    }

    // TODO: Add more tests for:
    // - complete
    // - onCall
    // - onRevert
} 