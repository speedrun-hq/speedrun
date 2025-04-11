import { TOKENS } from "@/constants/tokens";

/**
 * Truncate long strings (like addresses and IDs)
 * @param text Text to truncate
 * @param startLength Number of characters to show at the start
 * @param endLength Number of characters to show at the end
 * @returns Truncated text with ellipsis
 */
export const truncateText = (
  text: string,
  startLength = 8,
  endLength = 6,
): string => {
  if (!text) return ""; // Handle undefined or empty strings
  if (text.length <= startLength + endLength + 3) return text;
  return `${text.substring(0, startLength)}...${text.substring(text.length - endLength)}`;
};

/**
 * Format token amount with correct decimals
 * @param amount Token amount as string
 * @param tokenAddress Token contract address
 * @param sourceChain Chain ID as string
 * @returns Formatted amount with token symbol if available
 */
export const formatTokenAmount = (
  amount: string,
  tokenAddress: string,
  sourceChain: string,
): string => {
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

/**
 * Get token symbol from address
 * @param tokenAddress Token contract address
 * @param sourceChain Chain ID as string
 * @returns Token symbol or truncated address if symbol not found
 */
export const getTokenSymbol = (
  tokenAddress: string,
  sourceChain: string,
): string => {
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
