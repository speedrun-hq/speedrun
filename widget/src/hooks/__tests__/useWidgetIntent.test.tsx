import { renderHook, act, waitFor } from '@testing-library/react';
import { useWidgetIntent } from '../useWidgetIntent';
import { apiClient } from '../../services/apiClient';

// Mock the entire apiClient module
jest.mock('../../services/apiClient', () => ({
  apiClient: {
    createIntent: jest.fn(),
    getIntent: jest.fn(),
    getFulfillment: jest.fn(),
    setMockMode: jest.fn()
  }
}));

// Mock the wagmi hooks
jest.mock('wagmi', () => ({
  useAccount: () => ({
    address: '0x1234567890123456789012345678901234567890',
    isConnected: true
  })
}));

describe('useWidgetIntent', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    
    // Mock successful createIntent response
    (apiClient.createIntent as jest.Mock).mockResolvedValue({
      id: 'test_intent_id',
      intentHash: '0xabcdef1234567890'
    });
    
    // Mock successful getIntent response
    (apiClient.getIntent as jest.Mock).mockResolvedValue({
      id: 'test_intent_id',
      status: 'fulfilled',
      source_chain: 42161,
      destination_chain: 8453,
      token: '0xabcdef',
      amount: '1000000',
      recipient: '0x1111111111111111111111111111111111111111',
      sender: '0x2222222222222222222222222222222222222222',
      intent_fee: '100000',
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString()
    });
    
    // Mock successful getFulfillment response
    (apiClient.getFulfillment as jest.Mock).mockResolvedValue({
      intent_id: 'test_intent_id',
      tx_hash: '0xabcdef1234567890',
      fulfiller: '0x3333333333333333333333333333333333333333',
      amount: '1000000',
      created_at: new Date().toISOString()
    });
    
    // Clear all timers
    jest.useFakeTimers();
  });
  
  afterEach(() => {
    jest.useRealTimers();
  });
  
  it('should initialize with default values', () => {
    const { result } = renderHook(() => useWidgetIntent());
    
    expect(result.current).toEqual(expect.objectContaining({
      formState: expect.objectContaining({
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
      }),
      isValid: false
    }));
  });
  
  it('should update form state correctly', () => {
    const { result } = renderHook(() => useWidgetIntent());
    
    act(() => {
      result.current.updateSourceChain('ETHEREUM');
    });
    expect(result.current.formState.sourceChain).toBe('ETHEREUM');
    
    act(() => {
      result.current.updateDestinationChain('POLYGON');
    });
    expect(result.current.formState.destinationChain).toBe('POLYGON');
    
    act(() => {
      result.current.updateToken('DAI');
    });
    expect(result.current.formState.selectedToken).toBe('DAI');
    
    act(() => {
      result.current.updateAmount('10');
    });
    expect(result.current.formState.amount).toBe('10');
    
    act(() => {
      result.current.updateRecipient('0x1234567890123456789012345678901234567890');
    });
    expect(result.current.formState.recipient).toBe('0x1234567890123456789012345678901234567890');
    
    act(() => {
      result.current.updateTip('0.5');
    });
    expect(result.current.formState.tip).toBe('0.5');
  });
  
  it('should not allow source and destination chains to be the same', () => {
    const { result } = renderHook(() => useWidgetIntent());
    
    act(() => {
      result.current.updateSourceChain('BASE');
      result.current.updateDestinationChain('BASE');
    });
    
    // Should have chosen a different destination
    expect(result.current.formState.sourceChain).toBe('ARBITRUM');
    expect(result.current.formState.destinationChain).not.toBe('ARBITRUM');
  });
  
  it('should handle form submission correctly', async () => {
    const { result } = renderHook(() => useWidgetIntent());
    
    // Set valid form values
    act(() => {
      result.current.updateAmount('10');
      result.current.updateRecipient('0x1234567890123456789012345678901234567890');
      result.current.updateTip('0.5');
    });
    
    // Wait for balance to load in the background
    act(() => {
      jest.advanceTimersByTime(600);
    });
    
    // Now submit the form
    act(() => {
      result.current.handleSubmit();
    });
    
    // Should be submitting
    expect(result.current.formState.isSubmitting).toBe(true);
    
    // Wait for the async operations to complete
    await waitFor(() => {
      expect(apiClient.createIntent).toHaveBeenCalledWith(expect.objectContaining({
        sourceChainId: expect.any(Number),
        destinationChainId: expect.any(Number),
        tokenAddress: expect.any(String),
        amount: '10',
        recipient: '0x1234567890123456789012345678901234567890',
        tip: '0.5',
        sender: '0x1234567890123456789012345678901234567890'
      }));
    });
    
    // Form should be reset with success
    await waitFor(() => {
      expect(result.current.formState.success).toBe(true);
      expect(result.current.formState.intentId).toBe('test_intent_id');
      expect(result.current.formState.amount).toBe('');
    });
    
    // Should poll for status updates
    act(() => {
      jest.advanceTimersByTime(1500);
    });
    
    await waitFor(() => {
      expect(apiClient.getIntent).toHaveBeenCalledWith('test_intent_id');
      expect(apiClient.getFulfillment).toHaveBeenCalledWith('test_intent_id');
    });
    
    // Should have fulfillment data
    await waitFor(() => {
      expect(result.current.formState.fulfillmentTxHash).toBe('0xabcdef1234567890');
    });
  });
  
  it('should handle API errors during submission', async () => {
    // Mock API error
    (apiClient.createIntent as jest.Mock).mockRejectedValueOnce(new Error('API Error'));
    
    const { result } = renderHook(() => useWidgetIntent());
    
    // Set valid form values
    act(() => {
      result.current.updateAmount('10');
      result.current.updateRecipient('0x1234567890123456789012345678901234567890');
      result.current.updateTip('0.5');
    });
    
    // Wait for balance to load in the background
    act(() => {
      jest.advanceTimersByTime(600);
    });
    
    // Now submit the form
    act(() => {
      result.current.handleSubmit();
    });
    
    // Wait for error
    await waitFor(() => {
      expect(result.current.formState.error).toEqual(expect.objectContaining({
        message: expect.stringContaining('API Error')
      }));
      expect(result.current.formState.isSubmitting).toBe(false);
      expect(result.current.formState.success).toBe(false);
    });
  });
  
  it('should reset the form correctly', () => {
    const { result } = renderHook(() => useWidgetIntent());
    
    // Set up form with values
    act(() => {
      result.current.updateAmount('10');
      result.current.updateRecipient('0x1234567890123456789012345678901234567890');
      result.current.updateTip('0.5');
    });
    
    // Manually set success state
    act(() => {
      // This is a workaround since we can't directly modify result.current.formState
      result.current.handleSubmit(); // This won't actually do the submission due to isValid check
      
      // Directly call the success part that resetForm would clear
      const intentId = 'test_intent_id';
      const error = new Error('Test error');
      const fulfillmentTxHash = '0xabcdef';
      
      // Now we can test resetForm clears these
      Object.defineProperty(result.current.formState, 'success', { value: true });
      Object.defineProperty(result.current.formState, 'intentId', { value: intentId });
      Object.defineProperty(result.current.formState, 'error', { value: error });
      Object.defineProperty(result.current.formState, 'fulfillmentTxHash', { value: fulfillmentTxHash });
    });
    
    // Reset the form
    act(() => {
      result.current.resetForm();
    });
    
    // Check form is reset
    expect(result.current.formState.success).toBe(false);
    expect(result.current.formState.intentId).toBe(null);
    expect(result.current.formState.error).toBe(null);
    expect(result.current.formState.fulfillmentTxHash).toBe(null);
    expect(result.current.formState.amount).toBe('');
  });
}); 