import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { SpeedrunWidget } from '../SpeedrunWidget';
import { useWidgetIntent } from '../hooks/useWidgetIntent';

// Mock the useWidgetIntent hook
jest.mock('../hooks/useWidgetIntent');

// Mock the useAccount hook from wagmi
jest.mock('wagmi', () => ({
  useAccount: () => ({
    address: null,
    isConnected: false
  })
}));

describe('SpeedrunWidget', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    
    // Default mock implementation
    (useWidgetIntent as jest.Mock).mockReturnValue({
      formState: {
        sourceChain: 'BASE',
        destinationChain: 'ARBITRUM',
        selectedToken: 'USDC',
        amount: '',
        recipient: '',
        tip: '0.2',
        isSubmitting: false,
        error: null,
        success: false,
        intentId: null,
        fulfillmentTxHash: null
      },
      balance: '100',
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
      updateTip: jest.fn(),
      resetForm: jest.fn()
    });
  });
  
  it('renders correctly with default props', () => {
    render(<SpeedrunWidget />);
    
    expect(screen.getByText('Transfer Tokens')).toBeInTheDocument();
    expect(screen.getByText('From')).toBeInTheDocument();
    expect(screen.getByText('To')).toBeInTheDocument();
    expect(screen.getByText('Token')).toBeInTheDocument();
    expect(screen.getByText('Amount')).toBeInTheDocument();
    expect(screen.getByText('Recipient')).toBeInTheDocument();
    expect(screen.getByText('Fee (USDC)')).toBeInTheDocument();
    expect(screen.getByText('Transfer')).toBeInTheDocument();
  });
  
  it('applies default values from props', async () => {
    const updateSourceChain = jest.fn();
    const updateDestinationChain = jest.fn();
    const updateToken = jest.fn();
    const updateAmount = jest.fn();
    const updateRecipient = jest.fn();
    const updateTip = jest.fn();
    
    (useWidgetIntent as jest.Mock).mockReturnValue({
      formState: {
        sourceChain: 'BASE',
        destinationChain: 'ARBITRUM',
        selectedToken: 'USDC',
        amount: '',
        recipient: '',
        tip: '0.2',
      },
      updateSourceChain,
      updateDestinationChain,
      updateToken,
      updateAmount,
      updateRecipient,
      updateTip,
      isConnected: true,
      isValid: false,
      handleSubmit: jest.fn()
    });
    
    render(
      <SpeedrunWidget
        defaultSourceChain="ETHEREUM"
        defaultDestinationChain="POLYGON"
        defaultToken="ETH"
        defaultAmount="5"
        defaultRecipient="0x0987654321098765432109876543210987654321"
        defaultTip="0.3"
      />
    );
    
    // Verify hook methods were called with default props
    await waitFor(() => {
      expect(updateSourceChain).toHaveBeenCalledWith('ETHEREUM');
      expect(updateDestinationChain).toHaveBeenCalledWith('POLYGON');
      expect(updateToken).toHaveBeenCalledWith('ETH');
      expect(updateAmount).toHaveBeenCalledWith('5');
      expect(updateRecipient).toHaveBeenCalledWith('0x0987654321098765432109876543210987654321');
      expect(updateTip).toHaveBeenCalledWith('0.3');
    });
  });
  
  it('handles source chain change correctly', () => {
    const updateSourceChain = jest.fn();
    
    (useWidgetIntent as jest.Mock).mockReturnValue({
      formState: {
        sourceChain: 'BASE',
        destinationChain: 'ARBITRUM',
        selectedToken: 'USDC',
        amount: '',
        recipient: '',
        tip: '0.2',
      },
      updateSourceChain,
      updateDestinationChain: jest.fn(),
      updateToken: jest.fn(),
      isConnected: true,
      isValid: false,
      handleSubmit: jest.fn()
    });
    
    render(<SpeedrunWidget />);
    
    // Find chain selector and change value
    const sourceChainSelect = screen.getByTestId('chain-select-Source Chain');
    fireEvent.change(sourceChainSelect, { target: { value: '1' } });
    
    // Verify hook method was called with the right chain name
    expect(updateSourceChain).toHaveBeenCalledWith('ETHEREUM');
  });
  
  it('handles destination chain change correctly', () => {
    const updateDestinationChain = jest.fn();
    
    (useWidgetIntent as jest.Mock).mockReturnValue({
      formState: {
        sourceChain: 'BASE',
        destinationChain: 'ARBITRUM',
        selectedToken: 'USDC',
        amount: '',
        recipient: '',
        tip: '0.2',
      },
      updateSourceChain: jest.fn(),
      updateDestinationChain,
      updateToken: jest.fn(),
      isConnected: true,
      isValid: false,
      handleSubmit: jest.fn()
    });
    
    render(<SpeedrunWidget />);
    
    // Find chain selector and change value
    const destChainSelect = screen.getByTestId('chain-select-Destination Chain');
    fireEvent.change(destChainSelect, { target: { value: '1' } });
    
    // Verify hook method was called with the right chain name
    expect(updateDestinationChain).toHaveBeenCalledWith('ETHEREUM');
  });
  
  it('handles form submission correctly', () => {
    const handleSubmit = jest.fn(e => e.preventDefault());
    
    (useWidgetIntent as jest.Mock).mockReturnValue({
      formState: {
        sourceChain: 'BASE',
        destinationChain: 'ARBITRUM',
        selectedToken: 'USDC',
        amount: '10',
        recipient: '0x1234567890123456789012345678901234567890',
        tip: '0.2',
        isSubmitting: false,
        error: null,
        success: false,
        intentId: null
      },
      balance: '100',
      isError: false,
      isLoading: false,
      symbol: 'USDC',
      isConnected: true,
      isValid: true,
      handleSubmit,
      updateSourceChain: jest.fn(),
      updateDestinationChain: jest.fn(),
      updateToken: jest.fn(),
      updateAmount: jest.fn(),
      updateRecipient: jest.fn(),
      updateTip: jest.fn()
    });
    
    render(<SpeedrunWidget />);
    
    // Submit form
    const submitButton = screen.getByText('Transfer');
    fireEvent.click(submitButton);
    
    // Verify handleSubmit was called
    expect(handleSubmit).toHaveBeenCalled();
  });
  
  it('shows error messages when they occur', () => {
    (useWidgetIntent as jest.Mock).mockReturnValue({
      formState: {
        sourceChain: 'BASE',
        destinationChain: 'ARBITRUM',
        selectedToken: 'USDC',
        amount: '10',
        recipient: '0x1234567890123456789012345678901234567890',
        tip: '0.2',
        isSubmitting: false,
        error: new Error('Test error message'),
        success: false,
        intentId: null
      },
      balance: '100',
      isError: true,
      isLoading: false,
      symbol: 'USDC',
      isConnected: true,
      isValid: true,
      handleSubmit: jest.fn(),
      updateSourceChain: jest.fn(),
      updateDestinationChain: jest.fn(),
      updateToken: jest.fn(),
      updateAmount: jest.fn(),
      updateRecipient: jest.fn(),
      updateTip: jest.fn()
    });
    
    render(<SpeedrunWidget />);
    
    // Verify error message is displayed
    expect(screen.getByText('Test error message')).toBeInTheDocument();
  });
  
  it('shows success message after successful submission', () => {
    (useWidgetIntent as jest.Mock).mockReturnValue({
      formState: {
        sourceChain: 'BASE',
        destinationChain: 'ARBITRUM',
        selectedToken: 'USDC',
        amount: '',
        recipient: '',
        tip: '0.2',
        isSubmitting: false,
        error: null,
        success: true,
        intentId: 'test_intent_id'
      },
      balance: '100',
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
      updateTip: jest.fn(),
      resetForm: jest.fn()
    });
    
    render(<SpeedrunWidget />);
    
    // Verify success message with intent ID is displayed
    expect(screen.getByText(/Transfer initiated! Intent ID: test_int/)).toBeInTheDocument();
  });
  
  it('toggles compact mode correctly', () => {
    (useWidgetIntent as jest.Mock).mockReturnValue({
      formState: {
        sourceChain: 'BASE',
        destinationChain: 'ARBITRUM',
        selectedToken: 'USDC',
        amount: '',
        recipient: '',
        tip: '0.2',
        isSubmitting: false,
        error: null,
        success: false,
        intentId: null
      },
      balance: '100',
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
      updateTip: jest.fn()
    });
    
    render(<SpeedrunWidget />);
    
    // Check initial state - should be expanded
    expect(screen.getByText('Compact')).toBeInTheDocument();
    expect(screen.getByText('Token')).toBeInTheDocument();
    expect(screen.getByText('Recipient')).toBeInTheDocument();
    
    // Toggle to compact mode
    fireEvent.click(screen.getByText('Compact'));
    
    // Now should be in compact mode
    expect(screen.getByText('Expand')).toBeInTheDocument();
    
    // Recipient label should no longer be visible in compact mode
    expect(screen.queryByText('Recipient')).not.toBeInTheDocument();
    
    // Toggle back to expanded mode
    fireEvent.click(screen.getByText('Expand'));
    
    // Should be back to expanded mode
    expect(screen.getByText('Compact')).toBeInTheDocument();
    expect(screen.getByText('Recipient')).toBeInTheDocument();
  });
  
  it('calls onSuccess callback when intent is created', async () => {
    const onSuccess = jest.fn();
    
    // First render without success
    const { rerender } = render(<SpeedrunWidget onSuccess={onSuccess} />);
    
    // Now update to success state
    (useWidgetIntent as jest.Mock).mockReturnValue({
      formState: {
        sourceChain: 'BASE',
        destinationChain: 'ARBITRUM',
        selectedToken: 'USDC',
        amount: '',
        recipient: '',
        tip: '0.2',
        isSubmitting: false,
        error: null,
        success: true,
        intentId: 'test_intent_id'
      },
      balance: '100',
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
      updateTip: jest.fn()
    });
    
    rerender(<SpeedrunWidget onSuccess={onSuccess} />);
    
    // Verify onSuccess callback was called with intent ID
    await waitFor(() => {
      expect(onSuccess).toHaveBeenCalledWith('test_intent_id');
    });
  });
  
  it('applies custom styles correctly', () => {
    render(
      <SpeedrunWidget
        customStyles={{
          containerClass: "test-container-class",
          buttonClass: "test-button-class",
          inputClass: "test-input-class",
          labelClass: "test-label-class"
        }}
      />
    );
    
    // Check that custom classes were applied
    const container = document.querySelector('.test-container-class');
    expect(container).toBeInTheDocument();
    
    const button = document.querySelector('.test-button-class');
    expect(button).toBeInTheDocument();
    
    const labels = document.querySelectorAll('.test-label-class');
    expect(labels.length).toBeGreaterThan(0);
  });
}); 