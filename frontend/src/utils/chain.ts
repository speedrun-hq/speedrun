import { base, arbitrum } from 'wagmi/chains';

export type ChainName = 'BASE' | 'ARBITRUM';

export function getChainId(chainName: ChainName): number {
  return chainName === 'BASE' ? base.id : arbitrum.id;
}

export function getChainName(chainId: number): ChainName {
  return chainId === base.id ? 'BASE' : 'ARBITRUM';
}

export function isValidChainId(chainId: number): boolean {
  return chainId === base.id || chainId === arbitrum.id;
}

export function getChainRpcUrl(chainId: number): string {
  switch (chainId) {
    case base.id:
      return process.env.NEXT_PUBLIC_BASE_RPC_URL || '';
    case arbitrum.id:
      return process.env.NEXT_PUBLIC_ARBITRUM_RPC_URL || '';
    default:
      return '';
  }
} 