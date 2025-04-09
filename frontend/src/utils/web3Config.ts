"use client";

import {
  getDefaultWallets,
  connectorsForWallets,
} from "@rainbow-me/rainbowkit";
import { configureChains, createConfig } from "wagmi";
import { base, arbitrum, mainnet, bsc, polygon, avalanche } from "wagmi/chains";
import { alchemyProvider } from "wagmi/providers/alchemy";
import { publicProvider } from "wagmi/providers/public";

if (!process.env.NEXT_PUBLIC_WALLETCONNECT_PROJECT_ID) {
  throw new Error("Missing NEXT_PUBLIC_WALLETCONNECT_PROJECT_ID");
}

if (!process.env.NEXT_PUBLIC_ALCHEMY_ID) {
  throw new Error("Missing NEXT_PUBLIC_ALCHEMY_ID");
}

const projectId = process.env.NEXT_PUBLIC_WALLETCONNECT_PROJECT_ID;
const alchemyId = process.env.NEXT_PUBLIC_ALCHEMY_ID;

console.log("Configuring chains with Alchemy ID:", alchemyId);

// Configure Ethereum chain with custom RPC URL
const customMainnet = {
  ...mainnet,
  rpcUrls: {
    ...mainnet.rpcUrls,
    default: {
      http: [`https://eth-mainnet.g.alchemy.com/v2/${alchemyId}`],
    },
    public: {
      http: ["https://eth.llamarpc.com"],
    },
  },
};

// Configure BNB Chain with custom RPC URL
const customBsc = {
  ...bsc,
  rpcUrls: {
    ...bsc.rpcUrls,
    default: {
      http: ["https://bsc-dataseed.bnbchain.org"],
    },
    public: {
      http: ["https://bsc-dataseed.bnbchain.org"],
    },
  },
};

// Configure Polygon chain with custom RPC URL
const customPolygon = {
  ...polygon,
  rpcUrls: {
    ...polygon.rpcUrls,
    default: {
      http: [`https://polygon-mainnet.g.alchemy.com/v2/${alchemyId}`],
    },
    public: {
      http: ["https://polygon-rpc.com"],
    },
  },
};

// Configure Base chain with custom RPC URL
const customBase = {
  ...base,
  rpcUrls: {
    ...base.rpcUrls,
    default: {
      http: [`https://base-mainnet.g.alchemy.com/v2/${alchemyId}`],
    },
    public: {
      http: ["https://mainnet.base.org"],
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
      http: ["https://arb1.arbitrum.io/rpc"],
    },
  },
};

// Configure Avalanche chain with custom RPC URL
const customAvalanche = {
  ...avalanche,
  rpcUrls: {
    ...avalanche.rpcUrls,
    default: {
      http: ["https://avalanche-c-chain-rpc.publicnode.com"],
    },
    public: {
      http: ["https://avalanche-c-chain-rpc.publicnode.com"],
    },
  },
};

const { chains, publicClient } = configureChains(
  [
    customMainnet,
    customBsc,
    customPolygon,
    customBase,
    customArbitrum,
    customAvalanche,
  ],
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
