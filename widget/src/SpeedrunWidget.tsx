"use client";

import React, { useState, useEffect } from "react";
// Use our standalone widget hook instead of the frontend hook
import { useWidgetIntent } from "./hooks/useWidgetIntent";
// But we can now use the shared components and utils
import { ChainSelector, TokenSelector, FormInput } from "@speedrun/components";
import { getChainId, getChainName, ChainName, TokenSymbol } from "@speedrun/utils";
import { useAccount } from "wagmi";

export interface SpeedrunWidgetProps {
  defaultSourceChain?: ChainName;
  defaultDestinationChain?: ChainName;
  defaultToken?: TokenSymbol;
  defaultAmount?: string;
  defaultRecipient?: string;
  defaultTip?: string;
  onSuccess?: (intentId: string) => void;
  onError?: (error: Error) => void;
  customStyles?: {
    containerClass?: string;
    buttonClass?: string;
    inputClass?: string;
    labelClass?: string;
  };
}

export const SpeedrunWidget: React.FC<SpeedrunWidgetProps> = ({
  defaultSourceChain = "BASE" as ChainName,
  defaultDestinationChain = "ARBITRUM" as ChainName,
  defaultToken = "USDC" as TokenSymbol,
  defaultAmount = "",
  defaultRecipient = "",
  defaultTip = "",
  onSuccess,
  onError,
  customStyles = {},
}) => {
  const { address } = useAccount();
  const [compact, setCompact] = useState(false);

  const {
    formState,
    balance,
    isError,
    isLoading,
    symbol,
    isConnected,
    isValid,
    handleSubmit,
    updateSourceChain,
    updateDestinationChain,
    updateToken,
    updateAmount,
    updateRecipient,
    updateTip,
    resetForm,
  } = useWidgetIntent({
    defaultSourceChain,
    defaultDestinationChain,
    defaultToken,
    defaultAmount,
  });

  // Apply default values on initial render
  useEffect(() => {
    updateSourceChain(defaultSourceChain as any);
    updateDestinationChain(defaultDestinationChain as any);
    updateToken(defaultToken as any);
    if (defaultAmount) updateAmount(defaultAmount);
    if (defaultRecipient) updateRecipient(defaultRecipient);
    if (defaultTip) updateTip(defaultTip);
  }, []);

  // Set default recipient to sender's address when connected and recipient is not set
  useEffect(() => {
    if (isConnected && address && !formState.recipient) {
      updateRecipient(address);
    }
  }, [isConnected, address, formState.recipient, updateRecipient]);

  // Call onSuccess callback when intent is created
  useEffect(() => {
    if (formState.success && formState.intentId && onSuccess) {
      onSuccess(formState.intentId);
    }
  }, [formState.success, formState.intentId, onSuccess]);

  // Call onError callback when error occurs
  useEffect(() => {
    if (formState.error && onError) {
      onError(formState.error);
    }
  }, [formState.error, onError]);

  // Handle chain selection
  const handleSourceChainChange = (chainId: number) => {
    const chainName = getChainName(chainId);
    updateSourceChain(chainName);
  };

  const handleDestinationChainChange = (chainId: number) => {
    const chainName = getChainName(chainId);
    updateDestinationChain(chainName);
  };

  const containerClass = customStyles.containerClass || "border border-gray-200 rounded-lg p-4 max-w-md";
  const buttonClass = customStyles.buttonClass || "w-full bg-blue-600 text-white py-2 px-4 rounded hover:bg-blue-700 disabled:opacity-50";
  const inputClass = customStyles.inputClass || "";
  const labelClass = customStyles.labelClass || "block text-sm font-medium text-gray-700 mb-1";

  return (
    <div className={containerClass}>
      <div className="flex justify-between items-center mb-4">
        <h3 className="font-bold">Transfer Tokens</h3>
        <button 
          onClick={() => setCompact(!compact)} 
          className="text-sm text-gray-500"
        >
          {compact ? "Expand" : "Compact"}
        </button>
      </div>

      <form onSubmit={handleSubmit} className="space-y-4">
        <div className={compact ? "grid grid-cols-2 gap-2" : "space-y-4"}>
          <div>
            <label className={labelClass}>From</label>
            <ChainSelector
              value={getChainId(formState.sourceChain)}
              onChange={handleSourceChainChange}
              label="Source Chain"
              disabled={formState.isSubmitting}
              selectorType="from"
            />
          </div>

          <div>
            <label className={labelClass}>To</label>
            <ChainSelector
              value={getChainId(formState.destinationChain)}
              onChange={handleDestinationChainChange}
              label="Destination Chain"
              disabled={formState.isSubmitting}
              selectorType="to"
            />
          </div>

          {!compact && (
            <div>
              <label className={labelClass}>Token</label>
              <TokenSelector
                value={formState.selectedToken}
                onChange={updateToken}
                sourceChain={formState.sourceChain}
                destinationChain={formState.destinationChain}
              />
            </div>
          )}

          <div>
            <div className="flex justify-between items-center">
              <label className={labelClass}>Amount</label>
              {!compact && (
                <span className="text-xs text-gray-500">
                  Available: {isLoading ? "Loading..." : isConnected && balance ? `${balance} ${symbol}` : "0.00"}
                </span>
              )}
            </div>
            <FormInput
              type="number"
              value={formState.amount}
              onChange={updateAmount}
              placeholder="0.00"
              disabled={formState.isSubmitting}
              className={inputClass}
            />
          </div>

          {!compact && (
            <div>
              <label className={labelClass}>Recipient</label>
              <FormInput
                value={formState.recipient}
                onChange={updateRecipient}
                placeholder="0x..."
                disabled={formState.isSubmitting}
                className={inputClass}
              />
            </div>
          )}

          {!compact && (
            <div>
              <label className={labelClass}>Fee ({symbol})</label>
              <FormInput
                type="number"
                value={formState.tip}
                onChange={updateTip}
                placeholder="0.1"
                disabled={formState.isSubmitting}
                min="0.01"
                step="0.01"
                className={inputClass}
              />
            </div>
          )}
        </div>

        {formState.error && (
          <div className="text-red-500 text-sm p-2 rounded bg-red-50">
            {formState.error.message}
          </div>
        )}

        {formState.success && formState.intentId && (
          <div className="text-green-500 text-sm p-2 rounded bg-green-50">
            Transfer initiated! Intent ID: {formState.intentId.slice(0, 8)}...
          </div>
        )}

        <button
          type="submit"
          disabled={
            !isConnected ||
            (!isValid && !formState.success) ||
            formState.isSubmitting
          }
          className={buttonClass}
        >
          {!isConnected
            ? "Connect Wallet"
            : formState.isSubmitting
              ? "Processing..."
              : formState.success
                ? "New Transfer"
                : "Transfer"}
        </button>
      </form>
    </div>
  );
}

export default SpeedrunWidget; 