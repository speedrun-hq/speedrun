'use client';

import { useState, useEffect, useRef } from 'react';
import { base, arbitrum, mainnet, bsc, polygon, avalanche } from 'wagmi/chains';

// Custom chain IDs for coming soon chains
const BITCOIN_CHAIN_ID = 9997;
const SOLANA_CHAIN_ID = 9999;
const SUI_CHAIN_ID = 9998;
const ZETACHAIN_CHAIN_ID = 7000; // ZetaChain actual mainnet ID
const TON_CHAIN_ID = 9996;       // TON custom ID

// Type to include both real and custom chain IDs
type ChainId = typeof mainnet.id | typeof bsc.id | typeof polygon.id | typeof base.id | typeof arbitrum.id | typeof avalanche.id | typeof BITCOIN_CHAIN_ID | typeof SOLANA_CHAIN_ID | typeof SUI_CHAIN_ID | typeof ZETACHAIN_CHAIN_ID | typeof TON_CHAIN_ID;

// Define color palette for dots
const chainColorMap: Record<number, string> = {
  [mainnet.id]: 'text-gray-400',
  [bsc.id]: 'text-yellow-400',
  [polygon.id]: 'text-purple-500',
  [base.id]: 'text-blue-400',
  [arbitrum.id]: 'text-blue-600',
  [avalanche.id]: 'text-red-600',
  [BITCOIN_CHAIN_ID]: 'text-orange-500',
  [SOLANA_CHAIN_ID]: 'text-purple-400',
  [SUI_CHAIN_ID]: 'text-teal-400',
  [ZETACHAIN_CHAIN_ID]: 'text-green-500',
  [TON_CHAIN_ID]: 'text-blue-500',
};

// Default border color for all selectors
const BORDER_COLOR = 'border-yellow-500';
const SHADOW_COLOR = 'shadow-yellow-500/50';
const TEXT_COLOR = 'text-yellow-500';

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
      { id: BITCOIN_CHAIN_ID, name: 'BITCOIN' },
      { id: TON_CHAIN_ID, name: 'TON' }
    ];
  } else {
    // Add ZetaChain to the TO selector
    comingSoonChains = [
      { id: ZETACHAIN_CHAIN_ID, name: 'ZETACHAIN' },
      { id: SOLANA_CHAIN_ID, name: 'SOLANA' },
      { id: SUI_CHAIN_ID, name: 'SUI' },
      { id: TON_CHAIN_ID, name: 'TON' }
    ];
  }
  
  const selectorRef = useRef<HTMLDivElement>(null);

  const handleClick = () => {
    if (disabled) return;
    setIsOpen(!isOpen);
  };

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

  const selectedChain = chains.find(chain => chain.id === value);
  
  return (
    <div className="relative w-full" ref={selectorRef}>
      <button
        type="button"
        onClick={handleClick}
        disabled={disabled}
        className={`w-full px-4 py-2 bg-black border-2 ${BORDER_COLOR} rounded-lg ${TEXT_COLOR} arcade-text text-xs focus:outline-none flex justify-between items-center cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed`}
      >
        <span className="flex items-center">
          {selectedChain && (
            <span className={`mr-2 inline-block text-xl leading-none ${chainColorMap[selectedChain.id]}`}>•</span>
          )}
          {selectedChain?.name || label}
        </span>
        <span className="ml-2">{isOpen ? '▲' : '▼'}</span>
      </button>
      
      {isOpen && (
        <div 
          className="absolute top-full left-0 right-0 mt-2 z-[1000]"
        >
          <div 
            className={`bg-black border-2 ${BORDER_COLOR} rounded-lg overflow-hidden shadow-lg ${SHADOW_COLOR} max-h-96 overflow-y-auto`}
          >
            {chains.map((chain) => (
              <button
                key={chain.id}
                type="button"
                onClick={() => {
                  onChange(chain.id);
                  setIsOpen(false);
                }}
                className={`w-full px-4 py-3 text-left arcade-text text-xs hover:bg-yellow-400 hover:text-black transition-colors cursor-pointer flex items-center ${
                  chain.id === value ? 'bg-black/50' : ''
                }`}
              >
                <span className={`mr-2 inline-block text-xl leading-none ${chainColorMap[chain.id]}`}>•</span>
                <span className={`${TEXT_COLOR}`}>{chain.name}</span>
              </button>
            ))}
            
            {/* Coming Soon Chains */}
            {comingSoonChains.map((chain) => (
              <div
                key={chain.id}
                className="w-full px-4 py-3 text-left arcade-text text-xs text-gray-500 cursor-not-allowed flex items-center justify-between"
              >
                <div className="flex items-center mr-2">
                  <span className={`mr-2 inline-block text-xl leading-none text-gray-500 flex-shrink-0`}>•</span>
                  <span>{chain.name}</span>
                </div>
                <span className="text-gray-500 opacity-70 text-[10px] whitespace-nowrap flex-shrink-0">
                  COMING SOON
                </span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
} 