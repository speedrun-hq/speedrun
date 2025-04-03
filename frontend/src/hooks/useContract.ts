import { useAccount, usePublicClient, useWalletClient } from 'wagmi';
import { getContract } from 'wagmi/actions';
import { Intent as IntentContract } from '@/contracts/Intent';
import { base, arbitrum } from 'wagmi/chains';
import { Intent } from '@/types';
import { parseEther } from 'viem';
import { useTokenApproval } from './useTokenApproval';
import { TOKENS } from '@/constants/tokens';
import { useCallback, useRef, useState } from 'react';

type ContractType = {
  write: {
    initiate: (args: [
      `0x${string}`, // asset
      bigint,        // amount
      bigint,        // targetChain
      `0x${string}`, // receiver
      bigint,        // tip
      bigint         // salt
    ]) => Promise<`0x${string}`>;
  };
};

export function useContract() {
  const { address } = useAccount();
  const publicClient = usePublicClient();
  const { data: walletClient } = useWalletClient();
  const { approveToken, isLoadingAllowance } = useTokenApproval();
  const isMounted = useRef(true);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Cleanup on unmount
  useCallback(() => {
    return () => {
      isMounted.current = false;
    };
  }, []);

  const createIntent = useCallback(async (
    sourceChain: number,
    destinationChain: number,
    token: string,
    amount: string,
    recipient: string,
    tip: string
  ): Promise<Intent> => {
    let approvalHash: `0x${string}` | undefined;
    let intentHash: `0x${string}` | undefined;

    try {
      setIsLoading(true);
      setError(null);

      if (!isMounted.current) {
        throw new Error('Component unmounted');
      }

      if (!address || !publicClient || !walletClient) {
        throw new Error('Wallet not connected');
      }

      const contractAddress = IntentContract.address[sourceChain as keyof typeof IntentContract.address];
      if (!contractAddress) {
        throw new Error(`No contract address for chain ${sourceChain}`);
      }

      // Validate chain IDs
      if (typeof sourceChain !== 'number' || typeof destinationChain !== 'number') {
        throw new Error('Invalid chain IDs');
      }

      // Calculate total amount (amount + tip)
      const totalAmount = (parseFloat(amount) + parseFloat(tip)).toString();

      // Determine token symbol
      const tokenSymbol = token === TOKENS[sourceChain].USDC.address ? 'USDC' : 'USDT';
      const chainName = sourceChain === base.id ? 'BASE' : 'ARBITRUM';

      console.log('Starting token approval process:', {
        chainName,
        tokenSymbol,
        contractAddress,
        totalAmount,
      });

      // Approve tokens for the Intent contract
      approvalHash = await approveToken(
        chainName,
        tokenSymbol,
        contractAddress,
        totalAmount
      );

      if (!isMounted.current) {
        throw new Error('Component unmounted during approval');
      }

      if (approvalHash) {
        console.log('Waiting for approval transaction:', approvalHash);
        // Wait for approval transaction to be mined
        await publicClient.waitForTransactionReceipt({ 
          hash: approvalHash,
          timeout: 30_000
        });
        console.log('Approval transaction confirmed');
      }

      if (!isMounted.current) {
        throw new Error('Component unmounted after approval');
      }

      console.log('Initializing contract:', {
        address: contractAddress,
        chainId: sourceChain,
      });

      const contract = getContract({
        address: contractAddress as `0x${string}`,
        abi: IntentContract.abi,
        walletClient,
        publicClient,
      }) as unknown as ContractType;

      if (!contract?.write?.initiate) {
        throw new Error('Contract not properly initialized');
      }

      // Convert amount and tip to wei (assuming 6 decimals for USDC)
      const amountWei = BigInt(Math.floor(parseFloat(amount) * 1e6));
      const tipWei = BigInt(Math.floor(parseFloat(tip) * 1e6));
      const salt = BigInt(Math.floor(Math.random() * Number.MAX_SAFE_INTEGER));

      // Ensure chain IDs are in the correct format
      const targetChainId = BigInt(destinationChain === base.id ? 8453 : 42161);

      console.log('Chain IDs:', {
        sourceChain,
        destinationChain,
        targetChainId: targetChainId.toString(),
      });

      const args: [
        `0x${string}`,
        bigint,
        bigint,
        `0x${string}`,
        bigint,
        bigint
      ] = [
        token as `0x${string}`,
        amountWei,
        targetChainId,
        recipient as `0x${string}`,
        tipWei,
        salt
      ];

      console.log('Creating intent with params:', {
        token,
        amountWei: amountWei.toString(),
        targetChainId: targetChainId.toString(),
        recipient,
        tipWei: tipWei.toString(),
        salt: salt.toString(),
      });

      // Call the initiate function on the contract
      const { hash } = await contract.write.initiate(args);
      intentHash = hash;

      if (!isMounted.current) {
        throw new Error('Component unmounted during intent creation');
      }

      console.log('Transaction hash:', hash);

      // Wait for the transaction to be mined
      const receipt = await publicClient.waitForTransactionReceipt({ 
        hash,
        timeout: 30_000 // 30 second timeout
      });

      if (!isMounted.current) {
        throw new Error('Component unmounted during transaction confirmation');
      }

      console.log('Transaction receipt:', receipt);

      // Find the IntentInitiated event in the receipt
      const event = receipt.logs.find((log) => {
        try {
          const { eventName } = contract.interface.parseLog(log);
          return eventName === 'IntentInitiated';
        } catch {
          return false;
        }
      });

      if (!event) {
        throw new Error('Failed to find IntentInitiated event');
      }

      // Decode the event data
      const { args: eventArgs } = contract.interface.parseLog(event);

      if (!isMounted.current) {
        throw new Error('Component unmounted during event processing');
      }

      return {
        id: eventArgs.intentId,
        source_chain: sourceChain === base.id ? 'BASE' : 'ARBITRUM',
        destination_chain: destinationChain === base.id ? 'BASE' : 'ARBITRUM',
        token,
        amount,
        recipient,
        intent_fee: tip,
        status: 'pending',
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      };
    } catch (error) {
      console.error('Error creating intent:', error);
      
      // Log transaction hashes for debugging
      if (approvalHash) {
        console.error('Approval transaction hash:', approvalHash);
      }
      if (intentHash) {
        console.error('Intent transaction hash:', intentHash);
      }

      if (error instanceof Error) {
        setError(error.message);
        throw new Error(`Failed to create intent: ${error.message}`);
      }
      throw error;
    } finally {
      if (isMounted.current) {
        setIsLoading(false);
      }
    }
  }, [address, publicClient, walletClient, approveToken]);

  return {
    createIntent,
    isConnected: !!address && !!walletClient,
    isLoading,
    error,
  };
} 