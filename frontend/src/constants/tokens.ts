import { base, arbitrum, avalanche, bsc, polygon, mainnet } from "wagmi/chains";

export type TokenSymbol = "USDC" | "USDT" | "BTC" | "ZETA";

interface Token {
  address: `0x${string}`;
  decimals: number;
  symbol: TokenSymbol;
}

interface TokenConfig {
  [key: string]: Token;
}

export const TOKENS: Record<number, TokenConfig> = {
  [base.id]: {
    USDC: {
      address: "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913",
      decimals: 6,
      symbol: "USDC",
    },
    USDT: {
      address: "0x50c5725949A6F0c72E6C4a641F24049A917DB0Cb",
      decimals: 6,
      symbol: "USDT",
    },
  },
  [arbitrum.id]: {
    USDC: {
      address: "0xaf88d065e77c8cC2239327C5EDb3A432268e5831",
      decimals: 6,
      symbol: "USDC",
    },
    USDT: {
      address: "0xFd086bC7CD5C481DCC9C85ebE478A1C0b69FCbb9",
      decimals: 6,
      symbol: "USDT",
    },
  },
  [mainnet.id]: {
    USDC: {
      address: "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48",
      decimals: 6,
      symbol: "USDC",
    },
    USDT: {
      address: "0xdAC17F958D2ee523a2206206994597C13D831ec7",
      decimals: 6,
      symbol: "USDT",
    },
  },
  [polygon.id]: {
    USDC: {
      address: "0x3c499c542cEF5E3811e1192ce70d8cC03d5c3359",
      decimals: 6,
      symbol: "USDC",
    },
    USDT: {
      address: "0xc2132D05D31c914a87C6611C10748AEb04B58e8F",
      decimals: 6,
      symbol: "USDT",
    },
  },
  [bsc.id]: {
    USDC: {
      address: "0x8AC76a51cc950d9822D68b83fE1Ad97B32Cd580d",
      decimals: 18,
      symbol: "USDC",
    },
    USDT: {
      address: "0x55d398326f99059fF775485246999027B3197955",
      decimals: 18,
      symbol: "USDT",
    },
  },
  [avalanche.id]: {
    USDC: {
      address: "0xB97EF9Ef8734C71904D8002F8b6Bc66Dd9c48a6E",
      decimals: 6,
      symbol: "USDC",
    },
    USDT: {
      address: "0x9702230A8Ea53601f5cD2dc00fDBc13d4dF4A8c7",
      decimals: 6,
      symbol: "USDT",
    },
  },
};
