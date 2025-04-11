"use client";

import { ConnectButton } from "@rainbow-me/rainbowkit";

export function ConnectWallet() {
  return (
    <ConnectButton.Custom>
      {({
        account,
        chain,
        openAccountModal,
        openChainModal,
        openConnectModal,
        mounted,
      }) => {
        const ready = mounted;
        const connected = ready && account && chain;

        return (
          <div
            {...(!ready && {
              "aria-hidden": true,
              style: {
                opacity: 0,
                pointerEvents: "none",
                userSelect: "none",
              },
            })}
            className="flex justify-center items-center"
          >
            {(() => {
              if (!connected) {
                return (
                  <button
                    onClick={openConnectModal}
                    type="button"
                    className="arcade-btn-sm border-yellow-500 text-yellow-500 hover:bg-yellow-500 hover:text-black transition-all duration-200 arcade-text text-xs min-w-[120px] justify-center"
                  >
                    CONNECT WALLET
                  </button>
                );
              }

              if (chain.unsupported) {
                return (
                  <button
                    onClick={openChainModal}
                    type="button"
                    className="arcade-btn-sm border-red-500 text-red-500 hover:bg-red-500 hover:text-black transition-all duration-200 arcade-text text-xs min-w-[120px] justify-center"
                  >
                    WRONG NETWORK
                  </button>
                );
              }

              return (
                <div className="flex items-center gap-2">
                  <button
                    onClick={openChainModal}
                    type="button"
                    className="flex items-center arcade-btn-sm border-primary-500 text-primary-500 hover:bg-primary-500 hover:text-black transition-all duration-200 arcade-text text-xs px-2 py-1"
                  >
                    {chain.hasIcon && (
                      <div
                        className="w-4 h-4 rounded-full overflow-hidden mr-1 flex items-center justify-center"
                        style={{ background: chain.iconBackground }}
                      >
                        {chain.iconUrl && (
                          <img
                            alt={chain.name ?? "Chain icon"}
                            src={chain.iconUrl}
                            className="w-3 h-3"
                          />
                        )}
                      </div>
                    )}
                    {chain.name}
                  </button>

                  <button
                    onClick={openAccountModal}
                    type="button"
                    className="arcade-btn-sm border-yellow-500 text-yellow-500 hover:bg-yellow-500 hover:text-black transition-all duration-200 arcade-text text-xs px-2 py-1"
                  >
                    {account.displayName}
                  </button>
                </div>
              );
            })()}
          </div>
        );
      }}
    </ConnectButton.Custom>
  );
}
