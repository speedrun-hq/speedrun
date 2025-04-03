import { Intent } from '@/types';
import { getContract } from 'wagmi/actions';
import { Intent as IntentContract } from '@/contracts/Intent';
import { base, arbitrum } from 'wagmi/chains';
import { Abi } from 'viem';
import { GetContractReturnType, PublicClient, WalletClient } from 'viem';
import { useAccount, usePublicClient, useWalletClient } from 'wagmi';

type ContractType = {
  write: {
    initiate: (...args: any[]) => Promise<`0x${string}`>;
  };
  interface: {
    getEventTopic: (eventName: string) => `0x${string}`;
    decodeEventLog: (eventName: string, data: `0x${string}`, topics: `0x${string}`[]) => any;
  };
};

class ContractService {
  async createIntent(
    sourceChain: number,
    destinationChain: number,
    token: string,
    amount: string,
    recipient: string,
    tip: string
  ): Promise<Intent> {
    try {
      const { address } = useAccount();
      const publicClient = usePublicClient();
      const { data: walletClient } = useWalletClient();

      if (!address || !publicClient || !walletClient) {
        throw new Error('Wallet not connected');
      }

      const contract = getContract({
        address: IntentContract.address[sourceChain as keyof typeof IntentContract.address] as `0x${string}`,
        abi: IntentContract.abi,
      }) as unknown as ContractType;

      if (!contract || !contract.write || !contract.write.initiate) {
        throw new Error('Contract not properly initialized');
      }

      // Convert amount and tip to wei
      const amountWei = BigInt(parseFloat(amount) * 1e18);
      const tipWei = BigInt(parseFloat(tip) * 1e18);

      // Generate a random salt for the intent
      const salt = BigInt(Math.floor(Math.random() * Number.MAX_SAFE_INTEGER));

      // Call the initiate function on the contract
      const hash = await contract.write.initiate([
        token as `0x${string}`, // asset
        amountWei, // amount
        BigInt(destinationChain), // targetChain
        recipient as `0x${string}`, // receiver
        tipWei, // tip
        salt, // salt
      ]);

      // Wait for the transaction to be mined
      const receipt = await publicClient.waitForTransactionReceipt({ hash });

      // Find the IntentInitiated event in the receipt
      const event = receipt.logs.find(
        (log) => log.topics[0] === contract.interface.getEventTopic('IntentInitiated')
      );

      if (!event) {
        throw new Error('Failed to find IntentInitiated event');
      }

      // Decode the event data
      const decoded = contract.interface.decodeEventLog(
        'IntentInitiated',
        event.data,
        event.topics
      );

      return {
        id: decoded.intentId,
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
  }
}

export const contractService = new ContractService(); 