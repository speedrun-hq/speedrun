import { base, arbitrum } from 'wagmi/chains';

export type TokenSymbol = 'USDC' | 'USDT';

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
      address: '0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913',
      decimals: 6,
      symbol: 'USDC',
    },
    USDT: {
      address: '0x50c5725949A6F0c72E6C4a641F24049A917DB0Cb',
      decimals: 6,
      symbol: 'USDT',
    },
  },
  [arbitrum.id]: {
    USDC: {
      address: '0xaf88d065e77c8cC2239327C5EDb3A432268e5831',
      decimals: 6,
      symbol: 'USDC',
    },
    USDT: {
      address: '0xFd086bC7CD5C481DCC9C85ebE478A1C0b69FCbb9',
      decimals: 6,
      symbol: 'USDT',
    },
  },
}; 