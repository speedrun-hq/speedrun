# EVM Contracts

This directory contains the Solidity contracts for intent-based bridge platform implementation.

## Intent Contract Interface

### Initiate Intent
```solidity
function initiate(
    bytes32 intentId,
    uint256 amount,
    address asset,
    bytes memory receiver,
    uint256 targetChain,
    address targetZRC20,
    uint256 tip
) external
```

Creates a new intent for cross-chain transfer:
- `intentId`: Unique identifier for the intent
- `amount`: Amount of tokens to transfer
- `asset`: Address of the token to transfer
- `receiver`: Address of the receiver on the target chain (in bytes format)
- `targetChain`: Chain ID of the destination chain
- `targetZRC20`: Address of the ZRC20 token on the target chain
- `tip`: Amount of tokens to pay for the cross-chain transfer

### Fulfill Intent
```solidity
function fulfill(
    bytes32 intentId,
    uint256 amount,
    address asset,
    address receiver,
    uint256 tip
) external
```

Fulfills an existing intent:
- `intentId`: Identifier of the intent to fulfill
- `amount`: Amount of tokens to transfer
- `asset`: Address of the token to transfer
- `receiver`: Address of the receiver
- `tip`: Amount of tokens to pay for the transfer

## Trading Path

All token swaps on ZetaChain are executed through the native ZETA token as an intermediary. For example:
- USDC on BSC → USDC on Base: USDC (BSC) → ZETA → USDC (Base)
- USDT on Polygon → USDC on Arbitrum: USDT (Polygon) → ZETA → USDC (Arbitrum)

This is handled automatically by the Uniswap V3 router, which finds the optimal path through the ZETA token. No manual path management is required.

## Development

### Prerequisites
- Foundry
- Solidity 0.8.26
- Node.js (for dependencies)

### Setup
1. Install dependencies:
```bash
npm install
```

2. Install Foundry:
```bash
curl -L https://foundry.paradigm.xyz | bash
foundryup
```

### Build
```bash
forge build
```

### Test
```bash
forge test
```

### Deploy
```bash
forge script script/Router.s.sol:RouterScript --rpc-url <your_rpc_url> --private-key <your_private_key>
```

### Mock Contracts
- `MockGateway`: Simulates ZetaChain gateway behavior
- `MockUniswapV3Factory`: Simulates Uniswap V3 factory
- `MockUniswapV3Router`: Simulates Uniswap V3 router
- `MockWETH`: Simulates WETH token

## License

MIT
