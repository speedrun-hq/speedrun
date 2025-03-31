export type Chain = 'ethereum' | 'base' | 'zetachain';
export type Token = 'USDC';
export type Status = 'pending' | 'completed' | 'failed' | 'cancelled';

export interface Intent {
  id: string;
  source_chain: Chain;
  source_address: string;
  target_chain: Chain;
  target_address: string;
  amount: string;
  token: Token;
  status: Status;
  created_at: string;
  updated_at: string;
}

export interface Fulfillment {
  id: string;
  intent_id: string;
  fulfiller: string;
  amount: string;
  token: Token;
  status: Status;
  created_at: string;
  updated_at: string;
}

export interface CreateIntentRequest {
  source_chain: Chain;
  source_address: string;
  target_chain: Chain;
  target_address: string;
  amount: string;
}

export interface CreateFulfillmentRequest {
  intent_id: string;
  fulfiller: string;
  amount: string;
} 