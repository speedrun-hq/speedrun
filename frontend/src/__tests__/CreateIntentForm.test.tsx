import { screen, fireEvent, act } from '@testing-library/react';
import { render } from '../test-utils';
import CreateIntentForm from '@/components/CreateIntentForm';
import { apiService } from '@/services/api';

// Mock the apiService
jest.mock('@/services/api', () => ({
  apiService: {
    createIntent: jest.fn(),
  },
}));

describe('CreateIntentForm', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('renders the form with all required fields', () => {
    render(<CreateIntentForm />);
    
    expect(screen.getByLabelText(/source chain/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/destination chain/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/recipient address/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/amount/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/intent fee/i)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /create intent/i })).toBeInTheDocument();
  });

  it('submits the form with correct data', async () => {
    const mockCreateIntent = jest.fn().mockResolvedValue({});
    (apiService.createIntent as jest.Mock) = mockCreateIntent;

    render(<CreateIntentForm />);

    // Fill in the form
    await act(async () => {
      fireEvent.change(screen.getByLabelText(/source chain/i), {
        target: { value: 'ethereum' },
      });
      fireEvent.change(screen.getByLabelText(/destination chain/i), {
        target: { value: 'base' },
      });
      fireEvent.change(screen.getByLabelText(/recipient address/i), {
        target: { value: '0x1234567890abcdef' },
      });
      fireEvent.change(screen.getByLabelText(/amount/i), {
        target: { value: '1.0' },
      });
      fireEvent.change(screen.getByLabelText(/intent fee/i), {
        target: { value: '0.1' },
      });
    });

    // Submit the form
    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: /create intent/i }));
    });

    // Verify API call
    expect(mockCreateIntent).toHaveBeenCalledWith({
      source_chain: 'ethereum',
      destination_chain: 'base',
      recipient: '0x1234567890abcdef',
      token: 'USDC',
      amount: '1.0',
      intent_fee: '0.1',
    });
  });

  it('displays error message when API call fails', async () => {
    const mockError = new Error('API Error');
    (apiService.createIntent as jest.Mock).mockRejectedValue(mockError);

    render(<CreateIntentForm />);

    // Fill in the form
    await act(async () => {
      fireEvent.change(screen.getByLabelText(/source chain/i), {
        target: { value: 'ethereum' },
      });
      fireEvent.change(screen.getByLabelText(/destination chain/i), {
        target: { value: 'base' },
      });
      fireEvent.change(screen.getByLabelText(/recipient address/i), {
        target: { value: '0x1234567890abcdef' },
      });
      fireEvent.change(screen.getByLabelText(/amount/i), {
        target: { value: '1.0' },
      });
      fireEvent.change(screen.getByLabelText(/intent fee/i), {
        target: { value: '0.1' },
      });
    });

    // Submit the form
    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: /create intent/i }));
    });

    // Verify error message
    expect(await screen.findByText(/error/i)).toBeInTheDocument();
  });
}); 