# Speedrun Widget for DeFi Applications

This widget allows DeFi applications to easily integrate Speedrun's intent-based cross-chain transfers directly into their user interface.

## Key Features

- **Cross-chain token transfers** - Enable users to send tokens across multiple blockchains
- **Intent-based architecture** - Fast transfers with settlement happening in the background
- **Highly customizable** - Style it to match your app's design system
- **Responsive layouts** - Compact and expanded views for different UI needs
- **Simple integration** - Just a few lines of code to get started

## Installation

```bash
npm install @speedrun/widget
# or
yarn add @speedrun/widget
```

## Basic Usage

```jsx
import { SpeedrunWidget } from '@speedrun/widget';

function MyApp() {
  return (
    <div>
      <h1>My DeFi App</h1>
      
      <SpeedrunWidget
        defaultSourceChain="BASE"
        defaultDestinationChain="ARBITRUM"
        defaultToken="USDC"
        onSuccess={(intentId) => console.log(`Transfer initiated: ${intentId}`)}
      />
    </div>
  );
}
```

## Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `defaultSourceChain` | string | `"BASE"` | Default source chain |
| `defaultDestinationChain` | string | `"ARBITRUM"` | Default destination chain |
| `defaultToken` | string | `"USDC"` | Default token to transfer |
| `defaultAmount` | string | `""` | Default amount to transfer |
| `defaultRecipient` | string | `""` | Default recipient address (wallet address of user if not specified) |
| `defaultTip` | string | `""` | Default fee for fulfiller |
| `onSuccess` | function | - | Callback when transfer is initiated `(intentId) => void` |
| `onError` | function | - | Callback when an error occurs `(error) => void` |
| `customStyles` | object | `{}` | Custom styling options |

### Custom Styling

The widget can be styled to match your application's design system:

```jsx
<SpeedrunWidget
  customStyles={{
    containerClass: "my-container-class",
    buttonClass: "my-button-class",
    inputClass: "my-input-class",
    labelClass: "my-label-class"
  }}
/>
```

## Integration Examples

### In a DEX Swap Interface

```jsx
function SwapInterface() {
  // Your DEX swap state
  const [selectedToken, setSelectedToken] = useState("USDC");
  const [amount, setAmount] = useState("");
  
  return (
    <div className="swap-container">
      {/* Your existing swap interface */}
      <div className="swap-form">
        {/* ... */}
      </div>
      
      {/* Cross-chain bridge option */}
      <div className="bridge-option">
        <h3>Need to bridge tokens first?</h3>
        <SpeedrunWidget
          defaultToken={selectedToken}
          defaultAmount={amount}
          customStyles={{
            containerClass: "my-dex-container",
            buttonClass: "my-dex-button"
          }}
        />
      </div>
    </div>
  );
}
```

### In a Wallet Application

```jsx
function WalletApp({ userAddress }) {
  const handleTransferSuccess = (intentId) => {
    // Update user's transaction history
    addToTransactionHistory({
      type: 'cross-chain-transfer',
      intentId,
      timestamp: Date.now()
    });
  };
  
  return (
    <div className="wallet-dashboard">
      <div className="wallet-balance">
        {/* Wallet balance display */}
      </div>
      
      <div className="actions-panel">
        <SpeedrunWidget
          defaultRecipient={userAddress}
          onSuccess={handleTransferSuccess}
          customStyles={{
            containerClass: "wallet-widget-container",
            buttonClass: "wallet-action-button"
          }}
        />
      </div>
    </div>
  );
}
```

## Benefits for DeFi Applications

Integrating cross-chain transfers directly into your DeFi app provides several benefits:

1. **Improved user experience** - Users don't need to leave your application to bridge tokens
2. **Increased engagement** - Solve the "wrong chain" problem without losing users to external bridging solutions
3. **New revenue opportunities** - Potential for revenue sharing on transfers initiated from your platform
4. **Differentiated offering** - Stand out from competitors by offering seamless cross-chain functionality

## Technical Details

### Architecture

The Speedrun Widget leverages Speedrun's intent-based architecture:

1. Users create transfer intents with parameters such as amount, destination chain, and recipient
2. Specialized fulfillers monitor and fulfill these intents in exchange for a fee
3. Users receive their tokens instantly from fulfillers while the cross-chain settlement happens in the background

### Required Dependencies

The widget requires:

- React 16.8+ (hooks support)
- Wagmi for wallet connection

### Browser Support

The widget supports all modern browsers:
- Chrome, Firefox, Safari, Edge (latest 2 versions)
- Mobile browsers on iOS and Android

## Development

To build the widget for development:

```bash
# Install dependencies
yarn install

# Start development build with watch mode
yarn dev

# Build for production
yarn build
```

## License

MIT 