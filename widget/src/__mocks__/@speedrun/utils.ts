export const ChainName = {
  ETHEREUM: 'ETHEREUM',
  ARBITRUM: 'ARBITRUM',
  BASE: 'BASE',
  POLYGON: 'POLYGON',
} as const;

export type ChainName = typeof ChainName[keyof typeof ChainName];

export const TokenSymbol = {
  ETH: 'ETH',
  USDC: 'USDC',
  DAI: 'DAI',
} as const;

export type TokenSymbol = typeof TokenSymbol[keyof typeof TokenSymbol];

export const TOKENS = {
  ETH: {
    name: 'Ethereum',
    symbol: 'ETH',
    decimals: 18,
  },
  USDC: {
    name: 'USD Coin',
    symbol: 'USDC',
    decimals: 6,
  },
  DAI: {
    name: 'Dai',
    symbol: 'DAI',
    decimals: 18,
  },
};

export const getChainId = (chain: ChainName): number => {
  switch (chain) {
    case 'ETHEREUM':
      return 1;
    case 'ARBITRUM':
      return 42161;
    case 'BASE':
      return 8453;
    case 'POLYGON':
      return 137;
    default:
      throw new Error(`Unknown chain: ${chain}`);
  }
};

export const getChainName = (chainId: number): ChainName => {
  switch (chainId) {
    case 1:
      return 'ETHEREUM';
    case 42161:
      return 'ARBITRUM';
    case 8453:
      return 'BASE';
    case 137:
      return 'POLYGON';
    default:
      throw new Error(`Unknown chain ID: ${chainId}`);
  }
}; 