import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import CreateNewIntent from '../CreateNewIntent';
import { useIntentForm } from '@/hooks/useIntentForm';

// Mock the useIntentForm hook
jest.mock('@/hooks/useIntentForm');

// Mock the dynamic import wrapper
jest.mock('../CreateNewIntentWrapper', () => ({
  CreateNewIntentWrapper: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

describe('CreateNewIntent', () => {
  const mockUseIntentForm = {
    formState: {
      sourceChain: 'BASE',
      destinationChain: 'ARBITRUM',
      selectedToken: 'USDC',
      amount: '',
      recipient: '',
      isSubmitting: false,
      error: null,
      success: false,
    },
    balance: '1000.00',
    isError: false,
    isLoading: false,
    symbol: 'USDC',
    isConnected: true,
    isValid: false,
    handleSubmit: jest.fn(),
    updateSourceChain: jest.fn(),
    updateDestinationChain: jest.fn(),
    updateToken: jest.fn(),
    updateAmount: jest.fn(),
    updateRecipient: jest.fn(),
  };

  beforeEach(() => {
    (useIntentForm as jest.Mock).mockReturnValue(mockUseIntentForm);
  });

  afterEach(() => {
    jest.clearAllMocks();
  });

  it('renders the form with all required fields', () => {
    render(<CreateNewIntent />);
    
    expect(screen.getByText('NEW TRANSFER')).toBeInTheDocument();
    expect(screen.getByText('SOURCE CHAIN')).toBeInTheDocument();
    expect(screen.getByText('DESTINATION CHAIN')).toBeInTheDocument();
    expect(screen.getByText('SELECT TOKEN')).toBeInTheDocument();
    expect(screen.getByText('RECIPIENT ADDRESS')).toBeInTheDocument();
    expect(screen.getByText('AMOUNT (USDC)')).toBeInTheDocument();
    expect(screen.getByText('Available: 1000.00 USDC')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'START RUN' })).toBeInTheDocument();
  });

  it('shows loading state when submitting', () => {
    (useIntentForm as jest.Mock).mockReturnValue({
      ...mockUseIntentForm,
      formState: {
        ...mockUseIntentForm.formState,
        isSubmitting: true,
      },
    });

    render(<CreateNewIntent />);
    expect(screen.getByRole('button', { name: 'CREATING RUN...' })).toBeInTheDocument();
  });

  it('shows error message when there is an error', () => {
    const errorMessage = 'Test error message';
    (useIntentForm as jest.Mock).mockReturnValue({
      ...mockUseIntentForm,
      formState: {
        ...mockUseIntentForm.formState,
        error: new Error(errorMessage),
      },
    });

    render(<CreateNewIntent />);
    expect(screen.getByText(errorMessage)).toBeInTheDocument();
  });

  it('shows success message when run is created', () => {
    (useIntentForm as jest.Mock).mockReturnValue({
      ...mockUseIntentForm,
      formState: {
        ...mockUseIntentForm.formState,
        success: true,
      },
    });

    render(<CreateNewIntent />);
    expect(screen.getByText('RUN CREATED SUCCESSFULLY!')).toBeInTheDocument();
  });

  it('disables form fields when submitting', () => {
    (useIntentForm as jest.Mock).mockReturnValue({
      ...mockUseIntentForm,
      formState: {
        ...mockUseIntentForm.formState,
        isSubmitting: true,
      },
    });

    render(<CreateNewIntent />);
    
    const amountInput = screen.getByPlaceholderText('0.00');
    const recipientInput = screen.getByPlaceholderText('0x...');
    
    expect(amountInput).toBeDisabled();
    expect(recipientInput).toBeDisabled();
  });

  it('shows disconnected state when wallet is not connected', () => {
    (useIntentForm as jest.Mock).mockReturnValue({
      ...mockUseIntentForm,
      isConnected: false,
    });

    render(<CreateNewIntent />);
    expect(screen.getByText('Please connect your wallet to continue')).toBeInTheDocument();
  });

  it('handles form input changes', async () => {
    render(<CreateNewIntent />);
    
    const amountInput = screen.getByPlaceholderText('0.00');
    const recipientInput = screen.getByPlaceholderText('0x...');
    
    await userEvent.type(amountInput, '100.00');
    await userEvent.type(recipientInput, '0x123');
    
    expect(mockUseIntentForm.updateAmount).toHaveBeenCalled();
    expect(mockUseIntentForm.updateRecipient).toHaveBeenCalled();
  });

  it('handles form submission', async () => {
    const handleSubmit = jest.fn(e => e.preventDefault());
    (useIntentForm as jest.Mock).mockReturnValue({
      ...mockUseIntentForm,
      handleSubmit,
      isValid: true,
    });

    render(<CreateNewIntent />);
    
    const form = screen.getByRole('form');
    await userEvent.click(screen.getByRole('button', { name: 'START RUN' }));
    fireEvent.submit(form);
    
    expect(handleSubmit).toHaveBeenCalled();
  });
}); 