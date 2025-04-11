import { Chain } from "wagmi";
import { base, arbitrum, mainnet, bsc, polygon, avalanche } from "wagmi/chains";

// Custom chain IDs for upcoming chains
export const BITCOIN_CHAIN_ID = 8332;
export const SOLANA_CHAIN_ID = 900;
export const ZETACHAIN_CHAIN_ID = 7000;

// Type to include both real and custom chain IDs
export type ChainId =
  | typeof mainnet.id
  | typeof bsc.id
  | typeof polygon.id
  | typeof base.id
  | typeof arbitrum.id
  | typeof avalanche.id
  | typeof BITCOIN_CHAIN_ID
  | typeof SOLANA_CHAIN_ID
  | typeof ZETACHAIN_CHAIN_ID;

export type ChainName =
  | "ETHEREUM"
  | "BSC"
  | "POLYGON"
  | "BASE"
  | "ARBITRUM"
  | "AVALANCHE"
  | "BITCOIN"
  | "SOLANA"
  | "ZETACHAIN";

// Create custom chain configurations
export const getCustomChains = (alchemyId: string): Chain[] => {
  // Configure Ethereum chain with custom RPC URL
  const customMainnet = {
    ...mainnet,
    rpcUrls: {
      ...mainnet.rpcUrls,
      default: {
        http: [`https://eth-mainnet.g.alchemy.com/v2/${alchemyId}`],
      },
      public: {
        http: ["https://eth.llamarpc.com"],
      },
    },
  };

  // Configure BNB Chain with custom RPC URL
  const customBsc = {
    ...bsc,
    rpcUrls: {
      ...bsc.rpcUrls,
      default: {
        http: ["https://bsc-dataseed.bnbchain.org"],
      },
      public: {
        http: ["https://bsc-dataseed.bnbchain.org"],
      },
    },
  };

  // Configure Polygon chain with custom RPC URL
  const customPolygon = {
    ...polygon,
    rpcUrls: {
      ...polygon.rpcUrls,
      default: {
        http: [`https://polygon-mainnet.g.alchemy.com/v2/${alchemyId}`],
      },
      public: {
        http: ["https://polygon-rpc.com"],
      },
    },
  };

  // Configure Base chain with custom RPC URL
  const customBase = {
    ...base,
    rpcUrls: {
      ...base.rpcUrls,
      default: {
        http: [`https://base-mainnet.g.alchemy.com/v2/${alchemyId}`],
      },
      public: {
        http: ["https://mainnet.base.org"],
      },
    },
  };

  // Configure Arbitrum chain with custom RPC URL
  const customArbitrum = {
    ...arbitrum,
    rpcUrls: {
      ...arbitrum.rpcUrls,
      default: {
        http: [`https://arb-mainnet.g.alchemy.com/v2/${alchemyId}`],
      },
      public: {
        http: ["https://arb1.arbitrum.io/rpc"],
      },
    },
  };

  // Configure Avalanche chain with custom RPC URL
  const customAvalanche = {
    ...avalanche,
    rpcUrls: {
      ...avalanche.rpcUrls,
      default: {
        http: ["https://avalanche-c-chain-rpc.publicnode.com"],
      },
      public: {
        http: ["https://avalanche-c-chain-rpc.publicnode.com"],
      },
    },
  };

  return [
    customMainnet,
    customBsc,
    customPolygon,
    customBase,
    customArbitrum,
    customAvalanche,
  ];
};

// Supported EVM chains data
export const SUPPORTED_CHAINS = [
  { id: mainnet.id, name: "ETHEREUM" as ChainName },
  { id: bsc.id, name: "BSC" as ChainName },
  { id: polygon.id, name: "POLYGON" as ChainName },
  { id: base.id, name: "BASE" as ChainName },
  { id: arbitrum.id, name: "ARBITRUM" as ChainName },
  { id: avalanche.id, name: "AVALANCHE" as ChainName },
];

// Coming soon chains - Source selector (FROM)
export const COMING_SOON_SOURCE_CHAINS = [
  { id: ZETACHAIN_CHAIN_ID, name: "ZETACHAIN" as ChainName },
  { id: SOLANA_CHAIN_ID, name: "SOLANA" as ChainName },
  { id: BITCOIN_CHAIN_ID, name: "BITCOIN" as ChainName },
];

// Coming soon chains - Destination selector (TO)
export const COMING_SOON_DESTINATION_CHAINS = [
  { id: ZETACHAIN_CHAIN_ID, name: "ZETACHAIN" as ChainName },
  { id: SOLANA_CHAIN_ID, name: "SOLANA" as ChainName },
  { id: BITCOIN_CHAIN_ID, name: "BITCOIN" as ChainName },
];

// Helper functions
export function getChainId(chainName: ChainName): number {
  const chain = SUPPORTED_CHAINS.find((c) => c.name === chainName);
  if (chain) return chain.id;

  // Check coming soon chains
  const comingSoonChain = [
    ...COMING_SOON_SOURCE_CHAINS,
    ...COMING_SOON_DESTINATION_CHAINS,
  ].find((c) => c.name === chainName);
  if (comingSoonChain) return comingSoonChain.id;

  return base.id; // Default to BASE if unknown
}

export function getChainName(chainId: number): ChainName {
  const chain = SUPPORTED_CHAINS.find((c) => c.id === chainId);
  if (chain) return chain.name;

  // Check coming soon chains
  const comingSoonChain = [
    ...COMING_SOON_SOURCE_CHAINS,
    ...COMING_SOON_DESTINATION_CHAINS,
  ].find((c) => c.id === chainId);
  if (comingSoonChain) return comingSoonChain.name;

  return "BASE"; // Default to BASE if unknown
}

export function isValidChainId(chainId: number): boolean {
  return SUPPORTED_CHAINS.some((chain) => chain.id === chainId);
}

export function getChainRpcUrl(chainId: number): string {
  switch (chainId) {
    case mainnet.id:
      return (
        process.env.NEXT_PUBLIC_ETHEREUM_RPC_URL || "https://eth.llamarpc.com"
      );
    case bsc.id:
      return (
        process.env.NEXT_PUBLIC_BSC_RPC_URL ||
        "https://bsc-dataseed.bnbchain.org"
      );
    case polygon.id:
      return (
        process.env.NEXT_PUBLIC_POLYGON_RPC_URL || "https://polygon-rpc.com"
      );
    case base.id:
      return process.env.NEXT_PUBLIC_BASE_RPC_URL || "https://mainnet.base.org";
    case arbitrum.id:
      return (
        process.env.NEXT_PUBLIC_ARBITRUM_RPC_URL ||
        "https://arb1.arbitrum.io/rpc"
      );
    case avalanche.id:
      return (
        process.env.NEXT_PUBLIC_AVALANCHE_RPC_URL ||
        "https://avalanche-c-chain-rpc.publicnode.com"
      );
    default:
      return "";
  }
}

export function getExplorerUrl(chainId: number, txHash: string): string {
  switch (chainId) {
    case mainnet.id:
      return `https://etherscan.io/tx/${txHash}`;
    case bsc.id:
      return `https://bscscan.com/tx/${txHash}`;
    case polygon.id:
      return `https://polygonscan.com/tx/${txHash}`;
    case base.id:
      return `https://basescan.org/tx/${txHash}`;
    case arbitrum.id:
      return `https://arbiscan.io/tx/${txHash}`;
    case avalanche.id:
      return `https://snowtrace.io/tx/${txHash}`;
    default:
      return "";
  }
}
