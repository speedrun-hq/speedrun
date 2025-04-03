'use client';

import { useState } from 'react';
import { ChainSelector } from './ChainSelector';
import { TokenSelector } from './TokenSelector';
import { FormInput } from './FormInput';
import ErrorMessage from '@/components/ErrorMessage';
import { useIntentForm } from '@/hooks/useIntentForm';
import { base } from 'wagmi/chains';
import { getChainId } from '@/utils/chain';
import { useAccount } from 'wagmi';

export default function CreateNewIntent() {
  const [showAdvanced, setShowAdvanced] = useState(false);
  const { address } = useAccount();
  
  const {
    formState,
    balance,
    isError,
    isLoading,
    symbol,
    isConnected,
    isValid,
    handleSubmit,
    updateSourceChain,
    updateDestinationChain,
    updateToken,
    updateAmount,
    updateRecipient,
    updateTip,
  } = useIntentForm();

  // Set default recipient to sender's address when connected and recipient is not set
  if (isConnected && address && !formState.recipient) {
    updateRecipient(address);
  }

  // Set default tip to recommended value (0.01) if not set
  if (!formState.tip) {
    updateTip('0.01');
  }
  
  const toggleAdvanced = () => {
    setShowAdvanced(!showAdvanced);
  };

  return (
    <div className="max-w-2xl mx-auto p-6 bg-black border-2 border-[hsl(var(--yellow))] rounded-lg shadow-lg relative z-0">
      <h2 className="text-2xl font-bold text-[hsl(var(--yellow))] mb-6 text-center font-mono">
        NEW TRANSFER
      </h2>
      
      <form
        onSubmit={handleSubmit}
        className="space-y-6 relative"
        role="form"
      >
        {formState.error && <ErrorMessage error={formState.error} className="mb-4" />}
        
        {formState.success && (
          <div className="bg-green-500/10 border border-green-500 text-green-500 p-4 rounded-lg mb-4">
            <p className="font-mono text-center">RUN CREATED SUCCESSFULLY!</p>
          </div>
        )}

        <div className="space-y-4">
          <div className="relative">
            <label className="block text-[hsl(var(--yellow))] mb-2 font-mono">FROM</label>
            <ChainSelector
              value={getChainId(formState.sourceChain)}
              onChange={(value) => updateSourceChain(value === base.id ? 'BASE' : 'ARBITRUM')}
              label="SELECT SOURCE CHAIN"
              disabled={formState.isSubmitting}
            />
          </div>

          <div className="relative">
            <label className="block text-[hsl(var(--yellow))] mb-2 font-mono">TO</label>
            <ChainSelector
              value={getChainId(formState.destinationChain)}
              onChange={(value) => updateDestinationChain(value === base.id ? 'BASE' : 'ARBITRUM')}
              label="SELECT DESTINATION CHAIN"
              disabled={formState.isSubmitting}
            />
          </div>

          <div className="relative">
            <label className="block text-[hsl(var(--yellow))] mb-2 font-mono">TOKEN</label>
            <TokenSelector
              value={formState.selectedToken}
              onChange={updateToken}
            />
          </div>

          <div className="relative">
            <div className="flex justify-between items-center mb-2">
              <label className="text-[hsl(var(--yellow))] font-mono">AMOUNT ({symbol})</label>
              <span className="text-[#00ff00] font-mono">
                Available: {isLoading ? 'Loading...' : balance ? `${balance} ${symbol}` : '0.00'}
              </span>
            </div>
            <FormInput
              type="number"
              value={formState.amount}
              onChange={updateAmount}
              placeholder="0.00"
              disabled={formState.isSubmitting}
              max={balance}
              step="0.01"
            />
            <p className="mt-2 text-[#00ff00] text-sm font-mono">fee: {formState.tip || '0.01'} {symbol}</p>
          </div>
          
          <div className="mt-4">
            <button
              type="button"
              onClick={toggleAdvanced}
              className="text-[hsl(var(--yellow))] text-sm font-mono hover:text-[hsl(var(--yellow)/0.8)]"
            >
              {showAdvanced ? '- HIDE ADVANCED OPTIONS' : '+ SHOW ADVANCED OPTIONS'}
            </button>
          </div>
          
          {showAdvanced && (
            <div className="space-y-4 pt-2 border-t border-gray-700">
              <div className="relative">
                <FormInput
                  label="CUSTOM RECIPIENT ADDRESS"
                  value={formState.recipient}
                  onChange={updateRecipient}
                  placeholder="0x..."
                  disabled={formState.isSubmitting}
                />
                <p className="text-xs text-gray-400 mt-1 font-mono">Default: Your wallet address</p>
              </div>

              <div className="relative">
                <div className="flex justify-between items-center mb-2">
                  <label className="text-[hsl(var(--yellow))] font-mono">CUSTOM FEE ({symbol})</label>
                  <span className="text-[#00ff00] font-mono">
                    Recommended: 0.01 {symbol}
                  </span>
                </div>
                <FormInput
                  type="number"
                  value={formState.tip}
                  onChange={updateTip}
                  placeholder="0.01"
                  disabled={formState.isSubmitting}
                  min="0.01"
                  step="0.01"
                />
                <div className="mt-2 text-xs text-gray-400 font-mono">
                  <p>Setting a lower fee may delay your transfer as speedrunners prioritize higher fees.</p>
                  <p className="mt-1">If the fee is too low, the network fees will be deducted from your transfer amount.</p>
                  <p className="mt-1">The default value is recommended for immediate processing.</p>
                  <p className="mt-1">
                    <a href="/about" className="text-[hsl(var(--yellow))] hover:underline">
                      Learn more about the intent-based architecture →
                    </a>
                  </p>
                </div>
              </div>
            </div>
          )}

          {formState.error && (
            <div className="text-red-500 text-sm font-mono">
              {formState.error.message}
            </div>
          )}
        </div>

        <button
          type="submit"
          disabled={!isConnected || !isValid || formState.isSubmitting}
          className="w-full arcade-btn bg-[hsl(var(--yellow))] text-black hover:bg-[hsl(var(--yellow)/0.8)] transition-colors duration-200 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {!isConnected ? 'CONNECT WALLET TO SUBMIT' : formState.isSubmitting ? 'APPROVING TOKENS...' : 'START'}
        </button>
      </form>
    </div>
  );
} 