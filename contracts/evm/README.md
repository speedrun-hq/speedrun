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

## Deployment on Mainnet

These commands showcase how the infrastucture can be deployed on Mainnet to support USDC between Base and Arbitrum.

Deploy the router contract on ZetaChain:
```
export PRIVATE_KEY=<private key>
export GATEWAY_ADDRESS="0xfEDD7A6e3Ef1cC470fbfbF955a22D793dDC0F44E"
export UNISWAP_FACTORY_ADDRESS="0x67AA6B2b715937Edc1Eb4D3b7B5d5dCD1fd93E8C"
export UNISWAP_ROUTER_ADDRESS="0x9b30CfbACD3504252F82263F72D6acf62bf733C2"
export WZETA_ADDRESS="0x5F0b1a82749cb4E2278EC87F8BF6B618dC71a8bf"

forge script script/Router.s.sol \
  --rpc-url https://zetachain-mainnet.g.allthatnode.com/archive/evm \
  --chain-id 7000  \
  --broadcast
```

Deploy the intent contract on Base:
```
export ROUTER_ADDRESS=<router>

forge script script/intent.s.sol \
  --rpc-url https://mainnet.base.org \
  --chain-id 8453  \
  --broadcast
```
Deploy the intent contract on Arbitrum:
```
export ROUTER_ADDRESS=<router>

forge script script/intent.s.sol \
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
export ASSET_ADDRESS=0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913
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
export ASSET_ADDRESS=0xaf88d065e77c8cC2239327C5EDb3A432268e5831
export ZRC20_ADDRESS="0x0327f0660525b15Cdb8f1f5FBF0dD7Cd5Ba182aD"
export TOKEN_NAME="USDC"

forge script script/AddTokenAssociation.s.sol \
  --rpc-url https://zetachain-mainnet.g.allthatnode.com/archive/evm \
  --chain-id 7000  \
  --broadcast
```

## License

MIT
