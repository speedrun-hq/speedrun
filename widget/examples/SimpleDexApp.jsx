import React, { useState } from 'react';
import { SpeedrunWidget } from '@speedrun/widget';
import { WagmiConfig, createConfig, configureChains } from 'wagmi';
import { publicProvider } from 'wagmi/providers/public';
import { arbitrum, base, mainnet } from 'wagmi/chains';
import { MetaMaskConnector } from 'wagmi/connectors/metaMask';

// Configure chains & providers
const { chains, publicClient, webSocketPublicClient } = configureChains(
  [arbitrum, base, mainnet],
  [publicProvider()]
);

// Set up wagmi config
const config = createConfig({
  autoConnect: true,
  connectors: [
    new MetaMaskConnector({ chains })
  ],
  publicClient,
  webSocketPublicClient,
});

function SimpleDexApp() {
  // App state
  const [activeTab, setActiveTab] = useState('swap');
  const [amount, setAmount] = useState('');
  const [selectedToken, setSelectedToken] = useState('USDC');
  const [notification, setNotification] = useState(null);
  
  // Handle form submission for the swap
  const handleSwapSubmit = (e) => {
    e.preventDefault();
    setNotification({
      type: 'info',
      message: `Swap request submitted for ${amount} ${selectedToken}`
    });
    setTimeout(() => setNotification(null), 3000);
  };
  
  // Handle successful intent creation
  const handleTransferSuccess = (intentId) => {
    setNotification({
      type: 'success',
      message: `Transfer initiated! Track it with ID: ${intentId.slice(0,8)}...`
    });
    setTimeout(() => setNotification(null), 5000);
  };
  
  // Handle transfer errors
  const handleTransferError = (error) => {
    setNotification({
      type: 'error',
      message: `Error: ${error.message}`
    });
    setTimeout(() => setNotification(null), 5000);
  };
  
  return (
    <WagmiConfig config={config}>
      <div className="dex-app">
        {/* App Header */}
        <header className="app-header">
          <h1>SimpleDEX</h1>
          <nav className="tabs">
            <button 
              className={activeTab === 'swap' ? 'active' : ''} 
              onClick={() => setActiveTab('swap')}
            >
              Swap
            </button>
            <button 
              className={activeTab === 'bridge' ? 'active' : ''} 
              onClick={() => setActiveTab('bridge')}
            >
              Bridge
            </button>
          </nav>
        </header>
        
        {/* Notification area */}
        {notification && (
          <div className={`notification ${notification.type}`}>
            {notification.message}
          </div>
        )}
        
        {/* Main content area */}
        <main className="app-content">
          {activeTab === 'swap' ? (
            <div className="swap-container">
              <h2>Swap Tokens</h2>
              <form onSubmit={handleSwapSubmit} className="swap-form">
                <div className="form-group">
                  <label>Amount</label>
                  <input 
                    type="text" 
                    value={amount} 
                    onChange={(e) => setAmount(e.target.value)}
                    placeholder="0.00"
                  />
                </div>
                
                <div className="form-group">
                  <label>Token</label>
                  <select 
                    value={selectedToken}
                    onChange={(e) => setSelectedToken(e.target.value)}
                  >
                    <option value="USDC">USDC</option>
                    <option value="USDT">USDT</option>
                    <option value="ETH">ETH</option>
                  </select>
                </div>
                
                <button type="submit" className="swap-button">
                  Swap
                </button>
              </form>
              
              <div className="bridge-cta">
                <p>Need tokens on another chain?</p>
                <button onClick={() => setActiveTab('bridge')}>
                  Use Cross-Chain Bridge
                </button>
              </div>
            </div>
          ) : (
            <div className="bridge-container">
              <h2>Cross-Chain Bridge</h2>
              <p>Transfer tokens between blockchain networks instantly</p>
              
              {/* Speedrun Widget Integration */}
              <SpeedrunWidget
                defaultSourceChain="ARBITRUM"
                defaultDestinationChain="BASE"
                defaultToken={selectedToken}
                defaultAmount={amount}
                onSuccess={handleTransferSuccess}
                onError={handleTransferError}
                customStyles={{
                  containerClass: "widget-container",
                  buttonClass: "widget-button",
                  inputClass: "widget-input",
                  labelClass: "widget-label"
                }}
              />
            </div>
          )}
        </main>
        
        {/* App Footer */}
        <footer className="app-footer">
          <p>
            Widget powered by <a href="https://speedrun.exchange" target="_blank" rel="noopener noreferrer">Speedrun</a>
          </p>
        </footer>
      </div>
      
      {/* Example CSS (would normally be in a separate file) */}
      <style jsx>{`
        .dex-app {
          font-family: system-ui, -apple-system, sans-serif;
          max-width: 1200px;
          margin: 0 auto;
          padding: 20px;
        }
        
        .app-header {
          display: flex;
          justify-content: space-between;
          align-items: center;
          margin-bottom: 20px;
          padding-bottom: 10px;
          border-bottom: 1px solid #eaeaea;
        }
        
        .tabs {
          display: flex;
          gap: 10px;
        }
        
        .tabs button {
          padding: 8px 16px;
          background: #f5f5f5;
          border: 1px solid #ddd;
          border-radius: 6px;
          cursor: pointer;
        }
        
        .tabs button.active {
          background: #3b82f6;
          color: white;
          border-color: #3b82f6;
        }
        
        .notification {
          padding: 10px 15px;
          margin-bottom: 20px;
          border-radius: 6px;
        }
        
        .notification.success {
          background-color: #d1fae5;
          color: #065f46;
          border: 1px solid #34d399;
        }
        
        .notification.error {
          background-color: #fee2e2;
          color: #991b1b;
          border: 1px solid #f87171;
        }
        
        .notification.info {
          background-color: #e0f2fe;
          color: #075985;
          border: 1px solid #7dd3fc;
        }
        
        .app-content {
          background: white;
          border-radius: 12px;
          box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1);
          padding: 20px;
        }
        
        .swap-form {
          display: flex;
          flex-direction: column;
          gap: 15px;
          margin-bottom: 20px;
        }
        
        .form-group {
          display: flex;
          flex-direction: column;
          gap: 5px;
        }
        
        .form-group label {
          font-size: 14px;
          font-weight: 500;
          color: #6b7280;
        }
        
        input, select {
          padding: 10px 12px;
          border: 1px solid #d1d5db;
          border-radius: 6px;
          font-size: 16px;
        }
        
        .swap-button {
          padding: 12px;
          background: #3b82f6;
          color: white;
          border: none;
          border-radius: 6px;
          font-weight: 500;
          cursor: pointer;
          margin-top: 10px;
        }
        
        .swap-button:hover {
          background: #2563eb;
        }
        
        .bridge-cta {
          margin-top: 30px;
          padding-top: 20px;
          border-top: 1px dashed #e5e7eb;
          text-align: center;
        }
        
        .bridge-cta button {
          margin-top: 10px;
          padding: 8px 16px;
          background: none;
          border: 1px solid #3b82f6;
          color: #3b82f6;
          border-radius: 6px;
          cursor: pointer;
        }
        
        .bridge-cta button:hover {
          background: #eff6ff;
        }
        
        .app-footer {
          margin-top: 40px;
          text-align: center;
          color: #6b7280;
          font-size: 14px;
        }
        
        .app-footer a {
          color: #3b82f6;
          text-decoration: none;
        }
        
        /* Custom styling for the widget */
        :global(.widget-container) {
          border: 1px solid #e5e7eb;
          border-radius: 12px;
          padding: 20px;
          background: #f9fafb;
        }
        
        :global(.widget-button) {
          background: #3b82f6;
          color: white;
          padding: 12px;
          border-radius: 6px;
          font-weight: 500;
          width: 100%;
        }
        
        :global(.widget-input) {
          border: 1px solid #d1d5db;
          border-radius: 6px;
          padding: 10px 12px;
        }
        
        :global(.widget-label) {
          color: #4b5563;
          font-weight: 500;
          font-size: 14px;
        }
      `}</style>
    </WagmiConfig>
  );
}

export default SimpleDexApp; 