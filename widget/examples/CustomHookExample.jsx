import React, { useEffect } from 'react';
import { useWidgetIntent } from '@speedrun/widget';
import { WagmiConfig, createConfig, configureChains } from 'wagmi';
import { publicProvider } from 'wagmi/providers/public';
import { arbitrum, base, mainnet } from 'wagmi/chains';
import { MetaMaskConnector } from 'wagmi/connectors/metaMask';

// Configure wagmi
const { chains, publicClient } = configureChains(
  [arbitrum, base, mainnet],
  [publicProvider()]
);

const config = createConfig({
  autoConnect: true,
  connectors: [new MetaMaskConnector({ chains })],
  publicClient,
});

// Custom styles
const styles = {
  container: {
    maxWidth: '600px',
    margin: '0 auto',
    padding: '20px',
    fontFamily: 'system-ui, sans-serif',
    borderRadius: '12px',
    boxShadow: '0 4px 12px rgba(0, 0, 0, 0.1)',
    background: 'white',
  },
  header: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: '20px',
  },
  title: {
    fontSize: '24px',
    fontWeight: 'bold',
    margin: 0,
  },
  form: {
    display: 'grid',
    gridTemplateColumns: '1fr 1fr',
    gap: '16px',
  },
  fullWidth: {
    gridColumn: '1 / -1',
  },
  formGroup: {
    marginBottom: '12px',
  },
  label: {
    display: 'block',
    marginBottom: '6px',
    fontWeight: '500',
  },
  select: {
    width: '100%',
    padding: '10px',
    borderRadius: '8px',
    border: '1px solid #ddd',
  },
  input: {
    width: '100%',
    padding: '10px',
    borderRadius: '8px',
    border: '1px solid #ddd',
  },
  button: {
    padding: '12px 16px',
    backgroundColor: '#0052ff',
    color: 'white',
    border: 'none',
    borderRadius: '8px',
    cursor: 'pointer',
    fontWeight: '600',
    marginTop: '16px',
  },
  buttonDisabled: {
    backgroundColor: '#cccccc',
    cursor: 'not-allowed',
  },
  notification: {
    padding: '12px',
    borderRadius: '8px',
    marginTop: '16px',
  },
  success: {
    backgroundColor: 'rgba(0, 200, 83, 0.1)',
    color: '#00833b',
  },
  error: {
    backgroundColor: 'rgba(255, 59, 48, 0.1)',
    color: '#c41e3a',
  },
};

// Custom chain selection component
function ChainDropdown({ value, onChange, options, label }) {
  return (
    <div style={styles.formGroup}>
      <label style={styles.label}>{label}</label>
      <select 
        style={styles.select} 
        value={value} 
        onChange={(e) => onChange(e.target.value)}
      >
        {options.map(option => (
          <option key={option} value={option}>
            {option}
          </option>
        ))}
      </select>
    </div>
  );
}

function CustomTransferForm() {
  // Use the widget intent hook
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

  // Available chains
  const chains = ['BASE', 'ARBITRUM', 'ETHEREUM', 'POLYGON', 'AVALANCHE', 'BSC'];
  
  // Available tokens (simplified for example)
  const tokens = ['USDC', 'USDT', 'ETH', 'DAI'];

  // Set default recipient to connected wallet
  useEffect(() => {
    if (isConnected && !formState.recipient) {
      // This would be your connected wallet address in a real implementation
      updateRecipient('0x1234...6789');
    }
  }, [isConnected, formState.recipient]);

  return (
    <div style={styles.container}>
      <div style={styles.header}>
        <h2 style={styles.title}>Custom Cross-Chain Transfer</h2>
      </div>

      <form onSubmit={handleSubmit} style={styles.form}>
        <ChainDropdown
          label="Source Chain"
          value={formState.sourceChain}
          onChange={updateSourceChain}
          options={chains}
        />

        <ChainDropdown
          label="Destination Chain"
          value={formState.destinationChain}
          onChange={updateDestinationChain}
          options={chains}
        />

        <div style={styles.formGroup}>
          <label style={styles.label}>Token</label>
          <select 
            style={styles.select} 
            value={formState.selectedToken} 
            onChange={(e) => updateToken(e.target.value)}
          >
            {tokens.map(token => (
              <option key={token} value={token}>{token}</option>
            ))}
          </select>
        </div>

        <div style={styles.formGroup}>
          <label style={styles.label}>
            Amount ({isLoading ? "Loading..." : `Available: ${balance} ${symbol}`})
          </label>
          <input
            type="number"
            value={formState.amount}
            onChange={(e) => updateAmount(e.target.value)}
            placeholder="0.00"
            style={styles.input}
          />
        </div>

        <div style={styles.formGroup}>
          <label style={styles.label}>Recipient Address</label>
          <input
            type="text"
            value={formState.recipient}
            onChange={(e) => updateRecipient(e.target.value)}
            placeholder="0x..."
            style={styles.input}
          />
        </div>

        <div style={styles.formGroup}>
          <label style={styles.label}>Fee Amount ({symbol})</label>
          <input
            type="number"
            value={formState.tip}
            onChange={(e) => updateTip(e.target.value)}
            placeholder="0.1"
            min="0.01"
            step="0.01"
            style={styles.input}
          />
        </div>

        {formState.error && (
          <div style={{...styles.notification, ...styles.error, ...styles.fullWidth}}>
            {formState.error.message}
          </div>
        )}

        {formState.success && formState.intentId && (
          <div style={{...styles.notification, ...styles.success, ...styles.fullWidth}}>
            Transfer initiated! Intent ID: {formState.intentId.slice(0, 8)}...
            {formState.fulfillmentTxHash && (
              <div>
                Fulfilled! Tx Hash: {formState.fulfillmentTxHash.slice(0, 8)}...
              </div>
            )}
          </div>
        )}

        <div style={styles.fullWidth}>
          <button
            type="submit"
            disabled={!isConnected || (!isValid && !formState.success) || formState.isSubmitting}
            style={{
              ...styles.button,
              ...((!isConnected || (!isValid && !formState.success) || formState.isSubmitting) ? styles.buttonDisabled : {})
            }}
          >
            {!isConnected
              ? "Connect Wallet"
              : formState.isSubmitting
                ? "Processing..."
                : formState.success
                  ? "Start New Transfer"
                  : "Transfer Tokens"}
          </button>
        </div>
      </form>
    </div>
  );
}

// Wrap the component with Wagmi provider
export default function CustomHookExample() {
  return (
    <WagmiConfig config={config}>
      <CustomTransferForm />
    </WagmiConfig>
  );
} 