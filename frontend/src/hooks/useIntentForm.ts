import { useCallback, useState } from 'react';
import { useAccount, useBalance, useNetwork } from 'wagmi';
import { useTokenBalance } from '@/hooks/useTokenBalance';
import { apiService } from '@/services/api';
import { useContract } from './useContract';
import { TOKENS } from '@/constants/tokens';
import { getChainId, getChainName, ChainName } from '@/utils/chain';

type TokenSymbol = 'USDC' | 'USDT';

interface FormState {
  sourceChain: ChainName;
  destinationChain: ChainName;
  selectedToken: TokenSymbol;
  amount: string;
  recipient: string;
  tip: string;
  isSubmitting: boolean;
  error: Error | null;
  success: boolean;
}

export function useIntentForm() {
  const { address, isConnected } = useAccount();
  const { chain } = useNetwork();
  const { createIntent, isConnected: isWalletConnected } = useContract();

  const [formState, setFormState] = useState<FormState>({
    sourceChain: 'BASE',
    destinationChain: 'ARBITRUM',
    selectedToken: 'USDC',
    amount: '',
    recipient: '',
    tip: '0.1',
    isSubmitting: false,
    error: null,
    success: false,
  });

  // Get token balance
  const { balance, symbol, isLoading, isError } = useTokenBalance(
    formState.sourceChain,
    formState.selectedToken
  );

  // Form validation
  const isValid = Boolean(
    formState.amount &&
    formState.recipient &&
    formState.tip &&
    parseFloat(formState.amount) > 0 &&
    parseFloat(formState.tip) > 0 &&
    parseFloat(formState.amount) <= (balance ? parseFloat(balance) : 0)
  );

  // Handle form submission
  const handleSubmit = useCallback(async (e?: React.FormEvent) => {
    if (e) {
      e.preventDefault();
    }
    if (!isValid) return;

    try {
      setFormState(prev => ({ ...prev, isSubmitting: true, error: null, success: false }));

      const sourceChainId = getChainId(formState.sourceChain);
      const destinationChainId = getChainId(formState.destinationChain);
      const tokenAddress = TOKENS[sourceChainId][formState.selectedToken].address;

      await createIntent(
        sourceChainId,
        destinationChainId,
        tokenAddress,
        formState.amount,
        formState.recipient,
        formState.tip
      );

      setFormState(prev => ({
        ...prev,
        success: true,
        amount: '',
        recipient: '',
        tip: '0.1',
      }));
    } catch (err) {
      setFormState(prev => ({
        ...prev,
        error: err instanceof Error ? err : new Error('Failed to create intent'),
      }));
    } finally {
      setFormState(prev => ({ ...prev, isSubmitting: false }));
    }
  }, [formState, isValid, createIntent]);

  // Update handlers
  const updateSourceChain = useCallback((chainName: ChainName) => {
    setFormState(prev => {
      // If source and destination would be the same, set destination to a different chain
      let newDestination = prev.destinationChain;
      if (chainName === prev.destinationChain) {
        // Choose a different chain for destination
        const options: ChainName[] = ['BASE', 'ARBITRUM', 'ETHEREUM', 'BSC', 'POLYGON', 'AVALANCHE'];
        newDestination = options.find(c => c !== chainName) || 'BASE';
      }
      
      return {
        ...prev,
        sourceChain: chainName,
        destinationChain: newDestination,
      };
    });
  }, []);

  const updateDestinationChain = useCallback((chainName: ChainName) => {
    setFormState(prev => {
      // If source and destination would be the same, set source to a different chain
      let newSource = prev.sourceChain;
      if (chainName === prev.sourceChain) {
        // Choose a different chain for source
        const options: ChainName[] = ['BASE', 'ARBITRUM', 'ETHEREUM', 'BSC', 'POLYGON', 'AVALANCHE'];
        newSource = options.find(c => c !== chainName) || 'ARBITRUM';
      }
      
      return {
        ...prev,
        destinationChain: chainName,
        sourceChain: newSource,
      };
    });
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

  const updateTip = useCallback((value: string) => {
    setFormState(prev => ({
      ...prev,
      tip: value,
    }));
  }, []);

  return {
    formState,
    balance,
    isError,
    isLoading,
    symbol,
    isConnected: isConnected && isWalletConnected,
    isValid,
    handleSubmit,
    updateSourceChain,
    updateDestinationChain,
    updateToken,
    updateAmount,
    updateRecipient,
    updateTip,
  };
} 