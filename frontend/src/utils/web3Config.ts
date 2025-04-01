'use client';

import { getDefaultWallets, connectorsForWallets } from '@rainbow-me/rainbowkit';
import { configureChains, createConfig } from 'wagmi';
import { base, arbitrum } from 'wagmi/chains';
import { publicProvider } from 'wagmi/providers/public';

if (!process.env.NEXT_PUBLIC_WALLETCONNECT_PROJECT_ID) {
  throw new Error('Missing NEXT_PUBLIC_WALLETCONNECT_PROJECT_ID');
}

const projectId = process.env.NEXT_PUBLIC_WALLETCONNECT_PROJECT_ID;

const { chains, publicClient } = configureChains(
  [base, arbitrum],
  [publicProvider()]
);

const { wallets } = getDefaultWallets({
  appName: 'ZetaFast',
  projectId,
  chains,
});

const connectors = connectorsForWallets([
  ...wallets,
]);

export const config = createConfig({
  autoConnect: true,
  connectors,
  publicClient,
});

export { chains }; 