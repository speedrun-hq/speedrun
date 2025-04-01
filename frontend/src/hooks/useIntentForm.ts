import { useState, useEffect, useCallback, useMemo } from 'react';
import { useAccount, useNetwork } from 'wagmi';
import { useTokenBalance } from './useTokenBalance';
import { TOKENS } from '../config/tokens';
import { base, arbitrum } from 'wagmi/chains';
import { apiService } from '@/services/api';
import { ApiError } from '@/utils/errors';

type TokenSymbol = 'USDC' | 'USDT';
type ChainName = 'BASE' | 'ARBITRUM';

interface IntentFormState {
  sourceChain: ChainName;
  destinationChain: ChainName;
  selectedToken: TokenSymbol;
  amount: string;
  recipient: string;
  isSubmitting: boolean;
  error: Error | null;
  success: boolean;
}

export function useIntentForm() {
  const { address, isConnected } = useAccount();
  const { chain } = useNetwork();
  
  const [formState, setFormState] = useState<IntentFormState>({
    sourceChain: 'BASE',
    destinationChain: 'ARBITRUM',
    selectedToken: 'USDC',
    amount: '',
    recipient: '',
    isSubmitting: false,
    error: null,
    success: false,
  });

  // Helper function to convert chain ID to name
  const getChainName = (chainId: number): ChainName => {
    return chainId === base.id ? 'BASE' : 'ARBITRUM';
  };

  // Helper function to convert chain name to ID
  const getChainId = (chainName: ChainName): number => {
    return chainName === 'BASE' ? base.id : arbitrum.id;
  };

  // Fetch token balance
  const { balance, isError, isLoading: isBalanceLoading, symbol } = useTokenBalance(
    formState.selectedToken,
    address,
    getChainId(formState.sourceChain)
  );

  // Update source chain when wallet chain changes
  useEffect(() => {
    if (chain?.id && TOKENS[chain.id]) {
      const sourceName = getChainName(chain.id);
      const destName = sourceName === 'BASE' ? 'ARBITRUM' : 'BASE';
      setFormState(prev => ({
        ...prev,
        sourceChain: sourceName,
        destinationChain: destName,
      }));
    }
  }, [chain?.id]);

  // Form validation
  const isValid = useMemo(() => {
    if (!formState.recipient || !formState.amount || isError || isBalanceLoading) {
      return false;
    }

    // Validate recipient address format
    if (!/^0x[a-fA-F0-9]{40}$/.test(formState.recipient)) {
      return false;
    }

    // Validate amount is not greater than balance
    if (balance && parseFloat(formState.amount) > parseFloat(balance)) {
      return false;
    }

    return true;
  }, [formState.recipient, formState.amount, balance, isError, isBalanceLoading]);

  // Handle form submission
  const handleSubmit = useCallback(async (e?: React.FormEvent) => {
    if (e) {
      e.preventDefault();
    }
    if (!isValid) return;

    try {
      setFormState(prev => ({ ...prev, isSubmitting: true, error: null, success: false }));

      await apiService.createIntent({
        source_chain: formState.sourceChain.toLowerCase(),
        destination_chain: formState.destinationChain.toLowerCase(),
        token: formState.selectedToken,
        amount: formState.amount,
        recipient: formState.recipient,
        intent_fee: '0.01',
      });

      setFormState(prev => ({
        ...prev,
        success: true,
        amount: '',
        recipient: '',
      }));
    } catch (err) {
      setFormState(prev => ({
        ...prev,
        error: err instanceof Error ? err : new Error('Failed to create intent'),
      }));
    } finally {
      setFormState(prev => ({ ...prev, isSubmitting: false }));
    }
  }, [formState, isValid]);

  // Update handlers
  const updateSourceChain = useCallback((value: ChainName) => {
    setFormState(prev => ({
      ...prev,
      sourceChain: value,
      destinationChain: value === 'BASE' ? 'ARBITRUM' : 'BASE',
    }));
  }, []);

  const updateDestinationChain = useCallback((value: ChainName) => {
    setFormState(prev => ({
      ...prev,
      destinationChain: value,
      sourceChain: value === 'BASE' ? 'ARBITRUM' : 'BASE',
    }));
  }, []);

  const updateToken = useCallback((value: TokenSymbol) => {
    setFormState(prev => ({ ...prev, selectedToken: value }));
  }, []);

  const updateAmount = useCallback((value: string) => {
    setFormState(prev => ({ ...prev, amount: value }));
  }, []);

  const updateRecipient = useCallback((value: string) => {
    setFormState(prev => ({ ...prev, recipient: value }));
  }, []);

  return {
    formState,
    balance,
    isError,
    isLoading: isBalanceLoading,
    symbol,
    isConnected,
    isValid,
    handleSubmit,
    updateSourceChain,
    updateDestinationChain,
    updateToken,
    updateAmount,
    updateRecipient,
  };
} 