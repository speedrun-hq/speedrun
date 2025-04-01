import { Intent, Fulfillment, ListIntentsResponse } from '@/types';

const MOCK_INTENTS: Intent[] = [
  {
    id: '1',
    source_chain: 'base',
    destination_chain: 'polygon',
    token: 'USDC',
    amount: '100.00',
    recipient: '0x1234567890123456789012345678901234567890',
    intent_fee: '0.01',
    status: 'pending',
    created_at: '2024-03-20T10:00:00Z',
    updated_at: '2024-03-20T10:00:00Z',
  },
  {
    id: '2',
    source_chain: 'polygon',
    destination_chain: 'base',
    token: 'USDC',
    amount: '250.50',
    recipient: '0x0987654321098765432109876543210987654321',
    intent_fee: '0.02',
    status: 'completed',
    created_at: '2024-03-19T15:30:00Z',
    updated_at: '2024-03-19T15:35:00Z',
  },
  {
    id: '3',
    source_chain: 'base',
    destination_chain: 'polygon',
    token: 'USDC',
    amount: '75.25',
    recipient: '0xabcdef1234567890abcdef1234567890abcdef12',
    intent_fee: '0.01',
    status: 'failed',
    created_at: new Date(Date.now() - 10800000).toISOString(), // 3 hours ago
    updated_at: new Date(Date.now() - 10800000).toISOString(), // 3 hours ago
  },
];

const MOCK_FULFILLMENTS: Fulfillment[] = [
  {
    id: '1',
    intent_id: '1',
    fulfiller: '0x1234567890123456789012345678901234567890',
    amount: '100.00',
    status: 'pending',
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  },
  {
    id: '2',
    intent_id: '2',
    fulfiller: '0x0987654321098765432109876543210987654321',
    amount: '250.50',
    status: 'completed',
    created_at: new Date(Date.now() - 3600000).toISOString(),
    updated_at: new Date(Date.now() - 3600000).toISOString(),
  },
];

export const getMockIntents = (page: number, limit: number): ListIntentsResponse => {
  const start = (page - 1) * limit;
  const end = start + limit;
  const paginatedIntents = MOCK_INTENTS.slice(start, end);
  
  return {
    intents: paginatedIntents,
    total: MOCK_INTENTS.length,
    page,
    limit,
  };
};

export const getMockIntent = (id: string) => {
  return MOCK_INTENTS.find(intent => intent.id === id) || null;
};

export const getMockFulfillment = (id: string) => {
  return MOCK_FULFILLMENTS.find(fulfillment => fulfillment.id === id) || null;
}; 