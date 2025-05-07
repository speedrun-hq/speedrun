# Integrating the Speedrun Widget in React Applications

This guide shows how to integrate the Speedrun cross-chain transfer widget into your React application.

## Installation

```bash
# Using npm
npm install @speedrun/widget

# Using yarn
yarn add @speedrun/widget

# Using pnpm
pnpm add @speedrun/widget
```

## Basic Usage

```jsx
import React from 'react';
import { SpeedrunWidget } from '@speedrun/widget';

function App() {
  // Handle successful intent creation
  const handleSuccess = (intentId) => {
    console.log(`Transfer initiated with intent ID: ${intentId}`);
    // Update your app state or UI as needed
  };

  // Handle errors
  const handleError = (error) => {
    console.error('Transfer error:', error.message);
    // Show error notification to the user
  };

  return (
    <div className="app">
      <h1>My DeFi Application</h1>
      
      <div className="widget-container">
        <h2>Cross-Chain Transfer</h2>
        <SpeedrunWidget
          defaultSourceChain="ARBITRUM"
          defaultDestinationChain="BASE"
          defaultToken="USDC"
          onSuccess={handleSuccess}
          onError={handleError}
        />
      </div>
    </div>
  );
}

export default App;
```

## Customizing the Widget

You can customize the appearance of the widget to match your application's design:

```jsx
import { SpeedrunWidget } from '@speedrun/widget';

function App() {
  // Custom styling to match your app's design system
  const customStyles = {
    containerClass: "border border-blue-200 rounded-xl p-5 bg-blue-50 shadow-sm",
    buttonClass: "w-full bg-blue-600 text-white py-3 rounded-lg font-medium hover:bg-blue-700 transition-colors",
    inputClass: "border rounded p-2 w-full focus:border-blue-500 focus:ring-1 focus:ring-blue-500",
    labelClass: "text-blue-800 font-medium"
  };
  
  return (
    <div className="app">
      <SpeedrunWidget
        customStyles={customStyles}
        defaultSourceChain="ARBITRUM"
        defaultToken="USDC"
        onSuccess={(intentId) => console.log(`Transfer initiated: ${intentId}`)}
      />
    </div>
  );
}
```

## Integration with State Management

Here's how to integrate with React state and form data:

```jsx
import React, { useState } from 'react';
import { SpeedrunWidget } from '@speedrun/widget';

function SwapInterface() {
  // Your existing DEX state
  const [selectedToken, setSelectedToken] = useState("USDC");
  const [amount, setAmount] = useState("");
  const [notification, setNotification] = useState(null);
  
  const handleSuccess = (intentId) => {
    setNotification({
      type: 'success',
      message: `Transfer initiated! Track it with ID: ${intentId}`
    });
    
    // Clear notification after 5 seconds
    setTimeout(() => setNotification(null), 5000);
  };
  
  return (
    <div className="swap-container">
      {/* Your existing swap interface */}
      <div className="swap-form">
        <input 
          type="text"
          value={amount}
          onChange={(e) => setAmount(e.target.value)}
          placeholder="Amount"
        />
        <select 
          value={selectedToken}
          onChange={(e) => setSelectedToken(e.target.value)}
        >
          <option value="USDC">USDC</option>
          <option value="USDT">USDT</option>
        </select>
        <button>Swap</button>
      </div>
      
      {notification && (
        <div className={`notification ${notification.type}`}>
          {notification.message}
        </div>
      )}
      
      {/* Cross-chain bridge option */}
      <div className="bridge-option">
        <h3>Need to bridge tokens first?</h3>
        <SpeedrunWidget
          defaultToken={selectedToken}
          defaultAmount={amount}
          onSuccess={handleSuccess}
        />
      </div>
    </div>
  );
}
```

## Advanced: Wallet Integration

The widget uses wagmi for wallet integration. If your app already uses wagmi, they'll share the same connection:

```jsx
import { WagmiConfig, createConfig } from 'wagmi';
import { mainnet, arbitrum, base } from 'wagmi/chains';
import { SpeedrunWidget } from '@speedrun/widget';

// Your wagmi config
const config = createConfig({
  // ...your wagmi configuration
});

function App() {
  return (
    <WagmiConfig config={config}>
      <div className="app">
        <SpeedrunWidget
          defaultSourceChain="ARBITRUM"
          defaultDestinationChain="BASE"
        />
      </div>
    </WagmiConfig>
  );
}
```

## Advanced: Using the Standalone Hook

For applications that need more control over the UI, we provide a standalone hook that gives you access to all the state and functions needed to build a custom transfer interface:

```jsx
import { useWidgetIntent } from '@speedrun/widget';

function CustomTransferInterface() {
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
      <h2>Custom Transfer Interface</h2>
      <form onSubmit={handleSubmit}>
        {/* Create your own form UI using the hook state and methods */}
        <div className="form-field">
          <label>From Chain</label>
          <select 
            value={formState.sourceChain} 
            onChange={e => updateSourceChain(e.target.value)}
          >
            <option value="ARBITRUM">Arbitrum</option>
            <option value="BASE">Base</option>
            {/* Add other chains */}
          </select>
        </div>
        
        {/* Add other form fields for destination chain, token, amount, etc. */}
        
        <button 
          type="submit" 
          disabled={!isConnected || !isValid || formState.isSubmitting}
        >
          {formState.isSubmitting ? "Submitting..." : "Transfer"}
        </button>
        
        {formState.success && (
          <div className="success-message">
            Transfer initiated! Intent ID: {formState.intentId}
          </div>
        )}
      </form>
    </div>
  );
}
```

Check out the full example in [CustomHookExample.jsx](./CustomHookExample.jsx) for a complete implementation.

## Advanced: Customizing the API Client

For applications that need to use a custom API endpoint (e.g., for staging environments), we provide access to the API client:

```jsx
import { createApiClient } from '@speedrun/widget';

// Configure a custom API endpoint
const apiClient = createApiClient('https://api.staging.speedrun.exchange');

// Now you can use this client with your custom integration
// The SpeedrunWidget and useWidgetIntent will automatically use the configured client
```

## All Available Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `defaultSourceChain` | string | `"BASE"` | Default source chain |
| `defaultDestinationChain` | string | `"ARBITRUM"` | Default destination chain |
| `defaultToken` | string | `"USDC"` | Default token to transfer |
| `defaultAmount` | string | `""` | Default amount to transfer |
| `defaultRecipient` | string | `""` | Default recipient address |
| `defaultTip` | string | `""` | Default fee for fulfiller |
| `onSuccess` | function | - | Callback when transfer is initiated |
| `onError` | function | - | Callback when an error occurs |
| `customStyles` | object | `{}` | Custom styling options |

## Need Help?

For additional support or questions, please visit [our documentation](https://docs.speedrun.exchange) or open an issue on our [GitHub repository](https://github.com/speedrun-hq/speedrun). 