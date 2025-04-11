export interface Intent {
  id: string;
  source_chain: string;
  destination_chain: string;
  token: string;
  amount: string;
  recipient: string;
  sender: string;
  intent_fee: string;
  status: string;
  created_at: string;
  updated_at: string;
  approvalHash: string | null;
  intentHash: string | null;
}

export interface Fulfillment {
  id: string;
  intent_id: string;
  fulfiller: string;
  amount: string;
  status: string;
  created_at: string;
  updated_at: string;
  tx_hash: string;
}

export interface CreateIntentRequest {
  source_chain: number;
  destination_chain: number;
  token: string;
  amount: string;
  recipient: string;
  intent_fee: string;
}

export interface CreateFulfillmentRequest {
  intent_id: string;
  fulfiller: string;
  amount: string;
}

export interface CreateIntentResponse {
  message: string;
  intent: Intent;
}

export interface ListIntentsResponse {
  intents: Intent[];
}

// Chain types
export type ChainName =
  | "BASE"
  | "ARBITRUM"
  | "ETHEREUM"
  | "BSC"
  | "POLYGON"
  | "AVALANCHE"
  | "ZETACHAIN";

// Chain mapping
export const CHAIN_ID_TO_NAME: Record<number, ChainName> = {
  8453: "BASE", // Base
  42161: "ARBITRUM", // Arbitrum
  1: "ETHEREUM", // Ethereum
  56: "BSC", // BNB Chain
  137: "POLYGON", // Polygon
  43114: "AVALANCHE", // Avalanche
  7000: "ZETACHAIN", // ZetaChain
};

export const CHAIN_NAME_TO_ID: Record<ChainName, number> = {
  BASE: 8453,
  ARBITRUM: 42161,
  ETHEREUM: 1,
  BSC: 56,
  POLYGON: 137,
  AVALANCHE: 43114,
  ZETACHAIN: 7000,
};

export interface Runner {
  address: string;
  score: string;
  total_transfers: number;
  total_volume: string;
  average_time: string;
  fastest_time: string;
  last_transfer: string;
}

export interface LeaderboardResponse {
  chain_id: number;
  runners: Runner[];
}

// Pagination types
export interface PaginationParams {
  page: number;
  page_size: number;
}

export interface PaginatedResponse<T> {
  data: T[];
  page: number;
  page_size: number;
  total_count: number;
  total_pages: number;
}

export interface PaginatedIntentsResponse extends PaginatedResponse<Intent> {}
export interface PaginatedFulfillmentsResponse
  extends PaginatedResponse<Fulfillment> {}
