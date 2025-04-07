import { useCallback, useState, useEffect, useRef } from 'react';
import { useAccount, useBalance, useNetwork } from 'wagmi';
import { useTokenBalance } from '@/hooks/useTokenBalance';
import { apiService } from '@/services/api';
import { useContract } from './useContract';
import { TOKENS, TokenSymbol } from '@/constants/tokens';
import { getChainId, getChainName, ChainName } from '@/utils/chain';
import { Intent, Fulfillment } from '@/types';

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
  intentId: string | null;
  fulfillmentTxHash: string | null;
  approvalHash: string | null;
  intentHash: string | null;
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
    tip: '0.2',
    isSubmitting: false,
    error: null,
    success: false,
    intentId: null,
    fulfillmentTxHash: null,
    approvalHash: null,
    intentHash: null,
  });
  
  // Private state for intent status tracking
  const [intentStatus, setIntentStatus] = useState<'pending' | 'fulfilled'>('pending');
  const pollingInterval = useRef<NodeJS.Timeout | null>(null);

  // Start polling when we have an intentId and success is true
  useEffect(() => {
    if (formState.success && formState.intentId) {
      // Start polling for intent status
      startPolling(formState.intentId);
      
      // Cleanup function to stop polling when unmounted
      return () => {
        stopPolling();
      };
    }
  }, [formState.success, formState.intentId]);

  // Query fulfillment when intent status changes to fulfilled
  useEffect(() => {
    const checkFulfillment = async () => {
      if (intentStatus === 'fulfilled' && formState.intentId) {
        try {
          // Get the intent to check its status
          const intent = await apiService.getIntent(formState.intentId);
          
          // Once we know the intent is fulfilled or settled
          if (intent && (intent.status === 'fulfilled' || intent.status === 'settled')) {
            try {
              // Get the fulfillment data using the API service
              const fulfillment = await apiService.getFulfillment(formState.intentId);
              
              if (fulfillment && fulfillment.tx_hash) {
                setFormState(prev => ({
                  ...prev,
                  fulfillmentTxHash: fulfillment.tx_hash
                }));
              } else {
                // Set empty string if no tx_hash is available
                setFormState(prev => ({
                  ...prev,
                  fulfillmentTxHash: ''
                }));
              }
            } catch (fulfillmentError) {
              console.error('Error accessing fulfillment data:', fulfillmentError);
              // Set empty string if error fetching fulfillment
              setFormState(prev => ({
                ...prev,
                fulfillmentTxHash: ''
              }));
            }
          }
        } catch (error) {
          console.error('Error fetching intent:', error);
        }
      }
    };
    
    checkFulfillment();
  }, [intentStatus, formState.intentId]);

  // Function to start polling the intent status
  const startPolling = useCallback((intentId: string) => {
    // Clear any existing interval
    stopPolling();
    
    // Set initial status to pending
    setIntentStatus('pending');
    
    // Start a new polling interval
    pollingInterval.current = setInterval(async () => {
      try {
        // Fetch the latest intent data
        const intent = await apiService.getIntent(intentId);
        
        // Update the status based on the intent's status
        if (intent.status === 'fulfilled' || intent.status === 'settled') {
          setIntentStatus('fulfilled');
          // Stop polling once we've reached a terminal state
          stopPolling();
        } else {
          setIntentStatus('pending');
        }
      } catch (error) {
        console.error('Error polling intent status:', error);
        // Don't stop polling on error, just try again next interval
      }
    }, 1500); // Poll every 1.5 seconds
  }, []);

  // Function to stop polling
  const stopPolling = useCallback(() => {
    if (pollingInterval.current) {
      clearInterval(pollingInterval.current);
      pollingInterval.current = null;
    }
  }, []);

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

  // Get recommended fee based on destination chain
  const getRecommendedFee = useCallback((chainName: ChainName): string => {
    switch (chainName) {
      case 'ETHEREUM':
        return '1.0';
      case 'BSC':
        return '0.5';
      default:
        return '0.2';
    }
  }, []);

  // Handle form submission
  const handleSubmit = useCallback(async (e?: React.FormEvent) => {
    if (e) {
      e.preventDefault();
    }
    if (!isValid) return;

    try {
      setFormState(prev => ({ 
        ...prev, 
        isSubmitting: true, 
        error: null, 
        success: false,
        intentId: null,
        fulfillmentTxHash: null,
        approvalHash: null,
        intentHash: null
      }));
      
      setIntentStatus('pending');

      const sourceChainId = getChainId(formState.sourceChain);
      const destinationChainId = getChainId(formState.destinationChain);
      const tokenAddress = TOKENS[sourceChainId][formState.selectedToken].address;

      // First, set the form state to indicate we're approving USDC
      setFormState(prev => ({
        ...prev,
        isSubmitting: true,
        approvalHash: null
      }));

      // First step: Get approval
      const approvalResult = await createIntent(
        sourceChainId,
        destinationChainId,
        tokenAddress,
        formState.amount,
        formState.recipient,
        formState.tip
      );
      
      // If we got an approval hash but no intent hash, we're in the approval phase
      if (approvalResult.approvalHash && !approvalResult.intentHash) {
        setFormState(prev => ({
          ...prev,
          approvalHash: approvalResult.approvalHash,
          isSubmitting: true // Keep submitting true as we're moving to intent creation
        }));

        // Second step: Create the intent
        const intentResult = await createIntent(
          sourceChainId,
          destinationChainId,
          tokenAddress,
          formState.amount,
          formState.recipient,
          formState.tip
        );

        // Update form state with the final success status
        setFormState(prev => {
          // Get the recommended fee for the current destination chain
          const recommendedFee = getRecommendedFee(prev.destinationChain);
          
          return {
            ...prev,
            success: true,
            intentId: intentResult.id,
            approvalHash: approvalResult.approvalHash,
            intentHash: intentResult.intentHash,
            amount: '',
            recipient: '',
            tip: recommendedFee,
          };
        });

        // If we have an intentId, start polling for status updates
        if (intentResult.id) {
          startPolling(intentResult.id);
        }
      }
    } catch (err) {
      setFormState(prev => ({
        ...prev,
        error: err instanceof Error ? err : new Error('Failed to create intent'),
        intentId: null,
        fulfillmentTxHash: null,
        approvalHash: null,
        intentHash: null
      }));
      
      setIntentStatus('pending');
    } finally {
      setFormState(prev => ({ ...prev, isSubmitting: false }));
    }
  }, [formState, isValid, createIntent, startPolling, getRecommendedFee]);

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
      
      // Set recommended fee based on destination chain
      const recommendedFee = getRecommendedFee(chainName);
      
      return {
        ...prev,
        destinationChain: chainName,
        sourceChain: newSource,
        tip: recommendedFee,
      };
    });
  }, [getRecommendedFee]);

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

  // Reset the form and stop polling
  const resetForm = useCallback(() => {
    stopPolling();
    setIntentStatus('pending');
    setFormState(prev => {
      // Get the recommended fee for the current destination chain
      const recommendedFee = getRecommendedFee(prev.destinationChain);
      
      return {
        ...prev,
        success: false,
        intentId: null,
        fulfillmentTxHash: null,
        error: null,
        amount: '',
        // Keep the current destination chain but update the fee accordingly
        tip: recommendedFee
      };
    });
  }, [stopPolling, getRecommendedFee]);

  return {
    formState,
    balance,
    isError,
    isLoading,
    symbol,
    isConnected: isConnected && isWalletConnected,
    isValid,
    fulfillmentTxHash: formState.fulfillmentTxHash || '',
    handleSubmit,
    updateSourceChain,
    updateDestinationChain,
    updateToken,
    updateAmount,
    updateRecipient,
    updateTip,
    resetForm
  };
} 