export interface Intent {
  id: string;
  source_chain: string;
  destination_chain: string;
  token: string;
  amount: string;
  recipient: string;
  intent_fee: string;
  status: string;
  created_at: string;
  updated_at: string;
}

export interface Fulfillment {
  id: string;
  intent_id: string;
  fulfiller: string;
  amount: string;
  status: string;
  created_at: string;
  updated_at: string;
}

export interface CreateIntentRequest {
  source_chain: string;
  destination_chain: string;
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
  total: number;
  page: number;
  limit: number;
} 