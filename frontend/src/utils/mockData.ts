import { Intent, Fulfillment, ListIntentsResponse } from "@/types";

const MOCK_INTENTS: Intent[] = [
  {
    id: "1",
    source_chain: "base",
    destination_chain: "arbitrum",
    token: "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913",
    amount: "100",
    recipient: "0x1234567890123456789012345678901234567890",
    intent_fee: "0.1",
    status: "pending",
    created_at: "2024-04-05T12:00:00Z",
    updated_at: "2024-04-05T12:00:00Z",
    approvalHash: null,
    intentHash: null,
  },
  {
    id: "2",
    source_chain: "arbitrum",
    destination_chain: "base",
    token: "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913",
    amount: "250.50",
    recipient: "0x0987654321098765432109876543210987654321",
    intent_fee: "0.02",
    status: "completed",
    created_at: "2024-03-19T15:30:00Z",
    updated_at: "2024-03-19T15:35:00Z",
    approvalHash: null,
    intentHash: null,
  },
  {
    id: "3",
    source_chain: "base",
    destination_chain: "arbitrum",
    token: "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913",
    amount: "75.25",
    recipient: "0xabcdef1234567890abcdef1234567890abcdef12",
    intent_fee: "0.01",
    status: "failed",
    created_at: new Date(Date.now() - 10800000).toISOString(), // 3 hours ago
    updated_at: new Date(Date.now() - 10800000).toISOString(), // 3 hours ago
    approvalHash: null,
    intentHash: null,
  },
];

const MOCK_FULFILLMENTS: Fulfillment[] = [
  {
    id: "1",
    intent_id: "1",
    fulfiller: "0x1234567890123456789012345678901234567890",
    amount: "100.00",
    status: "pending",
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
    tx_hash: "0x1234567890123456789012345678901234567890",
  },
  {
    id: "2",
    intent_id: "2",
    fulfiller: "0x0987654321098765432109876543210987654321",
    amount: "250.50",
    status: "completed",
    created_at: new Date(Date.now() - 3600000).toISOString(),
    updated_at: new Date(Date.now() - 3600000).toISOString(),
    tx_hash: "0x0987654321098765432109876543210987654321",
  },
];

export const getMockIntents = (
  limit: number,
  offset: number,
  status?: string,
): Intent[] => {
  let filteredIntents = MOCK_INTENTS;

  // Filter by status if provided
  if (status) {
    filteredIntents = filteredIntents.filter(
      (intent) => intent.status === status,
    );
  }

  // Apply pagination
  const start = offset;
  const end = start + limit;
  return filteredIntents.slice(start, end);
};

export const getMockIntent = (id: string) => {
  return MOCK_INTENTS.find((intent) => intent.id === id) || null;
};

export const getMockFulfillment = (id: string) => {
  return MOCK_FULFILLMENTS.find((fulfillment) => fulfillment.id === id) || null;
};
