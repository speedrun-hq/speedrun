"use client";

import {
  getDefaultWallets,
  connectorsForWallets,
} from "@rainbow-me/rainbowkit";
import { configureChains, createConfig } from "wagmi";
import { alchemyProvider } from "wagmi/providers/alchemy";
import { publicProvider } from "wagmi/providers/public";
import { getCustomChains } from "@/config/chains";

if (!process.env.NEXT_PUBLIC_WALLETCONNECT_PROJECT_ID) {
  throw new Error("Missing NEXT_PUBLIC_WALLETCONNECT_PROJECT_ID");
}

if (!process.env.NEXT_PUBLIC_ALCHEMY_ID) {
  throw new Error("Missing NEXT_PUBLIC_ALCHEMY_ID");
}

const projectId = process.env.NEXT_PUBLIC_WALLETCONNECT_PROJECT_ID;
const alchemyId = process.env.NEXT_PUBLIC_ALCHEMY_ID;

console.log("Configuring chains with Alchemy ID:", alchemyId);

// Get custom chain configurations from our centralized config
const customChains = getCustomChains(alchemyId);

const { chains, publicClient } = configureChains(
  customChains,
  [
    alchemyProvider({ apiKey: alchemyId }),
    publicProvider(), // Fallback to public provider
  ],
);

console.log(
  "Configured chains:",
  chains.map((chain) => ({
    id: chain.id,
    name: chain.name,
    rpcUrls: chain.rpcUrls,
  })),
);

const { wallets } = getDefaultWallets({
  appName: "Speedrun",
  projectId,
  chains,
});

const connectors = connectorsForWallets([...wallets]);

const config = createConfig({
  autoConnect: true,
  connectors,
  publicClient,
});

export { chains, config };
