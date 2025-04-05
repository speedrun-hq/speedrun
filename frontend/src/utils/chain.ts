import { base, arbitrum, mainnet, bsc, polygon, avalanche } from 'wagmi/chains';

export type ChainName = 'ETHEREUM' | 'BSC' | 'POLYGON' | 'BASE' | 'ARBITRUM' | 'AVALANCHE';

export function getChainId(chainName: ChainName): number {
  switch (chainName) {
    case 'ETHEREUM':
      return mainnet.id;
    case 'BSC':
      return bsc.id;
    case 'POLYGON':
      return polygon.id;
    case 'BASE':
      return base.id;
    case 'ARBITRUM':
      return arbitrum.id;
    case 'AVALANCHE':
      return avalanche.id;
    default:
      return base.id; // Default to BASE if unknown
  }
}

export function getChainName(chainId: number): ChainName {
  switch (chainId) {
    case mainnet.id:
      return 'ETHEREUM';
    case bsc.id:
      return 'BSC';
    case polygon.id:
      return 'POLYGON';
    case base.id:
      return 'BASE';
    case arbitrum.id:
      return 'ARBITRUM';
    case avalanche.id:
      return 'AVALANCHE';
    default:
      return 'BASE'; // Default to BASE if unknown
  }
}

export function isValidChainId(chainId: number): boolean {
  return (
    chainId === mainnet.id ||
    chainId === bsc.id ||
    chainId === polygon.id ||
    chainId === base.id ||
    chainId === arbitrum.id ||
    chainId === avalanche.id
  );
}

export function getChainRpcUrl(chainId: number): string {
  switch (chainId) {
    case mainnet.id:
      return process.env.NEXT_PUBLIC_ETHEREUM_RPC_URL || 'https://eth.llamarpc.com';
    case bsc.id:
      return process.env.NEXT_PUBLIC_BSC_RPC_URL || 'https://bsc-dataseed.bnbchain.org';
    case polygon.id:
      return process.env.NEXT_PUBLIC_POLYGON_RPC_URL || 'https://polygon-rpc.com';
    case base.id:
      return process.env.NEXT_PUBLIC_BASE_RPC_URL || 'https://mainnet.base.org';
    case arbitrum.id:
      return process.env.NEXT_PUBLIC_ARBITRUM_RPC_URL || 'https://arb1.arbitrum.io/rpc';
    case avalanche.id:
      return process.env.NEXT_PUBLIC_AVALANCHE_RPC_URL || 'https://avalanche-c-chain-rpc.publicnode.com';
    default:
      return '';
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
      return '';
  }
} 