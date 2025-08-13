import './App.css'
import speedLogo from '/speed.png'

function App() {
  return (
    <div className="app">
      <div className="content">
        {/* Logo and Title */}
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
        
        {/* Subtitle */}
        <p className="arcade-text subtitle">
          Blazing Fast Cross-Chain Transactions. Zero Compromise.
        </p>
        
        {/* Action Buttons */}
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

        {/* Description */}
        <div className="description">
          <p className="description-text">
            Speedrun is a novel cross-chain protocol that uses an intent-based architecture to make transactions across chains fast and affordable, powered by ZetaChain for secure and decentralized settlements.
          </p>
        </div>

        {/* Characteristics */}
        <div className="characteristics">
          <div className="characteristic">
            <span className="arrow">→</span>
            <span className="characteristic-label">Fast</span>
            <span className="characteristic-desc">Cross-chain transfers in under 5 seconds</span>
          </div>
          <div className="characteristic">
            <span className="arrow">→</span>
            <span className="characteristic-label">Cheap</span>
            <span className="characteristic-desc">Competitive fulfiller network drives down costs</span>
          </div>
          <div className="characteristic">
            <span className="arrow">→</span>
            <span className="characteristic-label">Programmable</span>
            <span className="characteristic-desc">Supports not only asset transfers but also cross-chain messaging</span>
          </div>
        </div>
      </div>
    </div>
  )
}

export default App
