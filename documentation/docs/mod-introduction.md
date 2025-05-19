# Introduction

## Overview

Modules are pre-built, ready-to-use implementations of the `IntentTarget` contract interface that handle common cross-chain operations. They allow developers to quickly integrate powerful cross-chain functionality without having to implement the intent target contracts themselves.

In the context of our platform, modules are pre-packaged solutions that:

1. Implement the `IntentTarget` interface described in the [Intent Contract Calls guide](./dev-intent-call.md)
2. Provide standardized implementations for common cross-chain operations
3. Can be directly referenced in your applications without requiring custom contract development
4. Support specific use cases like DEX swaps, liquidity provision, lending, and more

## Benefits of Using Modules

- **Reduced Development Time**: No need to write and audit custom intent target contracts
- **Enhanced Security**: All modules undergo thorough security audits
- **Standardized Interfaces**: Consistent interaction patterns across different operations
- **Optimized Gas Usage**: Implementations are optimized for efficiency
- **Regular Updates**: Modules are maintained and updated with new features

## Available Modules

Our platform offers several modules for common cross-chain operations:

- **Aerodrome**: Execute swaps on Aerodrome DEX on Base
