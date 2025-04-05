'use client';

import { useState, useEffect, useRef } from 'react';
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
  const tokens: TokenSymbol[] = ['USDC'];
  // Coming soon tokens
  const comingSoonTokens = ['BTC', 'USDT', 'ZETA'];
  const selectorRef = useRef<HTMLDivElement>(null);

  const handleClick = () => {
    if (disabled) return;
    setIsOpen(!isOpen);
  };

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

  return (
    <div className="relative w-full" ref={selectorRef}>
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
        >
          <div 
            className="bg-black border-2 border-yellow-500 rounded-lg overflow-hidden shadow-lg shadow-yellow-500/50"
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
            
            {/* Coming Soon Tokens */}
            {comingSoonTokens.map((token) => (
              <div
                key={token}
                className="w-full px-4 py-3 text-left arcade-text text-xs text-gray-500 cursor-not-allowed flex items-center justify-between"
              >
                <span>{token}</span>
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