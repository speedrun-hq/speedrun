import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import IntentList from '../IntentList';
import { apiService } from '@/services/api';

// Mock the API service
jest.mock('@/services/api', () => ({
  apiService: {
    listIntents: jest.fn(),
  },
}));

describe('IntentList', () => {
  const mockIntents = [
    {
      id: '1',
      source_chain: 'base',
      destination_chain: 'arbitrum',
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
      source_chain: 'arbitrum',
      destination_chain: 'base',
      token: 'USDC',
      amount: '250.50',
      recipient: '0x0987654321098765432109876543210987654321',
      intent_fee: '0.02',
      status: 'completed',
      created_at: '2024-03-19T15:30:00Z',
      updated_at: '2024-03-19T15:35:00Z',
    },
  ];

  beforeEach(() => {
    jest.clearAllMocks();
    (apiService.listIntents as jest.Mock).mockResolvedValue(mockIntents);
  });

  it('renders loading state initially', () => {
    render(<IntentList />);
    expect(screen.getByText('LOADING...')).toBeInTheDocument();
  });

  it('renders intents after loading', async () => {
    render(<IntentList />);

    await waitFor(() => {
      expect(screen.queryByText('LOADING...')).not.toBeInTheDocument();
    });

    expect(screen.getByText('RUNS')).toBeInTheDocument();
    expect(screen.getByText('ID: 1')).toBeInTheDocument();
    expect(screen.getByText('ID: 2')).toBeInTheDocument();
    expect(screen.getByText('base → arbitrum')).toBeInTheDocument();
    expect(screen.getByText('arbitrum → base')).toBeInTheDocument();
  });

  it('renders no records message when no intents', async () => {
    (apiService.listIntents as jest.Mock).mockResolvedValue([]);
    render(<IntentList />);

    await waitFor(() => {
      expect(screen.getByText('NO RECORDS FOUND')).toBeInTheDocument();
    });
  });

  it('handles pagination correctly', async () => {
    // Mock first page
    (apiService.listIntents as jest.Mock).mockResolvedValueOnce(mockIntents);
    
    render(<IntentList />);

    await waitFor(() => {
      expect(screen.queryByText('LOADING...')).not.toBeInTheDocument();
    });

    // Mock second page
    const secondPageIntents = [
      {
        id: '3',
        source_chain: 'base',
        destination_chain: 'arbitrum',
        token: 'USDC',
        amount: '75.25',
        recipient: '0xabcdef1234567890abcdef1234567890abcdef12',
        intent_fee: '0.01',
        status: 'failed',
        created_at: '2024-03-18T10:00:00Z',
        updated_at: '2024-03-18T10:00:00Z',
      },
    ];
    (apiService.listIntents as jest.Mock).mockResolvedValueOnce(secondPageIntents);

    // Click next button
    await userEvent.click(screen.getByText('NEXT'));

    await waitFor(() => {
      expect(screen.getByText('ID: 3')).toBeInTheDocument();
    });

    // Verify API calls
    expect(apiService.listIntents).toHaveBeenCalledWith(10, 0);
    expect(apiService.listIntents).toHaveBeenCalledWith(10, 10);
  });

  it('handles API errors', async () => {
    const error = new Error('Failed to fetch intents');
    (apiService.listIntents as jest.Mock).mockRejectedValue(error);

    render(<IntentList />);

    await waitFor(() => {
      expect(screen.getByText(error.message)).toBeInTheDocument();
    });
  });

  it('displays correct status colors', async () => {
    render(<IntentList />);

    await waitFor(() => {
      expect(screen.queryByText('LOADING...')).not.toBeInTheDocument();
    });

    const pendingStatus = screen.getByText('PENDING');
    const completedStatus = screen.getByText('COMPLETED');

    expect(pendingStatus.className).toContain('text-primary-500');
    expect(completedStatus.className).toContain('text-secondary-500');
  });
}); 