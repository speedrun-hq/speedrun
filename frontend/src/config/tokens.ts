interface TokenConfig {
  address: string;
  decimals: number;
  symbol: string;
  name: string;
}

interface ChainTokens {
  USDC: TokenConfig;
  USDT: TokenConfig;
}

export const TOKENS: Record<number, ChainTokens> = {
  // Base Mainnet
  8453: {
    USDC: {
      address: "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913",
      decimals: 6,
      symbol: "USDC",
      name: "USD Coin",
    },
    USDT: {
      address: "0x50c5725949A6F0c72E6C4a641F24049A917DB0Cb",
      decimals: 6,
      symbol: "USDT",
      name: "Tether USD",
    },
  },
  // Arbitrum Mainnet
  42161: {
    USDC: {
      address: "0xaf88d065e77c8cC2239327C5EDb3A432268e5831",
      decimals: 6,
      symbol: "USDC",
      name: "USD Coin",
    },
    USDT: {
      address: "0xFd086bC7CD5C481DCC9C85ebE478A1C0b69FCbb9",
      decimals: 6,
      symbol: "USDT",
      name: "Tether USD",
    },
  },
} as const;
