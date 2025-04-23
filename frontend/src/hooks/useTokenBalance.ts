import { useAccount, useContractRead, useNetwork } from "wagmi";
import { formatUnits } from "viem";
import { TOKENS, TokenSymbol } from "@/config/chainConfig";
import { ChainName, getChainId } from "@/utils/chain";

// ERC20 ABI for balanceOf
const ERC20_ABI = [
  {
    inputs: [{ name: "account", type: "address" }],
    name: "balanceOf",
    outputs: [{ name: "", type: "uint256" }],
    stateMutability: "view",
    type: "function",
  },
] as const;

export function useTokenBalance(
  chainName: ChainName,
  tokenSymbol: TokenSymbol,
) {
  const { address } = useAccount();
  const { chain } = useNetwork();
  const chainId = getChainId(chainName);
  const tokenConfig = TOKENS[chainId]?.[tokenSymbol];
  const tokenAddress = tokenConfig?.address;

  console.log("useTokenBalance Debug:", {
    currentChain: chain?.name,
    currentChainId: chain?.id,
    requestedChain: chainName,
    requestedChainId: chainId,
    tokenSymbol,
    tokenAddress,
    userAddress: address,
  });

  // Use configured decimals value
  const decimals = tokenConfig?.decimals ?? 6;

  // Get balance
  const {
    data: balance,
    isError,
    isLoading,
    error: balanceError,
  } = useContractRead({
    address: tokenAddress,
    abi: ERC20_ABI,
    functionName: "balanceOf",
    args: address ? [address] : undefined,
    chainId,
    enabled: !!tokenAddress && !!address,
  });

  if (balanceError) {
    console.error("Error fetching balance:", balanceError);
  }

  // Add console logs for debugging
  console.log("Token Balance Debug:", {
    decimals,
    balance,
    balanceError,
    isError,
    isLoading,
  });

  const formattedBalance = balance
    ? Number(formatUnits(balance as bigint, decimals)).toFixed(2)
    : "0.00";

  return {
    balance: formattedBalance,
    isError,
    isLoading,
    symbol: tokenSymbol,
  };
}
