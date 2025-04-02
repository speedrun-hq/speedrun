import { useState, useEffect } from 'react';
import { TokenSymbol } from '@/constants/tokens';

interface TokenSelectorProps {
  value: TokenSymbol;
  onChange: (value: TokenSymbol) => void;
}

export function TokenSelector({ value, onChange }: TokenSelectorProps) {
  const tokens: TokenSymbol[] = ['USDC', 'USDT'];

  // Basic debug log
  console.log('TokenSelector render:', { value });

  return (
    <div className="relative w-full">
      <select
        value={value}
        onChange={(e) => {
          console.log('Select changed:', e.target.value);
          onChange(e.target.value as TokenSymbol);
        }}
        className="w-full px-4 py-2 bg-black border-2 border-yellow-500 rounded-lg text-yellow-500 font-mono focus:outline-none focus:border-yellow-400"
      >
        {tokens.map((token) => (
          <option key={token} value={token}>
            {token}
          </option>
        ))}
      </select>
    </div>
  );
} 