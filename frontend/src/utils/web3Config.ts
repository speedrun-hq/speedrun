'use client';

import { getDefaultWallets, connectorsForWallets } from '@rainbow-me/rainbowkit';
import { configureChains, createConfig } from 'wagmi';
import { base, arbitrum } from 'wagmi/chains';
import { alchemyProvider } from 'wagmi/providers/alchemy';
import { publicProvider } from 'wagmi/providers/public';

if (!process.env.NEXT_PUBLIC_WALLETCONNECT_PROJECT_ID) {
  throw new Error('Missing NEXT_PUBLIC_WALLETCONNECT_PROJECT_ID');
}

if (!process.env.NEXT_PUBLIC_ALCHEMY_ID) {
  throw new Error('Missing NEXT_PUBLIC_ALCHEMY_ID');
}

const projectId = process.env.NEXT_PUBLIC_WALLETCONNECT_PROJECT_ID;
const alchemyId = process.env.NEXT_PUBLIC_ALCHEMY_ID;

console.log('Configuring chains with Alchemy ID:', alchemyId);

// Configure Base chain with custom RPC URL
const customBase = {
  ...base,
  rpcUrls: {
    ...base.rpcUrls,
    default: {
      http: [`https://base-mainnet.g.alchemy.com/v2/${alchemyId}`],
    },
    public: {
      http: ['https://mainnet.base.org'],
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
      http: ['https://arb1.arbitrum.io/rpc'],
    },
  },
};

const { chains, publicClient } = configureChains(
  [customBase, customArbitrum],
  [
    alchemyProvider({ apiKey: alchemyId }),
    publicProvider(), // Fallback to public provider
  ]
);

console.log('Configured chains:', chains.map(chain => ({
  id: chain.id,
  name: chain.name,
  rpcUrls: chain.rpcUrls,
})));

const { wallets } = getDefaultWallets({
  appName: 'ZetaFast',
  projectId,
  chains,
});

const connectors = connectorsForWallets([
  ...wallets,
]);

const config = createConfig({
  autoConnect: true,
  connectors,
  publicClient,
});

export { chains, config }; 