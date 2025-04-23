"use client";

import { useState, useEffect } from "react";
import { useParams } from "next/navigation";
import { apiService } from "@/services/api";
import { Intent, Fulfillment, CHAIN_ID_TO_NAME } from "@/types";
import Link from "next/link";
import { getExplorerUrl } from "@/utils/chain";
import { TOKENS } from "@/config/chainConfig";

export default function IntentPage() {
  const { id } = useParams();
  const [intent, setIntent] = useState<Intent | null>(null);
  const [fulfillment, setFulfillment] = useState<Fulfillment | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  useEffect(() => {
    async function fetchIntent() {
      if (!id) return;

      try {
        setIsLoading(true);
        const intentData = await apiService.getIntent(id as string);
        setIntent(intentData);

        // If intent is fulfilled or settled, fetch the fulfillment data
        if (
          intentData.status === "fulfilled" ||
          intentData.status === "settled"
        ) {
          try {
            const fulfillmentData = await apiService.getFulfillment(
              id as string,
            );
            setFulfillment(fulfillmentData);
          } catch (fulfillError) {
            console.error("Error fetching fulfillment:", fulfillError);
            // Don't set the main error, just log it
          }
        }
      } catch (err) {
        console.error("Error fetching intent:", err);
        setError(
          err instanceof Error ? err : new Error("Failed to fetch intent"),
        );
      } finally {
        setIsLoading(false);
      }
    }

    fetchIntent();
  }, [id]);

  // Format date in a human-readable format
  const formatDate = (dateString: string) => {
    try {
      if (!dateString) return "";
      return new Date(dateString).toLocaleString();
    } catch (error) {
      console.error("Error formatting date:", error);
      return dateString || "";
    }
  };

  // Get human-readable chain name
  const getChainName = (chainId: string) => {
    const id = parseInt(chainId);
    return CHAIN_ID_TO_NAME[id] || chainId;
  };

  // Truncate long strings (like addresses and IDs)
  const truncateText = (text: string, startLength = 8, endLength = 6) => {
    if (!text) return ""; // Handle undefined or empty strings
    if (text.length <= startLength + endLength + 3) return text;
    return `${text.substring(0, startLength)}...${text.substring(text.length - endLength)}`;
  };

  // Format token amount with correct decimals
  const formatTokenAmount = (
    amount: string,
    tokenAddress: string,
    sourceChain: string,
  ) => {
    try {
      if (!amount || !tokenAddress || !sourceChain) return "0"; // Handle undefined inputs

      const chainId = parseInt(sourceChain);

      // Find token symbol and decimals by matching address
      let tokenSymbol = "";
      let decimals = 18; // Default to 18 if not found

      if (TOKENS[chainId]) {
        // Loop through tokens to find matching address
        Object.entries(TOKENS[chainId]).forEach(([symbol, token]) => {
          if (token.address.toLowerCase() === tokenAddress.toLowerCase()) {
            tokenSymbol = symbol;
            decimals = token.decimals;
          }
        });
      }

      // Format amount based on decimals
      const formattedAmount = parseFloat(amount) / Math.pow(10, decimals);

      // If token symbol was found, add it to the display
      if (tokenSymbol) {
        return `${formattedAmount.toFixed(6)} ${tokenSymbol}`;
      }

      // Fallback to just showing the formatted amount
      return `${formattedAmount.toFixed(6)}`;
    } catch (error) {
      console.error("Error formatting token amount:", error);
      return amount || "0"; // Fallback to original amount or 0
    }
  };

  // Get token symbol from address
  const getTokenSymbol = (tokenAddress: string, sourceChain: string) => {
    try {
      if (!tokenAddress || !sourceChain) return ""; // Handle undefined inputs

      const chainId = parseInt(sourceChain);

      if (TOKENS[chainId]) {
        // Loop through tokens to find matching address
        for (const [symbol, token] of Object.entries(TOKENS[chainId])) {
          if (token.address.toLowerCase() === tokenAddress.toLowerCase()) {
            return symbol;
          }
        }
      }

      // If no match found, truncate the address
      return truncateText(tokenAddress);
    } catch (error) {
      console.error("Error getting token symbol:", error);
      return tokenAddress ? truncateText(tokenAddress) : ""; // Fallback to address
    }
  };

  return (
    <main className="flex min-h-[calc(100vh-150px)] flex-col items-center p-4 md:p-8 pt-6 md:pt-8 relative overflow-hidden">
      {/* Retro grid background */}
      <div className="fixed inset-0 bg-[linear-gradient(transparent_1px,_transparent_1px),_linear-gradient(90deg,_transparent_1px,_transparent_1px)] bg-[size:4rem_4rem] [mask-image:radial-gradient(ellipse_80%_50%_at_50%_0%,_#000_70%,_transparent_100%)] opacity-20" />

      {/* Neon glow effects */}
      <div className="fixed inset-0 bg-[radial-gradient(circle_at_50%_50%,_rgba(255,255,0,0.1)_0%,_transparent_50%)]" />

      <div className="z-10 max-w-5xl w-full relative">
        <div className="text-center mb-6 md:mb-8">
          <h1 className="arcade-text text-xl md:text-2xl text-primary-500 relative mb-2">
            <span className="absolute inset-0 blur-sm opacity-50">
              INTENT DETAILS
            </span>
            INTENT DETAILS
          </h1>
          <Link
            href="/"
            className="text-yellow-500 text-xs arcade-text hover:underline"
          >
            ← BACK TO HOME
          </Link>
        </div>

        <div className="arcade-container border-yellow-500 relative group">
          <div className="absolute inset-0 bg-yellow-500/10 blur-sm group-hover:bg-yellow-500/20 transition-all duration-300" />
          <div className="relative p-6">
            {isLoading ? (
              <div className="text-center py-8">
                <p className="arcade-text text-primary-500 animate-pulse">
                  LOADING INTENT DATA...
                </p>
              </div>
            ) : error ? (
              <div className="text-center py-8">
                <p className="arcade-text text-red-500">
                  ERROR: {error.message}
                </p>
                <Link
                  href="/"
                  className="arcade-text text-yellow-500 text-sm mt-4 inline-block hover:underline"
                >
                  RETURN HOME
                </Link>
              </div>
            ) : intent ? (
              <div className="space-y-4">
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div className="space-y-4">
                    {intent.id && (
                      <div>
                        <h3 className="arcade-text text-yellow-500 text-xs mb-1">
                          INTENT ID
                        </h3>
                        <p className="arcade-text text-gray-300">
                          {truncateText(intent.id, 12, 8)}
                          <button
                            onClick={() =>
                              navigator.clipboard.writeText(intent.id)
                            }
                            className="ml-2 text-primary-500 text-xs hover:text-primary-400"
                            title="Copy full ID"
                          >
                            [COPY]
                          </button>
                        </p>
                      </div>
                    )}

                    {intent.status && (
                      <div>
                        <h3 className="arcade-text text-yellow-500 text-xs mb-1">
                          STATUS
                        </h3>
                        <p
                          className={`arcade-text ${
                            intent.status === "pending"
                              ? "text-yellow-300"
                              : intent.status === "fulfilled"
                                ? "text-green-500"
                                : intent.status === "settled"
                                  ? "text-green-500"
                                  : "text-gray-300"
                          }`}
                        >
                          {intent.status.toUpperCase()}
                        </p>
                      </div>
                    )}

                    {intent.source_chain && (
                      <div>
                        <h3 className="arcade-text text-yellow-500 text-xs mb-1">
                          FROM CHAIN
                        </h3>
                        <p className="arcade-text text-gray-300">
                          {getChainName(intent.source_chain)}
                        </p>
                      </div>
                    )}

                    {intent.destination_chain && (
                      <div>
                        <h3 className="arcade-text text-yellow-500 text-xs mb-1">
                          TO CHAIN
                        </h3>
                        <p className="arcade-text text-gray-300">
                          {getChainName(intent.destination_chain)}
                        </p>
                      </div>
                    )}
                  </div>

                  <div className="space-y-4">
                    {intent.token && (
                      <div>
                        <h3 className="arcade-text text-yellow-500 text-xs mb-1">
                          TOKEN
                        </h3>
                        <p className="arcade-text text-gray-300">
                          {getTokenSymbol(intent.token, intent.source_chain)}
                        </p>
                      </div>
                    )}

                    {intent.amount && (
                      <div>
                        <h3 className="arcade-text text-yellow-500 text-xs mb-1">
                          AMOUNT
                        </h3>
                        <p className="arcade-text text-gray-300">
                          {formatTokenAmount(
                            intent.amount,
                            intent.token,
                            intent.source_chain,
                          )}
                        </p>
                      </div>
                    )}

                    {intent.recipient && (
                      <div>
                        <h3 className="arcade-text text-yellow-500 text-xs mb-1">
                          RECIPIENT
                        </h3>
                        <p className="arcade-text text-gray-300">
                          {truncateText(intent.recipient, 10, 8)}
                          <button
                            onClick={() =>
                              navigator.clipboard.writeText(intent.recipient)
                            }
                            className="ml-2 text-primary-500 text-xs hover:text-primary-400"
                            title="Copy full address"
                          >
                            [COPY]
                          </button>
                        </p>
                      </div>
                    )}

                    {intent.intent_fee && (
                      <div>
                        <h3 className="arcade-text text-yellow-500 text-xs mb-1">
                          FEE
                        </h3>
                        <p className="arcade-text text-gray-300">
                          {formatTokenAmount(
                            intent.intent_fee,
                            intent.token,
                            intent.source_chain,
                          )}
                        </p>
                      </div>
                    )}
                  </div>
                </div>

                {/* Fulfillment Details Section */}
                {fulfillment && (
                  <div className="border-t border-gray-700 pt-4 mt-4">
                    <h3 className="arcade-text text-green-500 text-sm mb-3">
                      FULFILLMENT DETAILS
                    </h3>
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                      {fulfillment.fulfiller && (
                        <div>
                          <h3 className="arcade-text text-yellow-500 text-xs mb-1">
                            FULFILLER
                          </h3>
                          <p className="arcade-text text-gray-300">
                            {truncateText(fulfillment.fulfiller, 10, 8)}
                            <button
                              onClick={() =>
                                navigator.clipboard.writeText(
                                  fulfillment.fulfiller,
                                )
                              }
                              className="ml-2 text-primary-500 text-xs hover:text-primary-400"
                              title="Copy full address"
                            >
                              [COPY]
                            </button>
                          </p>
                        </div>
                      )}

                      {fulfillment.tx_hash && (
                        <div>
                          <h3 className="arcade-text text-yellow-500 text-xs mb-1">
                            TRANSACTION
                          </h3>
                          <div className="flex items-center">
                            <p className="arcade-text text-gray-300">
                              {truncateText(fulfillment.tx_hash, 10, 8)}
                            </p>
                            <button
                              onClick={() =>
                                navigator.clipboard.writeText(
                                  fulfillment.tx_hash,
                                )
                              }
                              className="ml-2 text-primary-500 text-xs hover:text-primary-400"
                              title="Copy transaction hash"
                            >
                              [COPY]
                            </button>
                          </div>
                        </div>
                      )}

                      {fulfillment.amount && (
                        <div>
                          <h3 className="arcade-text text-yellow-500 text-xs mb-1">
                            AMOUNT FULFILLED
                          </h3>
                          <p className="arcade-text text-gray-300">
                            {formatTokenAmount(
                              fulfillment.amount,
                              intent.token,
                              intent.source_chain,
                            )}
                          </p>
                        </div>
                      )}

                      {fulfillment.created_at && (
                        <div>
                          <h3 className="arcade-text text-yellow-500 text-xs mb-1">
                            FULFILLMENT TIME
                          </h3>
                          <p className="arcade-text text-gray-300">
                            {formatDate(fulfillment.created_at)}
                          </p>
                        </div>
                      )}

                      {fulfillment.tx_hash &&
                        intent &&
                        intent.destination_chain && (
                          <div className="md:col-span-2 mt-2 text-center">
                            <a
                              href={getExplorerUrl(
                                parseInt(intent.destination_chain),
                                fulfillment.tx_hash,
                              )}
                              target="_blank"
                              rel="noopener noreferrer"
                              className="arcade-btn bg-green-500 text-black hover:bg-green-400 transition-colors duration-200 text-xs px-4 py-1 inline-block"
                            >
                              VIEW TRANSACTION ON{" "}
                              {getChainName(intent.destination_chain)} EXPLORER
                            </a>
                          </div>
                        )}
                    </div>
                  </div>
                )}

                <div className="border-t border-gray-700 pt-4 mt-4">
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    {intent.created_at && (
                      <div>
                        <h3 className="arcade-text text-yellow-500 text-xs mb-1">
                          CREATED AT
                        </h3>
                        <p className="arcade-text text-gray-300">
                          {formatDate(intent.created_at)}
                        </p>
                      </div>
                    )}

                    {intent.updated_at && (
                      <div>
                        <h3 className="arcade-text text-yellow-500 text-xs mb-1">
                          UPDATED AT
                        </h3>
                        <p className="arcade-text text-gray-300">
                          {formatDate(intent.updated_at)}
                        </p>
                      </div>
                    )}
                  </div>
                </div>
              </div>
            ) : (
              <div className="text-center py-8">
                <p className="arcade-text text-red-500">INTENT NOT FOUND</p>
                <Link
                  href="/"
                  className="arcade-text text-yellow-500 text-sm mt-4 inline-block hover:underline"
                >
                  RETURN HOME
                </Link>
              </div>
            )}
          </div>
        </div>
      </div>
    </main>
  );
}
