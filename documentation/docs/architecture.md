# Architecture

## Overview

Speedrun uses an **intent-based architecture** to enable fast and secure cross-chain transfers through a two-step process:

1. **Users create transfer intents** with parameters such as amount, destination chain, and recipient.
2. **Speedrunners** (specialized fulfillers) monitor and fulfill these intents in exchange for a fee.

This design **decouples settlement from user experience** — users receive tokens instantly from the fulfiller, while the actual cross-chain settlement happens in the background via ZetaChain.

> ⚠️ If no fulfiller picks up the intent, it will still be executed through ZetaChain's native cross-chain mechanism — just with higher latency.

## Workflow

The Speedrun protocol follows a simple and secure process to move tokens across chains in seconds.

**1. User Initiates a Transfer**

The user begins by submitting a cross-chain transfer request, called an **intent**. This includes details like the destination chain, recipient, and amount to be transferred.

**2. Intent Broadcasted on ZetaChain**

Once the intent is submitted, it's relayed through ZetaChain, which acts as the **trustless and decentralized messaging layer** connecting all supported blockchains. ZetaChain ensures the intent is safely recorded and available for fulfillers to act on.

**3. Fulfiller Observes and Acts**

A **fulfiller** — a participant with liquidity on the target chain — monitors new intents. If they choose to act, they **front the tokens** directly to the recipient on the destination chain in exchange for a small fee.

This step is what makes Speedrun extremely fast: the user doesn’t wait for final cross-chain settlement — they receive their tokens immediately.

**4. Cross-Chain Settlement**

While the user has already received their funds, the original intent continues its journey. ZetaChain coordinates the actual **cross-chain settlement** in the background, ensuring the fulfiller is reimbursed.

If no fulfiller steps in, the settlement is still guaranteed — but it takes longer to complete, similar to traditional bridges.

**5. Completion and Transparency**

The process is fully transparent. Users can track their transfer status in real-time, and fulfillers can monitor their fulfilled intents, fees earned, and settlement confirmations through the Speedrun interface and APIs.

**In short:** Speedrun separates _user experience_ from _settlement mechanics_, enabling fast transfers while preserving the security of decentralized cross-chain settlement.

<div align="center">
  <img src="/img/architecture.png" alt="Architecture" width="1400" />
</div>

## Components

### Smart Contracts

The smart contracts form the **core of Speedrun’s architecture**. They leverage **ZetaChain’s universal app framework** to support intent-based interoperability between chains. These contracts validate intents, enforce routing rules, and handle final settlement across connected chains.

- Handle intent creation and validation
- Perform routing and settlement on ZetaChain
- Deployed across supported EVMs

[Source code](hhttps://github.com/speedrun-hq/contracts-core)

### Backend API

Speedrun provides a robust **backend API** that exposes information about intent creation and fulfillment. This API can be used by:

- **Web developers** building Speedrun-enabled applications
- **Liquidity providers** creating custom fulfillers

It serves as the main off-chain indexing layer for tracking activity across the protocol.

- Registers and indexes new intents
- Tracks fulfillment status and fees
- Monitors cross-chain transaction status

[Source code](https://github.com/speedrun-hq/speedrun/tree/main/api)

### Fulfiller Tooling

Speedrun will include a suite of **open-source tooling** to allow anyone to participate in the network as a fulfiller.

- Scans for available intents
- Executes on-chain fulfillments
- Customize behavior (e.g. risk tolerance, liquidity use, strategy)

This tooling lowers the barrier to entry for new fulfillers and supports decentralized participation.

### Web Interface

The [**speedrun.exchange**](https://speedrun.exchange) webapp offers a simple, user-facing gateway to Speedrun. It allows users to:

- Initiate cross-chain transfers
- Monitor transaction status
- View intent and fulfillment history

In addition, it will feature a **dedicated dashboard for fulfillers**, providing performance insights and real-time metrics.

[Source code](https://github.com/speedrun-hq/speedrun/tree/main/frontend)

## Smart Contracts

### Intent Contract

The Intent contract is deployed on each connected blockchain (e.g., Base, Arbitrum) and serves as:

- **Entrypoint** for users initiating cross-chain transfers
- **Endpoint** for fulfilling completed transfers on the destination chain
- **Registry** for tracking pending intents and their fulfillment status

When a user wants to transfer tokens across chains, they interact with the Intent contract on their source chain. The contract initiates a ZetaChain crosschain transaction and emits an event that is observed by fulfillers. The goal is for the recipient to immediately receive tokens from the fulfiller, while the underlying crosschain transaction completes in the background.

### Router Contract

The Router contract is the central hub deployed on ZetaChain that:

- **Coordinates** cross-chain transfers between different blockchains
- **Manages** token associations between native tokens and their ZRC20 representations
- **Handles** gas token refunds to ensure smooth operation across chains

The Router maintains a registry of supported tokens and their corresponding ZRC20 addresses on ZetaChain. It orchestrates the entire flow of a cross-chain transfer by receiving messages from source chains, processing them through the swap module, and sending results to destination chains.

### Swap Module

The Swap Module is a specialized contract deployed on ZetaChain that:

- **Abstracts** the token swapping functionality for the Router
- **Integrates** with existing decentralized exchanges (DEXs) on ZetaChain
- **Optimizes** swap routes to minimize slippage and fees

Importantly, the platform itself is **not a DEX**. Instead, it leverages existing liquidity pools and decentralized exchanges on ZetaChain to efficiently convert between different ZRC20 tokens. This design choice allows the system to focus on its core competency - facilitating fast cross-chain transfers through the intent settlement protocol - while taking advantage of the established DeFi ecosystem on ZetaChain.
