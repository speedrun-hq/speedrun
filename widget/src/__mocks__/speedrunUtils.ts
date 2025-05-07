// Mock implementation for @speedrun/utils
export const getChainId = (chainName: string) => {
  const chains: Record<string, number> = {
    'ARBITRUM': 42161,
    'BASE': 8453,
    'ETHEREUM': 1,
    'POLYGON': 137,
    'AVALANCHE': 43114,
    'BSC': 56
  };
  return chains[chainName] || 1;
};

export const getChainName = (chainId: number) => {
  const chains: Record<number, string> = {
    42161: 'ARBITRUM',
    8453: 'BASE',
    1: 'ETHEREUM',
    137: 'POLYGON',
    43114: 'AVALANCHE',
    56: 'BSC'
  };
  return chains[chainId] || 'ETHEREUM';
};

export const TOKENS = {
  42161: {
    USDC: { address: '0xarbitrum_usdc', decimals: 6, symbol: 'USDC' },
    ETH: { address: '0xarbitrum_eth', decimals: 18, symbol: 'ETH' },
    DAI: { address: '0xarbitrum_dai', decimals: 18, symbol: 'DAI' }
  },
  8453: {
    USDC: { address: '0xbase_usdc', decimals: 6, symbol: 'USDC' },
    ETH: { address: '0xbase_eth', decimals: 18, symbol: 'ETH' },
    DAI: { address: '0xbase_dai', decimals: 18, symbol: 'DAI' }
  },
  1: {
    USDC: { address: '0xeth_usdc', decimals: 6, symbol: 'USDC' },
    ETH: { address: '0xeth_eth', decimals: 18, symbol: 'ETH' },
    DAI: { address: '0xeth_dai', decimals: 18, symbol: 'DAI' }
  }
};

export enum ChainName {
  ARBITRUM = 'ARBITRUM',
  BASE = 'BASE',
  ETHEREUM = 'ETHEREUM',
  POLYGON = 'POLYGON',
  AVALANCHE = 'AVALANCHE',
  BSC = 'BSC'
}

export enum TokenSymbol {
  USDC = 'USDC',
  ETH = 'ETH',
  DAI = 'DAI'
} 