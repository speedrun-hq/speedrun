// This file is now just a re-export of our centralized chain configuration
// It's kept for backward compatibility

import {
  ChainName,
  getChainId,
  getChainName,
  isValidChainId,
  getChainRpcUrl,
  getExplorerUrl
} from "@/config/chains";

export type { ChainName };
export { getChainId, getChainName, isValidChainId, getChainRpcUrl, getExplorerUrl };
