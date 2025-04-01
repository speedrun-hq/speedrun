'use client';

import { ChainSelector } from './ChainSelector';
import { TokenSelector } from './TokenSelector';
import { FormInput } from './FormInput';
import ErrorMessage from '@/components/ErrorMessage';
import { useIntentForm } from '@/hooks/useIntentForm';
import { base, arbitrum } from 'wagmi/chains';

// Helper function to convert chain name to ID
const getChainId = (chainName: 'BASE' | 'ARBITRUM'): number => {
  return chainName === 'BASE' ? base.id : arbitrum.id;
};

export default function CreateNewIntent() {
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
  } = useIntentForm();

  if (!isConnected) {
    return (
      <div className="max-w-2xl mx-auto p-6 bg-black border-2 border-[hsl(var(--yellow))] rounded-lg shadow-lg">
        <h2 className="text-2xl font-bold text-[hsl(var(--yellow))] mb-6 text-center font-mono">
          CREATE NEW RUN
        </h2>
        <p className="text-[hsl(var(--yellow))] text-center font-mono">
          Please connect your wallet to continue
        </p>
      </div>
    );
  }

  return (
    <div className="max-w-2xl mx-auto p-6 bg-black border-2 border-[hsl(var(--yellow))] rounded-lg shadow-lg">
      <h2 className="text-2xl font-bold text-[hsl(var(--yellow))] mb-6 text-center font-mono">
        CREATE NEW RUN
      </h2>
      
      <form
        onSubmit={handleSubmit}
        className="space-y-6"
        role="form"
      >
        {formState.error && <ErrorMessage error={formState.error} className="mb-4" />}
        
        {formState.success && (
          <div className="bg-green-500/10 border border-green-500 text-green-500 p-4 rounded-lg mb-4">
            <p className="font-mono text-center">RUN CREATED SUCCESSFULLY!</p>
          </div>
        )}

        <div className="space-y-4">
          <div>
            <label className="block text-[hsl(var(--yellow))] mb-2 font-mono">SOURCE CHAIN</label>
            <ChainSelector
              value={getChainId(formState.sourceChain)}
              onChange={(value) => updateSourceChain(value === base.id ? 'BASE' : 'ARBITRUM')}
              label="SELECT SOURCE CHAIN"
              disabled={formState.isSubmitting}
            />
          </div>

          <div>
            <label className="block text-[hsl(var(--yellow))] mb-2 font-mono">DESTINATION CHAIN</label>
            <ChainSelector
              value={getChainId(formState.destinationChain)}
              onChange={(value) => updateDestinationChain(value === base.id ? 'BASE' : 'ARBITRUM')}
              label="SELECT DESTINATION CHAIN"
              disabled={formState.isSubmitting}
            />
          </div>

          <div>
            <label className="block text-[hsl(var(--yellow))] mb-2 font-mono">SELECT TOKEN</label>
            <TokenSelector
              value={formState.selectedToken}
              onChange={updateToken}
            />
          </div>

          <FormInput
            label="RECIPIENT ADDRESS"
            value={formState.recipient}
            onChange={updateRecipient}
            placeholder="0x..."
            disabled={formState.isSubmitting}
          />

          <div>
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
          </div>
        </div>

        <button
          type="submit"
          disabled={!isValid || formState.isSubmitting}
          className="w-full arcade-btn bg-[hsl(var(--yellow))] text-black hover:bg-[hsl(var(--yellow)/0.8)] transition-colors duration-200 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {formState.isSubmitting ? 'CREATING RUN...' : 'START RUN'}
        </button>
      </form>
    </div>
  );
} 