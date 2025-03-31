import { Intent } from '../types';

const MOCK_INTENTS: Intent[] = [
  {
    id: '123e4567-e89b-12d3-a456-426614174000',
    source_chain: 'ethereum',
    source_address: '0x742d35Cc6634C0532925a3b844Bc454e4438f44e',
    target_chain: 'base',
    target_address: '0x742d35Cc6634C0532925a3b844Bc454e4438f44f',
    amount: '1000.00',
    token: 'USDC',
    status: 'pending',
    created_at: '2024-03-31T12:00:00Z',
    updated_at: '2024-03-31T12:00:00Z'
  },
  {
    id: '123e4567-e89b-12d3-a456-426614174001',
    source_chain: 'base',
    source_address: '0x742d35Cc6634C0532925a3b844Bc454e4438f44f',
    target_chain: 'zetachain',
    target_address: '0x742d35Cc6634C0532925a3b844Bc454e4438f44g',
    amount: '2500.50',
    token: 'USDC',
    status: 'completed',
    created_at: '2024-03-31T11:30:00Z',
    updated_at: '2024-03-31T11:35:00Z'
  },
  {
    id: '123e4567-e89b-12d3-a456-426614174002',
    source_chain: 'zetachain',
    source_address: '0x742d35Cc6634C0532925a3b844Bc454e4438f44g',
    target_chain: 'ethereum',
    target_address: '0x742d35Cc6634C0532925a3b844Bc454e4438f44e',
    amount: '500.75',
    token: 'USDC',
    status: 'failed',
    created_at: '2024-03-31T11:00:00Z',
    updated_at: '2024-03-31T11:05:00Z'
  },
  {
    id: '123e4567-e89b-12d3-a456-426614174003',
    source_chain: 'ethereum',
    source_address: '0x742d35Cc6634C0532925a3b844Bc454e4438f44e',
    target_chain: 'base',
    target_address: '0x742d35Cc6634C0532925a3b844Bc454e4438f44f',
    amount: '750.25',
    token: 'USDC',
    status: 'cancelled',
    created_at: '2024-03-31T10:30:00Z',
    updated_at: '2024-03-31T10:35:00Z'
  }
];

export const getMockIntents = (page: number = 1, limit: number = 10): { intents: Intent[]; total: number } => {
  const start = (page - 1) * limit;
  const end = start + limit;
  return {
    intents: MOCK_INTENTS.slice(start, end),
    total: MOCK_INTENTS.length
  };
};

export const getMockIntent = (id: string): Intent | undefined => {
  return MOCK_INTENTS.find(intent => intent.id === id);
}; 