import './App.css'
import speedLogo from '/speed.png'

function App() {
  return (
    <div className="app">
      {/* Floating Chain Images */}
      <div className="floating-chains">
        <img src="/chains/eth.png" alt="Ethereum" className="floating-chain eth" />
        <img src="/chains/btc.png" alt="Bitcoin" className="floating-chain btc" />
        <img src="/chains/bnb.png" alt="BNB" className="floating-chain bnb" />
        <img src="/chains/pol.png" alt="Polygon" className="floating-chain pol" />
        <img src="/chains/base.png" alt="Base" className="floating-chain base" />
        <img src="/chains/arb.png" alt="Arbitrum" className="floating-chain arb" />
        <img src="/chains/ava.png" alt="Avalanche" className="floating-chain ava" />
        <img src="/chains/sol.png" alt="Solana" className="floating-chain sol" />
        <img src="/chains/zeta.png" alt="ZetaChain" className="floating-chain zeta" />
      </div>

      {/* Logo and Title - Top Left */}
      <div className="logo-title">
        <img 
          src={speedLogo} 
          alt="Speedrun Logo" 
          className="logo"
        />
        <h1 className="arcade-text title">
          Speedrun
        </h1>
      </div>

      {/* Centered Content Container */}
      <div className="centered-content">
        {/* Subtitle - Centered */}
        <p className="arcade-text subtitle">
          Blazing Fast Cross-Chain Transactions.<br />
          Zero Compromise.
        </p>
        
        {/* Description */}
        <div className="description">
          <p className="description-text">
            Speedrun is a cross-chain protocol that uses an intent-based architecture to make transactions across chains fast and affordable, powered by ZetaChain for secure and decentralized settlements.
          </p>
        </div>

        {/* Action Buttons - Side by Side */}
        <div className="button-container">
          <a 
            href="https://app.speedrun.exchange" 
            target="_blank" 
            rel="noopener noreferrer"
            className="arcade-btn"
          >
            Try Speedrun
          </a>
          
          <a 
            href="https://docs.speedrun.exchange" 
            target="_blank" 
            rel="noopener noreferrer"
            className="arcade-btn"
          >
            Integrate in your App
          </a>
        </div>
      </div>

      {/* Section Title */}
      <div className="section-title">
        <h2 className="arcade-text section-title-text">Why using Speedrun</h2>
      </div>

      {/* Characteristics */}
      <div className="characteristics">
        <div className="characteristic">
          <span className="characteristic-label">Fast</span>
          <span className="characteristic-desc">Cross-chain transfers in under 5 seconds</span>
        </div>
        <div className="characteristic">
          <span className="characteristic-label">Cheap</span>
          <span className="characteristic-desc">Competitive fulfiller network drives down costs</span>
        </div>
        <div className="characteristic">
          <span className="characteristic-label">Secure</span>
          <span className="characteristic-desc">ZetaChain robust interoperability protocol ensures settlement of intents</span>
        </div>
        <div className="characteristic">
          <span className="characteristic-label">Custom</span>
          <span className="characteristic-desc">Supports not only asset transfers but also custom cross-chain messaging</span>
        </div>
        <div className="characteristic">
          <span className="characteristic-label">Global</span>
          <span className="characteristic-desc">Leverage ZetaChain universal blockchain to connect to non-EVM chains, including Bitcoin</span>
        </div>
        <div className="characteristic">
          <span className="characteristic-label">Low-Risk</span>
          <span className="characteristic-desc">Unfulfilled intents automatically fall back to ZetaChain settlement with fee refund</span>
        </div>
      </div>

      {/* Roadmap Section */}
      <div className="roadmap-section">
        <h3 className="arcade-text section-title-text">Roadmap</h3>
        <p className="roadmap-text">Coming soon...</p>
      </div>

      {/* Community Section */}
      <div className="community-section">
        <h3 className="arcade-text section-title-text">Community</h3>
        <a 
          href="https://x.com/speedrun_hq" 
          target="_blank" 
          rel="noopener noreferrer"
          className="community-link"
        >
          Follow us on X
        </a>
      </div>

      {/* Footer */}
      <div className="footer">
        <div className="footer-content">
          <p className="footer-text">
            © 2025 Speedrun
          </p>
          <a 
            href="https://www.zetachain.com/"
            target="_blank" 
            rel="noopener noreferrer"
            className="footer-link"
          >
            Powered by ZetaChain
          </a>
        </div>
      </div>
    </div>
  )
}

export default App
