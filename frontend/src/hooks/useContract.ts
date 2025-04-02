import { useAccount, usePublicClient, useWalletClient } from 'wagmi';
import { getContract } from 'wagmi/actions';
import { Intent as IntentContract } from '@/contracts/Intent';
import { base, arbitrum } from 'wagmi/chains';
import { Intent } from '@/types';
import { parseEther } from 'viem';

type ContractType = {
  write: {
    initiate: (...args: any[]) => Promise<`0x${string}`>;
  };
};

export function useContract() {
  const { address } = useAccount();
  const publicClient = usePublicClient();
  const { data: walletClient } = useWalletClient();

  const createIntent = async (
    sourceChain: number,
    destinationChain: number,
    token: string,
    amount: string,
    recipient: string,
    tip: string
  ): Promise<Intent> => {
    try {
      if (!address || !publicClient || !walletClient) {
        throw new Error('Wallet not connected');
      }

      const contractAddress = IntentContract.address[sourceChain as keyof typeof IntentContract.address];
      if (!contractAddress) {
        throw new Error(`No contract address for chain ${sourceChain}`);
      }

      const contract = getContract({
        address: contractAddress as `0x${string}`,
        abi: IntentContract.abi,
        walletClient,
        publicClient,
      });

      if (!contract || typeof contract.write?.initiate !== 'function') {
        throw new Error('Contract not properly initialized');
      }

      // Convert amount and tip to wei (assuming 6 decimals for USDC)
      const amountWei = BigInt(Math.floor(parseFloat(amount) * 1e6));
      const tipWei = BigInt(Math.floor(parseFloat(tip) * 1e6));
      const salt = BigInt(Math.floor(Math.random() * Number.MAX_SAFE_INTEGER));

      // Ensure chain IDs are in the correct format
      const targetChainId = destinationChain === base.id ? 8453n : 42161n;

      console.log('Creating intent with params:', {
        token,
        amountWei: amountWei.toString(),
        targetChainId: targetChainId.toString(),
        recipient,
        tipWei: tipWei.toString(),
        salt: salt.toString(),
      });

      // Call the initiate function on the contract
      const { hash } = await contract.write.initiate(
        token as `0x${string}`,
        amountWei,
        targetChainId,
        recipient as `0x${string}`,
        tipWei,
        salt
      );

      console.log('Transaction hash:', hash);

      // Wait for the transaction to be mined
      const receipt = await publicClient.waitForTransactionReceipt({ 
        hash,
        timeout: 30_000 // 30 second timeout
      });

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
      const { args } = contract.interface.parseLog(event);

      return {
        id: args.intentId,
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
      throw error;
    }
  };

  return {
    createIntent,
    isConnected: !!address && !!walletClient,
  };
} 