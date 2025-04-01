'use client';

import { useState, useEffect } from 'react';
import { base, arbitrum } from 'wagmi/chains';

interface ChainSelectorProps {
  value: number;
  onChange: (value: number) => void;
  label?: string;
  disabled?: boolean;
}

export function ChainSelector({ value, onChange, label = 'SELECT CHAIN', disabled }: ChainSelectorProps) {
  const [isOpen, setIsOpen] = useState(false);
  const chains = [
    { id: base.id, name: 'BASE' },
    { id: arbitrum.id, name: 'ARBITRUM' }
  ];

  // Debug logs
  useEffect(() => {
    console.log('ChainSelector mounted with:', {
      currentValue: value,
      isOpen,
      availableChains: chains
    });
  }, [value, isOpen]);

  const handleClick = () => {
    if (disabled) return;
    console.log('Button clicked, current state:', { isOpen });
    setIsOpen(!isOpen);
  };

  return (
    <div className="relative w-full">
      <button
        type="button"
        onClick={handleClick}
        disabled={disabled}
        className="w-full px-4 py-2 bg-black border-2 border-yellow-500 rounded-lg text-yellow-500 font-mono focus:outline-none focus:border-yellow-400 flex justify-between items-center cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
      >
        <span>{chains.find(chain => chain.id === value)?.name || label}</span>
        <span className="ml-2">{isOpen ? '▲' : '▼'}</span>
      </button>
      
      {isOpen && (
        <div 
          className="absolute top-full left-0 right-0 mt-2 z-[100]"
          onClick={() => {
            console.log('Overlay clicked, closing dropdown');
            setIsOpen(false);
          }}
        >
          <div 
            className="bg-black border-2 border-yellow-500 rounded-lg overflow-hidden shadow-lg shadow-yellow-500/50"
            onClick={e => {
              console.log('Dropdown content clicked, preventing close');
              e.stopPropagation();
            }}
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
                className={`w-full px-4 py-3 text-left font-mono text-lg hover:bg-yellow-500 hover:text-black transition-colors cursor-pointer ${
                  chain.id === value ? 'text-[#00ff00] bg-black/50' : 'text-yellow-500'
                }`}
              >
                {chain.name}
              </button>
            ))}
          </div>
        </div>
      )}
    </div>
  );
} 