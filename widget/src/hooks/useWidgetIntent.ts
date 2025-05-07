import { useState, useCallback, useEffect, useRef } from "react";
import { useAccount } from "wagmi";
import { ChainName, TokenSymbol, getChainId, TOKENS } from "@speedrun/utils";

// Simplified API client for Speedrun services
import { apiClient } from "../services/apiClient";

interface UseWidgetIntentConfig {
  defaultSourceChain?: ChainName;
  defaultDestinationChain?: ChainName;
  defaultToken?: TokenSymbol;
  defaultAmount?: string;
}

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
}

export function useWidgetIntent(config: UseWidgetIntentConfig = {}) {
  const { address, isConnected } = useAccount();
  
  const [formState, setFormState] = useState<FormState>({
    sourceChain: config.defaultSourceChain || "BASE",
    destinationChain: config.defaultDestinationChain || "ARBITRUM",
    selectedToken: config.defaultToken || "USDC",
    amount: config.defaultAmount || "",
    recipient: "",
    tip: "0.2",
    isSubmitting: false,
    error: null,
    success: false,
    intentId: null,
    fulfillmentTxHash: null,
  });

  // Status tracking
  const [intentStatus, setIntentStatus] = useState<"pending" | "fulfilled">("pending");
  const pollingInterval = useRef<NodeJS.Timeout | null>(null);

  // Balance simulation (simplified for the widget)
  const [balance, setBalance] = useState("0");
  const [isLoading, setIsLoading] = useState(false);
  
  // Clean up polling on unmount
  useEffect(() => {
    return () => {
      if (pollingInterval.current) {
        clearInterval(pollingInterval.current);
      }
    };
  }, []);

  // Function to stop polling
  const stopPolling = useCallback(() => {
    if (pollingInterval.current) {
      clearInterval(pollingInterval.current);
      pollingInterval.current = null;
    }
  }, []);

  // Function to start polling for intent status
  const startPolling = useCallback((intentId: string) => {
    stopPolling();
    setIntentStatus("pending");

    pollingInterval.current = setInterval(async () => {
      try {
        const intent = await apiClient.getIntent(intentId);
        
        if (intent.status === "fulfilled" || intent.status === "settled") {
          setIntentStatus("fulfilled");
          stopPolling();
          
          // Get fulfillment details if available
          try {
            const fulfillment = await apiClient.getFulfillment(intentId);
            if (fulfillment && fulfillment.tx_hash) {
              setFormState(prev => ({
                ...prev,
                fulfillmentTxHash: fulfillment.tx_hash
              }));
            }
          } catch (err) {
            console.error("Error fetching fulfillment:", err);
          }
        }
      } catch (err) {
        console.error("Error polling intent:", err);
      }
    }, 1500);
  }, [stopPolling]);

  // Mock token balance fetch - in real implementation, this would use wagmi hooks
  useEffect(() => {
    if (isConnected && address) {
      setIsLoading(true);
      
      // Simulate API call
      setTimeout(() => {
        setBalance("100"); // Mock balance
        setIsLoading(false);
      }, 500);
    }
  }, [isConnected, address, formState.sourceChain, formState.selectedToken]);

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
      case "ETHEREUM":
        return "1.0";
      case "BSC":
        return "0.5";
      default:
        return "0.2";
    }
  }, []);

  // Handle form submission
  const handleSubmit = useCallback(async (e?: React.FormEvent) => {
    if (e) {
      e.preventDefault();
    }
    
    if (!isValid || !isConnected || !address) return;

    try {
      setFormState(prev => ({
        ...prev,
        isSubmitting: true,
        error: null,
        success: false,
        intentId: null,
        fulfillmentTxHash: null,
      }));

      setIntentStatus("pending");

      const sourceChainId = getChainId(formState.sourceChain);
      const destinationChainId = getChainId(formState.destinationChain);
      const tokenAddress = TOKENS[sourceChainId][formState.selectedToken].address;

      // Simulate intent creation
      // This would normally interact with Speedrun's API and blockchain
      const intentResult = await apiClient.createIntent({
        sourceChainId,
        destinationChainId,
        tokenAddress,
        amount: formState.amount,
        recipient: formState.recipient,
        tip: formState.tip,
        sender: address,
      });

      // Update form state with success
      setFormState(prev => {
        const recommendedFee = getRecommendedFee(prev.destinationChain);
        
        return {
          ...prev,
          success: true,
          intentId: intentResult.id,
          amount: "",
          recipient: "",
          tip: recommendedFee,
          isSubmitting: false,
        };
      });

      // Start polling for status updates
      if (intentResult.id) {
        startPolling(intentResult.id);
      }
    } catch (err) {
      setFormState(prev => ({
        ...prev,
        error: err instanceof Error ? err : new Error("Failed to create intent"),
        isSubmitting: false,
      }));
      setIntentStatus("pending");
    }
  }, [formState, isValid, isConnected, address, getRecommendedFee, startPolling]);

  // Update handlers
  const updateSourceChain = useCallback((chainName: ChainName) => {
    setFormState(prev => {
      // If source and destination would be the same, set destination to a different chain
      let newDestination = prev.destinationChain;
      if (chainName === prev.destinationChain) {
        const options: ChainName[] = [
          "BASE", "ARBITRUM", "ETHEREUM", "BSC", "POLYGON", "AVALANCHE"
        ];
        newDestination = options.find(c => c !== chainName) || "BASE";
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
        const options: ChainName[] = [
          "BASE", "ARBITRUM", "ETHEREUM", "BSC", "POLYGON", "AVALANCHE"
        ];
        newSource = options.find(c => c !== chainName) || "ARBITRUM";
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
    setFormState(prev => ({ ...prev, tip: value }));
  }, []);

  // Reset the form and stop polling
  const resetForm = useCallback(() => {
    stopPolling();
    setIntentStatus("pending");
    setFormState(prev => {
      const recommendedFee = getRecommendedFee(prev.destinationChain);
      return {
        ...prev,
        success: false,
        intentId: null,
        fulfillmentTxHash: null,
        error: null,
        amount: "",
        tip: recommendedFee,
      };
    });
  }, [stopPolling, getRecommendedFee]);

  return {
    formState,
    balance,
    isError: false,
    isLoading,
    symbol: formState.selectedToken,
    isConnected: isConnected,
    isValid,
    fulfillmentTxHash: formState.fulfillmentTxHash || "",
    handleSubmit,
    updateSourceChain,
    updateDestinationChain,
    updateToken,
    updateAmount,
    updateRecipient,
    updateTip,
    resetForm,
  };
} 