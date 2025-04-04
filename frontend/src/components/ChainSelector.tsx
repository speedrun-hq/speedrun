'use client';

import { useState, useEffect, useRef } from 'react';
import { base, arbitrum } from 'wagmi/chains';

// Custom chain IDs for coming soon chains
const BITCOIN_CHAIN_ID = 9997;
const SOLANA_CHAIN_ID = 9999;
const SUI_CHAIN_ID = 9998;

// Type to include both real and custom chain IDs
type ChainId = typeof base.id | typeof arbitrum.id | typeof BITCOIN_CHAIN_ID | typeof SOLANA_CHAIN_ID | typeof SUI_CHAIN_ID;

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
  
  const chains: {id: ChainId, name: string}[] = [
    { id: base.id, name: 'BASE' },
    { id: arbitrum.id, name: 'ARBITRUM' }
  ];
  
  // Coming soon chains with placeholder IDs - different based on selector type
  let comingSoonChains: {id: ChainId, name: string}[] = [];
  
  if (selectorType === 'from') {
    // Bitcoin is "coming soon" in the FROM selector
    comingSoonChains = [
      { id: SOLANA_CHAIN_ID, name: 'SOLANA' },
      { id: SUI_CHAIN_ID, name: 'SUI' },
      { id: BITCOIN_CHAIN_ID, name: 'BITCOIN' }
    ];
  } else {
    // Bitcoin doesn't appear at all in the TO selector
    comingSoonChains = [
      { id: SOLANA_CHAIN_ID, name: 'SOLANA' },
      { id: SUI_CHAIN_ID, name: 'SUI' }
    ];
  }
  
  const selectorRef = useRef<HTMLDivElement>(null);

  // Debug logs
  useEffect(() => {
    console.log('ChainSelector mounted with:', {
      currentValue: value,
      isOpen,
      availableChains: chains,
      comingSoonChains,
      selectorType
    });
  }, [value, isOpen, selectorType]);

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
    console.log('Button clicked, current state:', { isOpen });
    setIsOpen(!isOpen);
  };

  return (
    <div className="relative w-full" ref={selectorRef}>
      <button
        type="button"
        onClick={handleClick}
        disabled={disabled}
        className="w-full px-4 py-2 bg-black border-2 border-yellow-500 rounded-lg text-yellow-500 arcade-text text-xs focus:outline-none focus:border-yellow-400 flex justify-between items-center cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
      >
        <span>{chains.find(chain => chain.id === value)?.name || label}</span>
        <span className="ml-2">{isOpen ? '▲' : '▼'}</span>
      </button>
      
      {isOpen && (
        <div 
          className="absolute top-full left-0 right-0 mt-2 z-[100]"
        >
          <div 
            className="bg-black border-2 border-yellow-500 rounded-lg overflow-hidden shadow-lg shadow-yellow-500/50"
          >
            {chains.map((chain) => (
              <button
                key={chain.id}
                type="button"
                onClick={() => {
                  console.log('Chain selected:', chain.name);
                  onChange(chain.id);
                  setIsOpen(false);
                }}
                className={`w-full px-4 py-3 text-left arcade-text text-xs hover:bg-yellow-500 hover:text-black transition-colors cursor-pointer ${
                  chain.id === value ? 'text-[#00ff00] bg-black/50' : 'text-yellow-500'
                }`}
              >
                {chain.name}
              </button>
            ))}
            
            {/* Coming Soon Chains */}
            {comingSoonChains.map((chain) => (
              <div
                key={chain.id}
                className="w-full px-4 py-3 text-left arcade-text text-xs text-gray-500 cursor-not-allowed flex justify-between items-center"
              >
                <span>{chain.name}</span>
                <span className="text-yellow-500 opacity-70 text-[10px]">COMING SOON</span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
} 