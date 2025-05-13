# Quick Start Guide

This guide will help you get started with Speedrun quickly, whether you're a user, developer, or fulfiller.

## For Users

### Making a Cross-Chain Transfer

1. Visit [speedrun.exchange](https://speedrun.exchange)
2. Connect your wallet
3. Select the source and destination chains
4. Enter the amount and token
5. Confirm the transaction

The transfer will be completed in seconds, with ZetaChain providing security guarantees.

## For Developers

### Basic Integration

1. **Install Dependencies**
   ```bash
   npm install @speedrun/sdk
   ```

2. **Initialize the SDK**
   ```javascript
   import { Speedrun } from '@speedrun/sdk';

   const speedrun = new Speedrun({
     rpcUrl: 'YOUR_RPC_URL',
     chainId: 42161 // Arbitrum
   });
   ```

3. **Create an Intent**
   ```javascript
   const intent = await speedrun.createIntent({
     asset: '0xaf88d065e77c8cc2239327c5edb3a432268e5831', // USDC
     amount: '1000000000', // 1000 USDC
     targetChain: 8453, // Base
     receiver: '0x...', // Recipient address
     tip: '3000000' // 3 USDC tip
   });
   ```

4. **Track Intent Status**
   ```javascript
   const status = await speedrun.getIntentStatus(intent.id);
   ```

## For Fulfillers

### Setting Up a Fulfiller

1. **Install Fulfiller Tools**
   ```bash
   npm install @speedrun/fulfiller
   ```

2. **Configure Your Environment**
   ```bash
   cp .env.example .env
   # Edit .env with your configuration
   ```

3. **Start the Fulfiller**
   ```bash
   npm run start
   ```

4. **Monitor Fulfillments**
   ```bash
   # Check fulfillment status
   curl -X GET "https://api.speedrun.exchange/api/v1/fulfillments/YOUR_FULFILLMENT_ID"
   ```

## Next Steps

- Read the [Architecture Overview](./architecture.md) to understand how Speedrun works
- Check out the [API Reference](./api-reference.md) for detailed endpoint documentation
- Review the [Smart Contracts](./contract-addresses.md) to access contract addresses
- Follow the [Developer Guide](./initiate-intent.md) to learn more about how to build applications powered by intents
- Follow the [Fulfiller Guide](./fulfill-intents.md) to learn more about how to fulfill intents and get rewards

## Common Issues

### Transaction Failures
- Ensure sufficient gas on the source chain
- Verify token approvals
- Check for minimum transfer amounts

### API Errors
- Verify API key and permissions
- Check rate limits
- Ensure correct chain IDs

### Fulfillment Issues
- Verify liquidity on target chain
- Check gas prices
- Monitor for competing fulfillments

## Support

- [GitHub Issues](https://github.com/speedrun-hq/speedrun/issues)
- [Documentation](https://docs.speedrun.exchange) 