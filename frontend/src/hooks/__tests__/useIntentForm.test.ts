import { renderHook, act } from '@testing-library/react';
import { useIntentForm } from '../useIntentForm';
import { useAccount, useBalance, useNetwork, useTokenBalance } from 'wagmi';
import { useContractRead } from 'wagmi';
import { apiService } from '@/services/api';

// Mock the wagmi hooks
jest.mock('wagmi', () => ({
  useAccount: jest.fn(),
  useBalance: jest.fn(),
  useNetwork: jest.fn(),
  useTokenBalance: jest.fn(),
  useContractRead: jest.fn().mockReturnValue({
    data: BigInt('1000000000'),
    isError: false,
    isLoading: false,
  }),
}));

// Mock the API service
jest.mock('@/services/api', () => ({
  apiService: {
    createIntent: jest.fn(),
  },
}));

describe('useIntentForm', () => {
  const mockAccount = {
    address: '0x123',
    isConnected: true,
  };

  const mockBalance = {
    data: {
      formatted: '1000.00',
    },
    isLoading: false,
  };

  const mockNetwork = {
    chain: {
      id: 8453, // Base chain ID
    },
  };

  beforeEach(() => {
    // Reset all mocks before each test
    jest.clearAllMocks();

    // Set up default mock values
    (useAccount as jest.Mock).mockReturnValue(mockAccount);
    (useBalance as jest.Mock).mockReturnValue(mockBalance);
    (useNetwork as jest.Mock).mockReturnValue(mockNetwork);
    (useTokenBalance as jest.Mock).mockReturnValue({
      balance: BigInt('1000000000'),
      isError: false,
      isLoading: false,
      symbol: 'USDC'
    });
    (apiService.createIntent as jest.Mock).mockResolvedValue({});
  });

  afterEach(() => {
    jest.clearAllMocks();
  });

  it('initializes with default values', () => {
    const { result } = renderHook(() => useIntentForm());

    expect(result.current.formState).toEqual({
      sourceChain: 'BASE',
      destinationChain: 'ARBITRUM',
      selectedToken: 'USDC',
      amount: '',
      recipient: '',
      isSubmitting: false,
      error: null,
      success: false,
    });
    expect(result.current.balance).toBe('1000.00');
    expect(result.current.isConnected).toBe(true);
    expect(result.current.isValid).toBe(false);
  });

  it('updates source chain and destination chain accordingly', () => {
    const { result } = renderHook(() => useIntentForm());

    act(() => {
      result.current.updateSourceChain('ARBITRUM');
    });

    expect(result.current.formState.sourceChain).toBe('ARBITRUM');
    expect(result.current.formState.destinationChain).toBe('BASE');
  });

  it('updates token selection', () => {
    const { result } = renderHook(() => useIntentForm());

    act(() => {
      result.current.updateToken('USDT');
    });

    expect(result.current.formState.selectedToken).toBe('USDT');
    expect(result.current.symbol).toBe('USDT');
  });

  it('updates amount', () => {
    const { result } = renderHook(() => useIntentForm());

    act(() => {
      result.current.updateAmount('100.00');
    });

    expect(result.current.formState.amount).toBe('100.00');
  });

  it('updates recipient address', () => {
    const { result } = renderHook(() => useIntentForm());

    act(() => {
      result.current.updateRecipient('0x1234567890123456789012345678901234567890');
    });

    expect(result.current.formState.recipient).toBe('0x1234567890123456789012345678901234567890');
  });

  it('validates form correctly', () => {
    const { result } = renderHook(() => useIntentForm());

    // Initially invalid
    expect(result.current.isValid).toBe(false);

    // Set valid values
    act(() => {
      result.current.updateAmount('100.00');
      result.current.updateRecipient('0x1234567890123456789012345678901234567890');
    });

    // Should be valid now
    expect(result.current.isValid).toBe(true);
  });

  it('validates amount is not greater than balance', () => {
    const { result } = renderHook(() => useIntentForm());

    // Set amount greater than balance
    act(() => {
      result.current.updateAmount('2000.00');
      result.current.updateRecipient('0x1234567890123456789012345678901234567890');
    });

    expect(result.current.isValid).toBe(false);
  });

  it('validates recipient address format', () => {
    const { result } = renderHook(() => useIntentForm());

    // Set invalid address
    act(() => {
      result.current.updateAmount('100.00');
      result.current.updateRecipient('invalid-address');
    });

    expect(result.current.isValid).toBe(false);
  });

  it('handles form submission successfully', async () => {
    const { result } = renderHook(() => useIntentForm());

    // Set valid values
    act(() => {
      result.current.updateAmount('100.00');
      result.current.updateRecipient('0x1234567890123456789012345678901234567890');
    });

    // Submit form
    await act(async () => {
      await result.current.handleSubmit();
    });

    expect(apiService.createIntent).toHaveBeenCalledWith({
      source_chain: 'base',
      destination_chain: 'arbitrum',
      token: 'USDC',
      amount: '100.00',
      recipient: '0x1234567890123456789012345678901234567890',
      intent_fee: '0.01',
    });
    expect(result.current.formState.success).toBe(true);
    expect(result.current.formState.error).toBeNull();
    expect(result.current.formState.amount).toBe('');
    expect(result.current.formState.recipient).toBe('');
  });

  it('handles form submission error', async () => {
    const { result } = renderHook(() => useIntentForm());

    // Set valid values
    act(() => {
      result.current.updateAmount('100.00');
      result.current.updateRecipient('0x1234567890123456789012345678901234567890');
    });

    // Mock error
    const error = new Error('Failed to create intent');
    (apiService.createIntent as jest.Mock).mockRejectedValueOnce(error);

    // Submit form
    await act(async () => {
      await result.current.handleSubmit();
    });

    expect(apiService.createIntent).toHaveBeenCalled();
    expect(result.current.formState.success).toBe(false);
    expect(result.current.formState.error).toBe(error);
  });

  it('handles loading balance state', () => {
    // Clear any previous mocks
    jest.clearAllMocks();

    // Set up default mock values for other hooks
    (useAccount as jest.Mock).mockReturnValue({
      address: '0x123',
      isConnected: true
    });
    (useBalance as jest.Mock).mockReturnValue(mockBalance);
    (useNetwork as jest.Mock).mockReturnValue(mockNetwork);

    // Mock the useTokenBalance hook to return loading state
    (useTokenBalance as jest.Mock).mockReturnValue({
      balance: null,
      isError: false,
      isLoading: true,
      symbol: 'USDC'
    });

    const { result } = renderHook(() => useIntentForm());
    expect(result.current.isLoading).toBe(true);
  });

  it('resets form after successful submission', async () => {
    const { result } = renderHook(() => useIntentForm());

    // Set valid values
    act(() => {
      result.current.updateAmount('100.00');
      result.current.updateRecipient('0x1234567890123456789012345678901234567890');
    });

    // Submit form
    await act(async () => {
      await result.current.handleSubmit();
    });

    expect(result.current.formState.amount).toBe('');
    expect(result.current.formState.recipient).toBe('');
    expect(result.current.formState.success).toBe(true);
  });
}); 