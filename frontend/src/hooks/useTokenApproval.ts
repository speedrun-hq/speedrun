import { useAccount, useNetwork, useWalletClient, useContractWrite, useContractRead, usePublicClient } from 'wagmi';
import { parseUnits, getContract } from 'viem';
import { TOKENS } from '@/constants/tokens';
import { base, arbitrum, mainnet, bsc, polygon, avalanche } from 'wagmi/chains';
import { useState, useCallback } from 'react';

// ERC20 ABI for approve and allowance
const ERC20_ABI = [
  {
    inputs: [
      { name: 'spender', type: 'address' },
      { name: 'amount', type: 'uint256' }
    ],
    name: 'approve',
    outputs: [{ name: '', type: 'bool' }],
    stateMutability: 'nonpayable',
    type: 'function',
  },
  {
    inputs: [
      { name: 'owner', type: 'address' },
      { name: 'spender', type: 'address' }
    ],
    name: 'allowance',
    outputs: [{ name: '', type: 'uint256' }],
    stateMutability: 'view',
    type: 'function',
  },
] as const;

// Define ZetaChain mainnet ID
const ZETACHAIN_ID = 7000;

// Update ChainName type to include all supported chains
type ChainName = 'BASE' | 'ARBITRUM' | 'ETHEREUM' | 'BSC' | 'POLYGON' | 'AVALANCHE' | 'ZETACHAIN';
type TokenSymbol = keyof typeof TOKENS[typeof base.id];

function getChainId(chainName: ChainName): number {
  switch (chainName) {
    case 'BASE':
      return base.id;
    case 'ARBITRUM':
      return arbitrum.id;
    case 'ETHEREUM':
      return mainnet.id;
    case 'BSC':
      return bsc.id;
    case 'POLYGON':
      return polygon.id;
    case 'AVALANCHE':
      return avalanche.id;
    case 'ZETACHAIN':
      return ZETACHAIN_ID;
    default:
      throw new Error(`Unsupported chain: ${chainName}`);
  }
}

export function useTokenApproval() {
  const { address } = useAccount();
  const { chain } = useNetwork();
  const { data: walletClient } = useWalletClient();
  const publicClient = usePublicClient();

  const approveToken = useCallback(async (
    chainName: ChainName,
    tokenSymbol: TokenSymbol,
    spender: `0x${string}`,
    amount: string
  ): Promise<`0x${string}` | undefined> => {
    try {
      // Check wallet connection
      if (!address) {
        throw new Error('Wallet address not available');
      }
      if (!walletClient) {
        throw new Error('Wallet client not initialized');
      }
      if (!publicClient) {
        throw new Error('Public client not initialized');
      }

      // Get chain ID and validate network
      const chainId = getChainId(chainName);
      if (chain?.id !== chainId) {
        throw new Error(`Please switch to ${chainName} network`);
      }

      // Get token configuration
      const tokenAddress = TOKENS[chainId][tokenSymbol].address as `0x${string}`;
      if (!tokenAddress) {
        throw new Error('Token not found');
      }

      // Create contract instance
      const contract = getContract({
        address: tokenAddress,
        abi: ERC20_ABI,
        walletClient,
        publicClient,
      });

      // Calculate amount in wei
      const amountWei = parseUnits(amount, TOKENS[chainId][tokenSymbol].decimals);

      // Check current allowance
      const currentAllowance = await contract.read.allowance([address, spender]);
      
      // If current allowance is sufficient, no need to approve
      if (currentAllowance >= amountWei) {
        return;
      }

      // Approve tokens
      const hash = await contract.write.approve([spender, amountWei]);

      return hash;
    } catch (error) {
      console.error('Error approving token:', error);
      throw error;
    }
  }, [address, walletClient, publicClient, chain?.id]);

  return {
    approveToken,
    isLoadingAllowance: false,
  };
} 