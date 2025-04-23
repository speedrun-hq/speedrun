"use client";

import {
  getDefaultWallets,
  connectorsForWallets,
} from "@rainbow-me/rainbowkit";
import { configureChains, createConfig } from "wagmi";
import { alchemyProvider } from "wagmi/providers/alchemy";
import { publicProvider } from "wagmi/providers/public";
import { getCustomChains } from "@/config/chainConfig";

// Check for required environment variables
const projectId = process.env.NEXT_PUBLIC_WALLETCONNECT_PROJECT_ID;
const alchemyId = process.env.NEXT_PUBLIC_ALCHEMY_ID;

if (!projectId) {
  console.error(
    "Missing NEXT_PUBLIC_WALLETCONNECT_PROJECT_ID environment variable",
  );
}

if (!alchemyId) {
  console.error("Missing NEXT_PUBLIC_ALCHEMY_ID environment variable");
}

console.log(
  "Configuring chains with Alchemy ID:",
  alchemyId ? "✓ Available" : "✗ Missing",
);

// Get custom chain configurations from our centralized config
const customChains = getCustomChains(alchemyId || "demo");

// Configure chains with appropriate providers
const providers = [];
if (alchemyId) {
  providers.push(alchemyProvider({ apiKey: alchemyId }));
}
providers.push(publicProvider());

const { chains, publicClient } = configureChains(customChains, providers);

// Debug logging for chain configuration
console.log("Chains configured:", chains.map((chain) => chain.name).join(", "));

// Configure wallet options
const { wallets } = getDefaultWallets({
  appName: "Speedrun",
  projectId: projectId || "",
  chains,
});

const connectors = connectorsForWallets([...wallets]);

// Create wagmi config
const config = createConfig({
  autoConnect: true,
  connectors,
  publicClient,
});

export { chains, config };
