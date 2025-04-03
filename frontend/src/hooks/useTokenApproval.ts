import { useAccount, useContractWrite, useContractRead } from 'wagmi';
import { parseUnits } from 'viem';
import { TOKENS } from '@/constants/tokens';
import { getChainId } from '@/utils/chain';

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

export function useTokenApproval() {
  const { address } = useAccount();
  const { writeAsync: approve } = useContractWrite({
    abi: ERC20_ABI,
    functionName: 'approve',
  });

  const { data: allowance, isLoading: isLoadingAllowance } = useContractRead({
    abi: ERC20_ABI,
    functionName: 'allowance',
  });

  const approveToken = async (
    chainName: string,
    tokenSymbol: string,
    spender: string,
    amount: string
  ) => {
    if (!address) throw new Error('Wallet not connected');

    const chainId = getChainId(chainName);
    const tokenConfig = TOKENS[chainId]?.[tokenSymbol];
    if (!tokenConfig) throw new Error('Invalid token configuration');

    const amountWei = parseUnits(amount, tokenConfig.decimals);

    // Check current allowance
    const currentAllowance = await allowance?.({
      address: tokenConfig.address,
      args: [address, spender],
      chainId,
    });

    // If current allowance is sufficient, no need to approve
    if (currentAllowance && currentAllowance >= amountWei) {
      return;
    }

    // Approve tokens
    const { hash } = await approve({
      address: tokenConfig.address,
      args: [spender, amountWei],
      chainId,
    });

    return hash;
  };

  return {
    approveToken,
    isLoadingAllowance,
  };
} 