import React, { useState } from "react";
import { SpeedrunWidget } from "@speedrun/widget";

/**
 * Example DeFi application that integrates the Speedrun Widget
 * This shows how a DEX or lending platform might incorporate the widget
 */
export default function ExampleDeFiApp() {
  const [notification, setNotification] = useState<string | null>(null);
  
  // Custom styling to match the app's design
  const customStyles = {
    containerClass: "border border-blue-200 rounded-xl p-5 bg-blue-50 shadow-sm",
    buttonClass: "w-full bg-blue-600 text-white py-3 rounded-lg font-medium hover:bg-blue-700 transition-colors",
    inputClass: "border rounded p-2 w-full focus:border-blue-500 focus:ring-1 focus:ring-blue-500",
    labelClass: "text-blue-800 font-medium"
  };

  // Handle successful intent creation
  const handleSuccess = (intentId: string) => {
    setNotification(`🎉 Transfer initiated! Your intent ID: ${intentId}`);
    
    // In a real app, you might:
    // - Update transaction history
    // - Refresh balances
    // - Show more detailed status updates
    
    // Clear notification after 5 seconds
    setTimeout(() => {
      setNotification(null);
    }, 5000);
  };

  // Handle errors
  const handleError = (error: Error) => {
    setNotification(`❌ Error: ${error.message}`);
    
    // Clear notification after 5 seconds
    setTimeout(() => {
      setNotification(null);
    }, 5000);
  };

  return (
    <div className="max-w-6xl mx-auto p-6">
      <header className="mb-10">
        <h1 className="text-3xl font-bold text-blue-900">DeFi Exchange</h1>
        <p className="text-gray-600">Your one-stop shop for decentralized finance</p>
      </header>

      {notification && (
        <div className="mb-6 p-4 rounded-lg bg-blue-100 border border-blue-200 text-blue-800">
          {notification}
        </div>
      )}

      <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
        <div className="col-span-2 border border-gray-200 rounded-xl p-6 bg-white shadow-sm">
          <h2 className="text-2xl font-semibold mb-6 text-blue-900">Swap Tokens</h2>
          
          {/* This would be your DEX swap interface */}
          <div className="mb-8 p-4 border border-gray-200 rounded-lg">
            <div className="text-center text-gray-500 py-10">
              [Your swap interface would go here]
            </div>
          </div>
          
          <div className="border-t border-gray-200 pt-6 mt-6">
            <h3 className="text-lg font-medium text-blue-900 mb-4">
              Need tokens on another chain?
            </h3>
            <p className="text-gray-600 mb-4">
              Use our integrated cross-chain transfer to quickly move tokens between networks
            </p>
          </div>
        </div>

        {/* Speedrun Widget Integration */}
        <div>
          <h2 className="text-2xl font-semibold mb-4 text-blue-900">Cross-Chain Transfer</h2>
          <SpeedrunWidget
            defaultSourceChain="ARBITRUM"
            defaultDestinationChain="BASE"
            defaultToken="USDC"
            onSuccess={handleSuccess}
            onError={handleError}
            customStyles={customStyles}
          />
          
          <div className="mt-4 text-sm text-gray-500">
            <p>Powered by <a href="https://speedrun.exchange" className="text-blue-600 hover:underline">Speedrun</a></p>
          </div>
        </div>
      </div>
    </div>
  );
} 