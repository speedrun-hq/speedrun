"use client";

import { RainbowKitProvider } from "@rainbow-me/rainbowkit";
import "@rainbow-me/rainbowkit/styles.css";
import { WagmiConfig } from "wagmi";
import { config, chains } from "../utils/web3Config";
import { arcadeTheme } from "../utils/rainbowKitTheme";
import { RpcTest } from "./RpcTest";

// Check if the required environment variable is set
if (!process.env.NEXT_PUBLIC_WALLETCONNECT_PROJECT_ID) {
  throw new Error("NEXT_PUBLIC_WALLETCONNECT_PROJECT_ID is not defined");
}

export function Web3Provider({ children }: { children: React.ReactNode }) {
  return (
    <WagmiConfig config={config}>
      <RainbowKitProvider
        chains={chains}
        theme={arcadeTheme}
        modalSize="compact"
      >
        {process.env.NODE_ENV === "test" && <RpcTest />}
        {children}
      </RainbowKitProvider>
    </WagmiConfig>
  );
}
