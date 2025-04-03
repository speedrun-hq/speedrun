import { useCallback, useState } from 'react';
import { useAccount, useBalance, useNetwork } from 'wagmi';
import { useTokenBalance } from '@/hooks/useTokenBalance';
import { apiService } from '@/services/api';
import { useContract } from './useContract';
import { TOKENS } from '@/constants/tokens';
import { getChainId } from '@/utils/chain';

type ChainName = 'BASE' | 'ARBITRUM';
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
    tip: '0.01',
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

      const intent = await createIntent(
        sourceChainId,
        destinationChainId,
        tokenAddress,
        formState.amount,
        formState.recipient,
        formState.tip
      );

      // Create intent in the backend
      await apiService.createIntent({
        source_chain: formState.sourceChain,
        destination_chain: formState.destinationChain,
        token: formState.selectedToken,
        amount: formState.amount,
        recipient: formState.recipient,
        intent_fee: formState.tip,
      });

      setFormState(prev => ({
        ...prev,
        success: true,
        amount: '',
        recipient: '',
        tip: '0.01',
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