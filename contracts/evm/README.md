# EVM Contracts

This directory contains the Solidity contracts for intent-based bridge platform implementation.

## Intent Contract Interface

### Initiate Intent
```solidity
function initiate(
  address asset,
  uint256 amount,
  uint256 targetChain,
  bytes calldata receiver,
  uint256 tip,
  uint256 salt
) external
```

Creates a new intent for cross-chain transfer:
- `asset`: Address of the token to transfer
- `amount`: Amount of tokens to transfer
- `targetChain`: Chain ID of the destination chain
- `receiver`: Address of the receiver on the target chain (in bytes format)
- `tip`: Amount of tokens to pay as fee for the cross-chain transfer
- `salt`: Random value to ensure uniqueness of the intent ID

### Fulfill Intent
```solidity
function fulfill(
  bytes32 intentId,
  uint256 amount,
  address asset,
  address receiver
) external
```

Fulfills an existing intent:
- `intentId`: Identifier of the intent to fulfill
- `amount`: Actual amount of tokens being transferred (may differ from intent's amount)
- `asset`: Address of the token being transferred
- `receiver`: Address of the recipient on the target chain

## Architecture

[Learn more about the contract architecture](./architecture.md)

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

## Deploying on Mainnet to Enable USDT on Base and Arbitrum

These commands showcase how the infrastucture can be deployed on Mainnet to support USDC between Base and Arbitrum.

All commands requires to set the private key env var, here we assume the same key is used across all networks.
```
export PRIVATE_KEY=<private key>
```
Deploy the swap module that uses Uniswap V2:
```
export UNISWAP_V2_ROUTER="0x2ca7d64A7EFE2D62A725E2B35Cf7230D6677FfEe"
export WZETA="0x5F0b1a82749cb4E2278EC87F8BF6B618dC71a8bf"

forge script script/SwapV2.s.sol \
  --rpc-url https://zetachain-mainnet.g.allthatnode.com/archive/evm \
  --chain-id 7000 \
  --broadcast
```

Deploy the router contract on ZetaChain:
```
export GATEWAY_ADDRESS="0xfEDD7A6e3Ef1cC470fbfbF955a22D793dDC0F44E"
export SWAP_MODULE_ADDRESS=<swap module>

forge script script/Router.s.sol \
  --rpc-url https://zetachain-mainnet.g.allthatnode.com/archive/evm \
  --chain-id 7000  \
  --broadcast
```

Deploy the intent contract on Base:
```
export ROUTER_ADDRESS=<router>
export GATEWAY_ADDRESS="0x48B9AACC350b20147001f88821d31731Ba4C30ed"

forge script script/Intent.s.sol \
  --rpc-url https://mainnet.base.org \
  --chain-id 8453  \
  --broadcast
```
Deploy the intent contract on Arbitrum:
```
export ROUTER_ADDRESS=<router>
export GATEWAY_ADDRESS="0x1C53e188Bc2E471f9D4A4762CFf843d32C2C8549"

forge script script/Intent.s.sol \
  --rpc-url https://arb1.arbitrum.io/rpc \
  --chain-id 42161  \
  --broadcast
```
Set intent contracts on the router
```
export ROUTER_ADDRESS=<router>
export INTENT_ADDRESS=<base intent>
export CHAIN_ID=8453

forge script script/SetIntent.s.sol \
  --rpc-url https://zetachain-mainnet.g.allthatnode.com/archive/evm \
  --chain-id 7000  \
  --broadcast

export INTENT_ADDRESS=<arbitrum intent>
export CHAIN_ID=42161

forge script script/SetIntent.s.sol \
  --rpc-url https://zetachain-mainnet.g.allthatnode.com/archive/evm \
  --chain-id 7000  \
  --broadcast
```
Create USDC token in the router:
```
export ROUTER_ADDRESS=<router>
export TOKEN_NAME="USDC"

forge script script/AddToken.s.sol \
  --rpc-url https://zetachain-mainnet.g.allthatnode.com/archive/evm \
  --chain-id 7000  \
  --broadcast
```
Associate USDC from Base:
```
export ROUTER_ADDRESS=<router>
export CHAIN_ID=8453
export ASSET_ADDRESS="0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"
export ZRC20_ADDRESS="0x96152E6180E085FA57c7708e18AF8F05e37B479D"
export TOKEN_NAME="USDC"

forge script script/AddTokenAssociation.s.sol \
  --rpc-url https://zetachain-mainnet.g.allthatnode.com/archive/evm \
  --chain-id 7000  \
  --broadcast
```
Associate USDC from Arbitrum:
```
export ROUTER_ADDRESS=<router>
export CHAIN_ID=42161
export ASSET_ADDRESS="0xaf88d065e77c8cC2239327C5EDb3A432268e5831"
export ZRC20_ADDRESS="0x0327f0660525b15Cdb8f1f5FBF0dD7Cd5Ba182aD"
export TOKEN_NAME="USDC"

forge script script/AddTokenAssociation.s.sol \
  --rpc-url https://zetachain-mainnet.g.allthatnode.com/archive/evm \
  --chain-id 7000  \
  --broadcast
```

## License

MIT
