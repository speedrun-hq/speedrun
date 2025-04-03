'use client';

import { useState, useEffect } from 'react';
import { TokenSymbol } from '@/constants/tokens';

interface TokenSelectorProps {
  value: TokenSymbol;
  onChange: (value: TokenSymbol) => void;
  label?: string;
  disabled?: boolean;
}

export function TokenSelector({ 
  value, 
  onChange, 
  label = 'SELECT TOKEN', 
  disabled = false 
}: TokenSelectorProps) {
  const [isOpen, setIsOpen] = useState(false);
  const tokens: TokenSymbol[] = ['USDC', 'USDT'];

  const handleClick = () => {
    if (disabled) return;
    setIsOpen(!isOpen);
  };

  return (
    <div className="relative w-full">
      <button
        type="button"
        onClick={handleClick}
        disabled={disabled}
        className="w-full px-4 py-2 bg-black border-2 border-yellow-500 rounded-lg text-yellow-500 arcade-text text-xs focus:outline-none focus:border-yellow-400 flex justify-between items-center cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
      >
        <span>{value || label}</span>
        <span className="ml-2">{isOpen ? '▲' : '▼'}</span>
      </button>
      
      {isOpen && (
        <div 
          className="absolute top-full left-0 right-0 mt-2 z-[100]"
          onClick={() => setIsOpen(false)}
        >
          <div 
            className="bg-black border-2 border-yellow-500 rounded-lg overflow-hidden shadow-lg shadow-yellow-500/50"
            onClick={e => e.stopPropagation()}
          >
            {tokens.map((token) => (
              <button
                key={token}
                type="button"
                onClick={() => {
                  onChange(token);
                  setIsOpen(false);
                }}
                className={`w-full px-4 py-3 text-left arcade-text text-xs hover:bg-yellow-500 hover:text-black transition-colors cursor-pointer ${
                  token === value ? 'text-[#00ff00] bg-black/50' : 'text-yellow-500'
                }`}
              >
                {token}
              </button>
            ))}
          </div>
        </div>
      )}
    </div>
  );
} 