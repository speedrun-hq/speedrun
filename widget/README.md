# Speedrun Widget for DeFi Applications

This widget allows DeFi applications to easily integrate Speedrun's intent-based cross-chain transfers directly into their user interface.

## Implementation Status

The Speedrun Widget is feature-complete and ready for integration, with a few remaining tasks:

- ✅ Independent hook-based architecture that works without frontend dependencies
- ✅ Real API integration with axios (with mock mode for testing)
- ✅ Comprehensive testing setup
- ✅ Multiple integration patterns supported (component, hook, API client)
- ✅ Full documentation and examples
- ✅ GitHub Actions workflow for publishing

For detailed implementation status, see [COMPLETED_ITEMS.md](./COMPLETED_ITEMS.md).  
For development setup instructions, see [SETUP_INSTRUCTIONS.md](./SETUP_INSTRUCTIONS.md).

![Speedrun Widget Screenshot](https://example.com/widget-screenshot.png)

## React Integration

Speedrun Widget is available as an NPM package for easy integration into React applications.

### Installation

```bash
# Using npm
npm install @speedrun/widget

# Using yarn
yarn add @speedrun/widget

# Using pnpm
pnpm add @speedrun/widget
```

### Basic Usage

```jsx
import { SpeedrunWidget } from '@speedrun/widget';
import { WagmiConfig, createConfig } from 'wagmi';
// ... your wagmi configuration

function App() {
  return (
    <WagmiConfig config={yourWagmiConfig}>
      <div className="app">
        <h1>My DeFi App</h1>
        
        <SpeedrunWidget
          defaultSourceChain="ARBITRUM"
          defaultDestinationChain="BASE"
          defaultToken="USDC"
          onSuccess={(intentId) => console.log(`Transfer initiated: ${intentId}`)}
        />
      </div>
    </WagmiConfig>
  );
}
```

## Advanced Integration Options

### Using the Standalone Hook

For applications that need more control, we provide the `useWidgetIntent` hook:

```jsx
import { useWidgetIntent } from '@speedrun/widget';
import { WagmiConfig } from 'wagmi';

function CustomTransfer() {
  const {
    formState,
    balance,
    isLoading,
    symbol,
    isConnected,
    isValid,
    handleSubmit,
    updateSourceChain,
    updateDestinationChain,
    updateToken,
    updateAmount,
    updateRecipient,
    updateTip,
    resetForm,
  } = useWidgetIntent();

  return (
    <div className="custom-transfer">
      <h2>Custom Transfer UI</h2>
      {/* Build your own UI with the hook's state and methods */}
      <form onSubmit={handleSubmit}>
        {/* Your custom form elements */}
        <button type="submit" disabled={!isValid || !isConnected}>
          Submit Transfer
        </button>
      </form>
    </div>
  );
}
```

### Customizing the API Client

The widget exports an API client that can be configured for custom endpoints:

```jsx
import { createApiClient, SpeedrunWidget } from '@speedrun/widget';

// Configure custom API endpoint (e.g., for staging or local development)
const customApiClient = createApiClient('https://api.staging.speedrun.exchange');

function App() {
  // Later you can use the widget with your customized services
  return <SpeedrunWidget />;
}
```

## Key Features

- **Cross-chain token transfers** - Enable users to send tokens across multiple blockchains
- **Intent-based architecture** - Fast transfers with settlement happening in the background
- **Highly customizable** - Style it to match your app's design system
- **Responsive layouts** - Compact and expanded views for different UI needs
- **Simple integration** - Just a few lines of code to get started

## Customization Options

The widget can be styled to match your application's design:

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

## Available Props

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

## Examples

Check out the [examples directory](./examples) for:
- Minimal integration example
- Integration with a DeFi app
- Complete documentation

## Requirements

- React 18.0+
- Wagmi 1.0+ for wallet connection

## Browser Support

The widget supports all modern browsers:
- Chrome, Firefox, Safari, Edge (latest 2 versions)
- Mobile browsers on iOS and Android

## License

MIT 