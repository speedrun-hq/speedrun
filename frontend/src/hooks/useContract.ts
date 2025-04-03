import { useAccount, usePublicClient, useWalletClient } from 'wagmi';
import { getContract } from 'wagmi/actions';
import { Intent as IntentContract } from '@/contracts/Intent';
import { base, arbitrum } from 'wagmi/chains';
import { Intent } from '@/types';
import { parseEther } from 'viem';
import { useTokenApproval } from './useTokenApproval';
import { TOKENS } from '@/constants/tokens';
import { useCallback, useRef, useState } from 'react';
import { Abi } from 'viem';
import { keccak256, toUtf8Bytes } from 'ethers';

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
  createEventFilter: {
    IntentInitiated: (
      args?: {
        intentId?: `0x${string}`;
        asset?: `0x${string}`;
      }
    ) => Promise<any>;
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

      // Enhanced wallet connection checks
      if (!address) {
        throw new Error('Wallet address not available');
      }
      if (!publicClient) {
        throw new Error('Public client not initialized');
      }
      if (!walletClient) {
        throw new Error('Wallet client not initialized');
      }

      // Wait for wallet connection to be fully established
      await new Promise(resolve => setTimeout(resolve, 1000));

      // Verify wallet is still connected after delay
      if (!address || !publicClient || !walletClient) {
        throw new Error('Wallet connection lost');
      }

      // Validate numeric inputs
      if (typeof sourceChain !== 'number' || isNaN(sourceChain)) {
        throw new Error('Invalid source chain ID');
      }
      if (typeof destinationChain !== 'number' || isNaN(destinationChain)) {
        throw new Error('Invalid destination chain ID');
      }

      const contractAddress = IntentContract.address[sourceChain as keyof typeof IntentContract.address];
      if (!contractAddress) {
        throw new Error(`No contract address for chain ${sourceChain}`);
      }

      // Validate amount and tip
      const amountNum = parseFloat(amount);
      const tipNum = parseFloat(tip);
      
      if (isNaN(amountNum) || amountNum <= 0) {
        throw new Error('Invalid amount');
      }
      if (isNaN(tipNum) || tipNum < 0) {
        throw new Error('Invalid tip amount');
      }

      // Calculate total amount (amount + tip)
      const totalAmount = (amountNum + tipNum).toString();

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
      }) as unknown as ContractType;

      if (!contract?.write?.initiate) {
        throw new Error('Contract not properly initialized');
      }

      // Convert amount and tip to wei (assuming 6 decimals for USDC)
      const amountWei = BigInt(Math.floor(amountNum * 1e6));
      const tipWei = BigInt(Math.floor(tipNum * 1e6));
      const salt = BigInt(Math.floor(Math.random() * Number.MAX_SAFE_INTEGER));

      // Ensure chain IDs are in the correct format
      const targetChainId = BigInt(destinationChain === base.id ? 8453 : 42161);

      console.log('Chain IDs:', {
        sourceChain,
        destinationChain,
        targetChainId: targetChainId.toString(),
      });

      // Validate all BigInt values before creating args
      if (amountWei <= BigInt(0)) {
        throw new Error('Amount must be greater than 0');
      }
      if (targetChainId <= BigInt(0)) {
        throw new Error('Invalid target chain ID');
      }

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
      const hash = await contract.write.initiate(args);
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
        // The first topic is the event signature
        const eventSignature = 'IntentInitiated(bytes32,address,uint256,uint256,bytes,uint256,uint256)';
        const eventSignatureHash = keccak256(toUtf8Bytes(eventSignature));
        return log.topics[0] === eventSignatureHash;
      });

      if (!event || !event.topics[1] || !event.topics[2]) {
        throw new Error('Failed to find IntentInitiated event or missing required topics');
      }

      // Parse the event data
      const intentId = event.topics[1] as `0x${string}`; // First indexed parameter (intentId)
      const asset = `0x${event.topics[2].slice(26)}` as `0x${string}`; // Second indexed parameter (asset), convert to address format

      if (!isMounted.current) {
        throw new Error('Component unmounted during event processing');
      }

      return {
        id: intentId,
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