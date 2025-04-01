import { useContractRead } from 'wagmi';
import { formatUnits } from 'viem';
import { TOKENS } from '../config/tokens';

// ERC20 ABI for balanceOf
const ERC20_ABI = [
  {
    constant: true,
    inputs: [{ name: '_owner', type: 'address' }],
    name: 'balanceOf',
    outputs: [{ name: 'balance', type: 'uint256' }],
    type: 'function',
  },
] as const;

export function useTokenBalance(
  tokenSymbol: 'USDC' | 'USDT',
  userAddress: string | undefined,
  chainId: number
) {
  const tokenConfig = TOKENS[chainId]?.[tokenSymbol];
  const tokenAddress = tokenConfig?.address;
  const decimals = tokenConfig?.decimals ?? 6;

  const { data: balance, isError, isLoading } = useContractRead({
    address: tokenAddress as `0x${string}`,
    abi: ERC20_ABI,
    functionName: 'balanceOf',
    args: userAddress ? [userAddress as `0x${string}`] : undefined,
    chainId,
    enabled: !!tokenAddress && !!userAddress,
  });

  // Add console logs for debugging
  console.log('Token Config:', {
    symbol: tokenSymbol,
    address: tokenAddress,
    decimals,
    chainId,
    userAddress,
  });
  console.log('Raw Balance:', balance);

  const formattedBalance = balance 
    ? Number(formatUnits(balance as bigint, decimals)).toFixed(2)
    : '0.00';

  console.log('Formatted Balance:', formattedBalance);

  return {
    balance: formattedBalance,
    isError,
    isLoading,
    symbol: tokenSymbol,
  };
} 