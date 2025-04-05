'use client';

import { useState } from 'react';
import { ChainSelector } from './ChainSelector';
import { TokenSelector } from './TokenSelector';
import { FormInput } from './FormInput';
import ErrorMessage from '@/components/ErrorMessage';
import PendingAnimation from '@/components/PendingAnimation';
import CompletedAnimation from '@/components/CompletedAnimation';
import { useIntentForm } from '@/hooks/useIntentForm';
import { getChainId, getChainName, getExplorerUrl } from '@/utils/chain';
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
    fulfillmentTxHash,
    resetForm
  } = useIntentForm();

  // Set default recipient to sender's address when connected and recipient is not set
  if (isConnected && address && !formState.recipient) {
    updateRecipient(address);
  }

  // Set default tip to recommended value (0.1) if not set
  if (!formState.tip) {
    updateTip('0.1');
  }
  
  const toggleAdvanced = () => {
    setShowAdvanced(!showAdvanced);
  };

  // Handle chain selection
  const handleSourceChainChange = (chainId: number) => {
    const chainName = getChainName(chainId);
    updateSourceChain(chainName);
  };

  const handleDestinationChainChange = (chainId: number) => {
    const chainName = getChainName(chainId);
    updateDestinationChain(chainName);
  };

  return (
    <div className="max-w-2xl mx-auto arcade-container border-yellow-500 relative group">
      <div className="absolute inset-0 bg-yellow-500/10 blur-sm group-hover:bg-yellow-500/20 transition-all duration-300" />
      <div className="relative p-6">
        <form
          onSubmit={handleSubmit}
          className="space-y-6 relative"
          role="form"
        >
          <div className="space-y-4">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div className="relative">
                <label className="block text-yellow-500 mb-2 arcade-text">FROM</label>
                <ChainSelector
                  value={getChainId(formState.sourceChain)}
                  onChange={handleSourceChainChange}
                  label="SELECT SOURCE CHAIN"
                  disabled={formState.isSubmitting}
                  selectorType="from"
                />
              </div>

              <div className="relative">
                <label className="block text-yellow-500 mb-2 arcade-text">TO</label>
                <ChainSelector
                  value={getChainId(formState.destinationChain)}
                  onChange={handleDestinationChainChange}
                  label="SELECT DESTINATION CHAIN"
                  disabled={formState.isSubmitting}
                  selectorType="to"
                />
              </div>
            </div>

            <div className="relative">
              <label className="block text-yellow-500 mb-2 arcade-text">TOKEN</label>
              <TokenSelector
                value={formState.selectedToken}
                onChange={updateToken}
              />
            </div>

            <div className="relative">
              <div className="flex justify-between items-center mb-2">
                <label className="text-yellow-500 arcade-text">AMOUNT</label>
                <span className="text-[#00ff00] arcade-text text-xs">
                  Available: {isLoading ? 'Loading...' : isConnected && balance ? `${balance} ${symbol}` : '0.00'}
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
              <p className="mt-2 text-[#00ff00] text-[10px] arcade-text opacity-80">
                fee: {formState.tip || '0.01'} {symbol} (based on destination chain)
              </p>
            </div>
            
            {formState.error && (
              <div className="text-red-500 text-sm arcade-text border border-red-500 p-3 rounded bg-red-500/10 mt-4">
                {formState.error.message}
              </div>
            )}
          </div>
          
          {formState.success && (
            <div className="my-6">
              <div className="w-full flex justify-center">
                {fulfillmentTxHash ? (
                  <div className="w-full">
                    <CompletedAnimation />
                    <div className="flex justify-center">
                      <a 
                        href={getExplorerUrl(getChainId(formState.destinationChain), fulfillmentTxHash)}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="text-green-500 text-xs mt-2 arcade-text hover:underline"
                      >
                        TRANSFER TX: {fulfillmentTxHash.slice(0, 6)}...{fulfillmentTxHash.slice(-4)}
                      </a>
                    </div>
                  </div>
                ) : (
                  <div className="w-full">
                    <PendingAnimation />
                  </div>
                )}
              </div>
            </div>
          )}

          <button
            type="submit"
            disabled={!isConnected || (!isValid && !formState.success) || formState.isSubmitting}
            className="w-full arcade-btn bg-yellow-500 text-black hover:bg-yellow-400 transition-colors duration-200 disabled:opacity-50 disabled:cursor-not-allowed"
            onClick={formState.success ? (e) => { e.preventDefault(); resetForm(); } : undefined}
          >
            {!isConnected 
              ? 'CONNECT WALLET TO TRANSFER' 
              : formState.isSubmitting 
                ? 'APPROVING TOKENS...' 
                : formState.success 
                  ? 'START NEW TRANSFER'
                  : 'START'}
          </button>
          
          <div className="mt-3 text-center">
            <button
              type="button"
              onClick={toggleAdvanced}
              className="text-yellow-500 text-[10px] arcade-text hover:text-yellow-400 opacity-70 hover:opacity-100"
            >
              {showAdvanced ? '- HIDE ADVANCED OPTIONS' : '+ SHOW ADVANCED OPTIONS'}
            </button>
          </div>
          
          {showAdvanced && (
            <div className="space-y-4 pt-2 border-t border-gray-700 mt-4">
              <div className="relative">
                <FormInput
                  label="CUSTOM RECIPIENT"
                  labelClassName="text-yellow-500 arcade-text"
                  value={formState.recipient}
                  onChange={updateRecipient}
                  placeholder="0x..."
                  disabled={formState.isSubmitting}
                />
                <p className="text-[10px] text-gray-400 mt-1 arcade-text">Default: Your wallet address</p>
              </div>

              <div className="relative">
                <div className="flex justify-between items-center mb-2">
                  <label className="text-yellow-500 arcade-text">CUSTOM FEE ({symbol})</label>
                  <span className="text-[#00ff00] arcade-text text-xs">
                    {formState.destinationChain === 'ETHEREUM' 
                      ? `Recommended: 1.0 ${symbol}` 
                      : formState.destinationChain === 'BSC' 
                        ? `Recommended: 0.5 ${symbol}` 
                        : `Recommended: 0.2 ${symbol}`}
                  </span>
                </div>
                <FormInput
                  type="number"
                  value={formState.tip}
                  onChange={updateTip}
                  placeholder="0.1"
                  disabled={formState.isSubmitting}
                  min="0.01"
                  step="0.01"
                />
                <div className="mt-2 text-[10px] text-gray-400 arcade-text">
                  <p>Setting a lower fee may delay your transfer as speedrunners prioritize higher fees.</p>
                  <p className="mt-1">If the fee is too low, the network fees will be deducted from your transfer amount.</p>
                  <p className="mt-1">The recommended fee varies by chain: 1.0 for Ethereum, 0.5 for BSC, and 0.2 for all others.</p>
                  <p className="mt-1">
                    <a href="/about" className="text-yellow-500 hover:underline">
                      Learn more about the intent-based architecture →
                    </a>
                  </p>
                </div>
              </div>
            </div>
          )}
        </form>
      </div>
    </div>
  );
} 