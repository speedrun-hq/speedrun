import { useAccount, usePublicClient, useWalletClient } from "wagmi";
import { getContract } from "viem";
import { arbitrum } from "viem/chains";
import { Intent as IntentContract } from "../contracts/Intent";

export function useContractService() {
  const { address } = useAccount();
  const publicClient = usePublicClient();
  const { data: walletClient } = useWalletClient();

  const getIntentContract = () => {
    if (!publicClient || !walletClient) return null;

    return getContract({
      address: IntentContract.address[arbitrum.id] as `0x${string}`, // Using Arbitrum address
      abi: IntentContract.abi,
      publicClient,
      walletClient,
    });
  };

  return {
    getIntentContract,
    address,
    publicClient,
    walletClient,
  };
}
