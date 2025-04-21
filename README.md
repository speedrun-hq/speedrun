# Speedrun

A permissionless intent-based token transfer system built on ZetaChain that enables fast cross-chain transfers through incentivized third-party fulfillers.

## Overview

Speedrun is a novel cross-chain transfer system that leverages ZetaChain's infrastructure while providing faster transfer execution through a market-driven fulfillment mechanism. Users can initiate transfers with intent fees, allowing third-party agents (fulfillers) to execute transfers early on the destination chain in exchange for compensation.

### Key Features

- **Permissionless Fulfillment**: Anyone can participate as a fulfiller
- **Market-Driven Speed**: Faster transfers through incentivized early execution
- **Partial Fulfillment Support**: Multiple fulfillers can split a single transfer
- **Simple Interface**: Clean, intuitive web interface for transfer initiation
- **Extensible Architecture**: Ready for future features like intent-based swaps

## Smart Contracts

[Learn more about the smart contracts](https://github.com/speedrun-hq/contracts-core)

## Architecture

### Core Components

1. **Smart Contracts**
   - Intent-based transfers witht routing on ZetaChain
   - Smart contracts for each supported VMs

1. **Web Interface (React)**
   - Transfer initiation form
   - Real-time status tracking
   - Transaction history

2. **Backend API**
   - Intent registration
   - Fulfillment tracking
   - CCTX monitoring

3. **Fulfiller Tooling**
   - Intent scanning
   - Fulfillment execution
   - Fee calculation

### Workflow

1. **Transfer Creation**
   - User specifies:
     - Source Chain
     - Destination Chain
     - Token
     - Amount
     - Intent Fee
   - System creates CCTX and registers Intent ID

2. **Fulfillment Process**
   - Fulfillers monitor intents
   - Early execution on destination chain
   - Partial fulfillment support
   - Proportional fee distribution

3. **Reconciliation**
   - CCTX arrival processing
   - Fulfiller compensation
   - Original transfer completion

## Getting Started

### Prerequisites

- Node.js (v16 or higher)
- Go (v1.20 or higher)
- Access to ZetaChain network


## Fee Structure

- **Intent Fee**: Set by the platform (configurable)
- **Fulfiller Compensation**: Proportional to fulfilled amount
- **Refund Policy**: Full refund if no fulfillment occurs

## Future Enhancements

- More token swap integrations
- Market-based fee determination
- Additional chain support (Solana, Sui)
- Advanced automation features

