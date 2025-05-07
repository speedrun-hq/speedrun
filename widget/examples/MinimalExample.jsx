import React from 'react';
import { WagmiConfig, createConfig, configureChains } from 'wagmi';
import { publicProvider } from 'wagmi/providers/public';
import { arbitrum, base } from 'wagmi/chains';
import { MetaMaskConnector } from 'wagmi/connectors/metaMask';
import { SpeedrunWidget } from '@speedrun/widget';

// Minimal configuration for wagmi
const { chains, publicClient } = configureChains(
  [arbitrum, base],
  [publicProvider()]
);

const config = createConfig({
  autoConnect: true,
  connectors: [new MetaMaskConnector({ chains })],
  publicClient,
});

function MinimalExample() {
  return (
    <WagmiConfig config={config}>
      <div style={{ maxWidth: '500px', margin: '0 auto', padding: '20px' }}>
        <h1>Speedrun Widget Demo</h1>
        
        <div style={{ marginTop: '20px' }}>
          <SpeedrunWidget 
            defaultSourceChain="ARBITRUM"
            defaultDestinationChain="BASE"
            defaultToken="USDC"
            onSuccess={(intentId) => alert(`Transfer initiated! Intent ID: ${intentId}`)}
            onError={(error) => console.error('Error:', error.message)}
          />
        </div>
      </div>
    </WagmiConfig>
  );
}

export default MinimalExample; 