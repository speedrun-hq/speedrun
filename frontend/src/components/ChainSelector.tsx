'use client';

import { useState, useEffect, useRef } from 'react';
import { base, arbitrum, mainnet, bsc, polygon, avalanche } from 'wagmi/chains';

// Custom chain IDs for coming soon chains
const BITCOIN_CHAIN_ID = 9997;
const SOLANA_CHAIN_ID = 9999;
const SUI_CHAIN_ID = 9998;
const ZETACHAIN_CHAIN_ID = 7000; // ZetaChain actual mainnet ID

// Type to include both real and custom chain IDs
type ChainId = typeof mainnet.id | typeof bsc.id | typeof polygon.id | typeof base.id | typeof arbitrum.id | typeof avalanche.id | typeof BITCOIN_CHAIN_ID | typeof SOLANA_CHAIN_ID | typeof SUI_CHAIN_ID | typeof ZETACHAIN_CHAIN_ID;

// Define color palette just for text colors of each chain
const chainColorMap: Record<number, { text: string, hoverBg: string }> = {
  [mainnet.id]: { text: 'text-blue-500', hoverBg: 'hover:bg-blue-500' },
  [bsc.id]: { text: 'text-yellow-400', hoverBg: 'hover:bg-yellow-400' },
  [polygon.id]: { text: 'text-purple-500', hoverBg: 'hover:bg-purple-500' },
  [base.id]: { text: 'text-blue-400', hoverBg: 'hover:bg-blue-400' },
  [arbitrum.id]: { text: 'text-blue-600', hoverBg: 'hover:bg-blue-600' },
  [avalanche.id]: { text: 'text-red-600', hoverBg: 'hover:bg-red-600' },
  [BITCOIN_CHAIN_ID]: { text: 'text-orange-500', hoverBg: 'hover:bg-orange-500' },
  [SOLANA_CHAIN_ID]: { text: 'text-purple-400', hoverBg: 'hover:bg-purple-400' },
  [SUI_CHAIN_ID]: { text: 'text-teal-400', hoverBg: 'hover:bg-teal-400' },
  [ZETACHAIN_CHAIN_ID]: { text: 'text-green-500', hoverBg: 'hover:bg-green-500' },
};

// Default border color for all selectors
const BORDER_COLOR = 'border-yellow-400';
const SHADOW_COLOR = 'shadow-yellow-400/50';

interface ChainSelectorProps {
  value: number;
  onChange: (value: number) => void;
  label?: string;
  disabled?: boolean;
  selectorType?: 'from' | 'to'; // Indicates whether this is a source or destination selector
}

export function ChainSelector({ 
  value, 
  onChange, 
  label = 'SELECT CHAIN', 
  disabled,
  selectorType = 'from' // Default to 'from' if not specified
}: ChainSelectorProps) {
  const [isOpen, setIsOpen] = useState(false);
  
  // All supported chains
  const chains: {id: ChainId, name: string}[] = [
    { id: mainnet.id, name: 'ETHEREUM' },
    { id: bsc.id, name: 'BSC' },
    { id: polygon.id, name: 'POLYGON' },
    { id: base.id, name: 'BASE' },
    { id: arbitrum.id, name: 'ARBITRUM' },
    { id: avalanche.id, name: 'AVALANCHE' }
  ];
  
  // Coming soon chains with placeholder IDs - different based on selector type
  let comingSoonChains: {id: ChainId, name: string}[] = [];
  
  if (selectorType === 'from') {
    // Add ZetaChain to the FROM selector
    comingSoonChains = [
      { id: ZETACHAIN_CHAIN_ID, name: 'ZETACHAIN' },
      { id: SOLANA_CHAIN_ID, name: 'SOLANA' },
      { id: SUI_CHAIN_ID, name: 'SUI' },
      { id: BITCOIN_CHAIN_ID, name: 'BITCOIN' }
    ];
  } else {
    // Add ZetaChain to the TO selector
    comingSoonChains = [
      { id: ZETACHAIN_CHAIN_ID, name: 'ZETACHAIN' },
      { id: SOLANA_CHAIN_ID, name: 'SOLANA' },
      { id: SUI_CHAIN_ID, name: 'SUI' }
    ];
  }
  
  const selectorRef = useRef<HTMLDivElement>(null);

  // Get text color for the selected chain
  const selectedChain = chains.find(chain => chain.id === value);
  const textColor = selectedChain 
    ? chainColorMap[selectedChain.id].text 
    : 'text-yellow-400';

  // Handle click outside to close dropdown
  useEffect(() => {
    function handleOutsideClick(event: MouseEvent) {
      if (selectorRef.current && !selectorRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    }

    if (isOpen) {
      document.addEventListener('mousedown', handleOutsideClick);
    }

    return () => {
      document.removeEventListener('mousedown', handleOutsideClick);
    };
  }, [isOpen]);

  const handleClick = () => {
    if (disabled) return;
    setIsOpen(!isOpen);
  };

  return (
    <div className="relative w-full" ref={selectorRef}>
      <button
        type="button"
        onClick={handleClick}
        disabled={disabled}
        className={`w-full px-4 py-2 bg-black border-2 ${BORDER_COLOR} rounded-lg ${textColor} arcade-text text-xs focus:outline-none flex justify-between items-center cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed`}
      >
        <span>{chains.find(chain => chain.id === value)?.name || label}</span>
        <span className="ml-2">{isOpen ? '▲' : '▼'}</span>
      </button>
      
      {isOpen && (
        <div 
          className="absolute top-full left-0 right-0 mt-2 z-[1000]"
        >
          <div 
            className={`bg-black border-2 ${BORDER_COLOR} rounded-lg overflow-hidden shadow-lg ${SHADOW_COLOR}`}
          >
            {chains.map((chain) => {
              const chainColor = chainColorMap[chain.id];
              return (
                <button
                  key={chain.id}
                  type="button"
                  onClick={() => {
                    onChange(chain.id);
                    setIsOpen(false);
                  }}
                  className={`w-full px-4 py-3 text-left arcade-text text-xs ${chainColor.hoverBg} hover:text-black transition-colors cursor-pointer ${
                    chain.id === value ? 'text-white bg-black/50' : chainColor.text
                  }`}
                >
                  {chain.name}
                </button>
              );
            })}
            
            {/* Coming Soon Chains */}
            {comingSoonChains.map((chain) => {
              const chainColor = chainColorMap[chain.id];
              return (
                <div
                  key={chain.id}
                  className="w-full px-4 py-3 text-left arcade-text text-xs text-gray-500 cursor-not-allowed flex justify-between items-center"
                >
                  <span>{chain.name}</span>
                  <span className={`${chainColor.text} opacity-70 text-[10px]`}>
                    COMING SOON
                  </span>
                </div>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
} 