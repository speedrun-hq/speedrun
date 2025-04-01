'use client';

import { useState, useEffect } from 'react';

interface AmountInputProps {
  value: string;
  onChange: (value: string) => void;
  max?: string;
  disabled?: boolean;
}

export function AmountInput({ value, onChange, max, disabled }: AmountInputProps) {
  const [error, setError] = useState<string>('');

  useEffect(() => {
    if (!value) {
      setError('');
      return;
    }

    const numValue = parseFloat(value);
    if (isNaN(numValue) || numValue <= 0) {
      setError('Please enter a valid amount');
      return;
    }

    if (max && numValue > parseFloat(max)) {
      setError('Amount exceeds your balance');
      return;
    }

    setError('');
  }, [value, max]);

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const newValue = e.target.value;
    if (max && parseFloat(newValue) > parseFloat(max)) {
      return;
    }
    onChange(newValue);
  };

  return (
    <div className="space-y-2">
      <input
        type="number"
        value={value}
        onChange={handleChange}
        min="0"
        step="0.01"
        max={max}
        disabled={disabled}
        className="w-full px-4 py-2 bg-black border-2 border-yellow-500 rounded-lg text-yellow-500 font-mono focus:outline-none focus:border-yellow-400 disabled:opacity-50 disabled:cursor-not-allowed"
        placeholder="0.00"
      />
      {error && (
        <p className="text-red-500 text-sm font-mono">{error}</p>
      )}
    </div>
  );
} 