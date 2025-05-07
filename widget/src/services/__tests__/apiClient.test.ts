import axios from 'axios';
import { apiClient, createApiClient } from '../apiClient';

// Mock axios with a proper spy setup
jest.mock('axios', () => {
  // Create mock functions that we'll use in the mock implementation
  const mockPostImpl = jest.fn();
  const mockGetImpl = jest.fn();
  const mockUseImpl = jest.fn();
  
  return {
    create: jest.fn(() => ({
      post: mockPostImpl,
      get: mockGetImpl,
      interceptors: {
        response: {
          use: mockUseImpl
        }
      }
    })),
    // Export the mocks so we can access them in tests
    __mocks: {
      post: mockPostImpl,
      get: mockGetImpl,
      use: mockUseImpl
    }
  };
});

// Get references to the mock functions
const mockPost = (axios as any).__mocks.post;
const mockGet = (axios as any).__mocks.get;
const mockUse = (axios as any).__mocks.use;

describe('ApiClient', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    
    // Mock successful responses by default
    mockPost.mockResolvedValue({
      data: {
        id: 'test_intent_id',
        intent_hash: '0xtest_intent_hash'
      }
    });
    
    mockGet.mockImplementation((url: string) => {
      if (url.includes('/fulfillment')) {
        return Promise.resolve({
          data: {
            intent_id: 'test_intent_id',
            tx_hash: '0xtest_tx_hash',
            fulfiller: '0xtest_fulfiller',
            amount: '1000000',
            created_at: new Date().toISOString()
          }
        });
      } else {
        return Promise.resolve({
          data: {
            id: 'test_intent_id',
            status: 'fulfilled',
            source_chain: 42161,
            destination_chain: 8453,
            token: '0xtest_token',
            amount: '1000000',
            recipient: '0xtest_recipient',
            sender: '0xtest_sender',
            intent_fee: '100000',
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString()
          }
        });
      }
    });
  });
  
  describe('constructor', () => {
    it('should create axios instance with default URL', () => {
      const client = createApiClient('https://test.api.com');
      
      expect(axios.create).toHaveBeenCalledWith(expect.objectContaining({
        baseURL: 'https://test.api.com',
        headers: {
          'Content-Type': 'application/json',
          'Accept': 'application/json',
        }
      }));
    });
  });
  
  describe('configure', () => {
    it('should update API URL and recreate axios instance', () => {
      const client = createApiClient('https://api.speedrun.exchange');
      
      // Reset mock count
      (axios.create as jest.Mock).mockClear();
      
      // Configure with new URL
      client.configure('https://new.api.com');
      
      // Should have created a new axios instance
      expect(axios.create).toHaveBeenCalledWith(expect.objectContaining({
        baseURL: 'https://new.api.com'
      }));
    });
  });
  
  describe('createIntent', () => {
    it('should make POST request to create intent', async () => {
      // Real API mode
      apiClient.setMockMode(false);
      
      const result = await apiClient.createIntent({
        sourceChainId: 42161,
        destinationChainId: 8453,
        tokenAddress: '0xtest_token',
        amount: '10',
        recipient: '0xtest_recipient',
        tip: '0.5',
        sender: '0xtest_sender'
      });
      
      // Should make POST request with correct data
      expect(mockPost).toHaveBeenCalledWith('/v1/intents', {
        source_chain_id: 42161,
        destination_chain_id: 8453,
        token_address: '0xtest_token',
        amount: '10',
        recipient: '0xtest_recipient',
        tip: '0.5',
        sender: '0xtest_sender'
      });
      
      // Should return formatted response
      expect(result).toEqual({
        id: 'test_intent_id',
        intentHash: '0xtest_intent_hash'
      });
    });
    
    it('should return mock data in mock mode', async () => {
      // Mock mode
      apiClient.setMockMode(true);
      
      const result = await apiClient.createIntent({
        sourceChainId: 42161,
        destinationChainId: 8453,
        tokenAddress: '0xtest_token',
        amount: '10',
        recipient: '0xtest_recipient',
        tip: '0.5',
        sender: '0xtest_sender'
      });
      
      // Should not make API call
      expect(mockPost).not.toHaveBeenCalled();
      
      // Should return mock data
      expect(result).toEqual(expect.objectContaining({
        id: expect.stringContaining('intent_'),
        intentHash: expect.stringContaining('0x')
      }));
    });
    
    it('should handle errors correctly', async () => {
      // Real API mode
      apiClient.setMockMode(false);
      
      // Mock error response
      mockPost.mockRejectedValueOnce(new Error('API error'));
      
      // Should throw error with message
      await expect(apiClient.createIntent({
        sourceChainId: 42161,
        destinationChainId: 8453,
        tokenAddress: '0xtest_token',
        amount: '10',
        recipient: '0xtest_recipient',
        tip: '0.5',
        sender: '0xtest_sender'
      })).rejects.toThrow('Failed to create intent: API error');
    });
  });
  
  describe('getIntent', () => {
    it('should make GET request to fetch intent', async () => {
      // Real API mode
      apiClient.setMockMode(false);
      
      const result = await apiClient.getIntent('test_intent_id');
      
      // Should make GET request with correct URL
      expect(mockGet).toHaveBeenCalledWith('/v1/intents/test_intent_id');
      
      // Should return API response
      expect(result).toEqual(expect.objectContaining({
        id: 'test_intent_id',
        status: 'fulfilled'
      }));
    });
    
    it('should return mock data in mock mode', async () => {
      // Mock mode
      apiClient.setMockMode(true);
      
      const result = await apiClient.getIntent('test_intent_id');
      
      // Should not make API call
      expect(mockGet).not.toHaveBeenCalled();
      
      // Should return mock data
      expect(result).toEqual(expect.objectContaining({
        id: 'test_intent_id',
        status: expect.stringMatching(/pending|fulfilled/),
      }));
    });
  });
  
  describe('getFulfillment', () => {
    it('should make GET request to fetch fulfillment', async () => {
      // Real API mode
      apiClient.setMockMode(false);
      
      const result = await apiClient.getFulfillment('test_intent_id');
      
      // Should make GET request with correct URL
      expect(mockGet).toHaveBeenCalledWith('/v1/intents/test_intent_id/fulfillment');
      
      // Should return API response
      expect(result).toEqual(expect.objectContaining({
        intent_id: 'test_intent_id',
        tx_hash: '0xtest_tx_hash'
      }));
    });
    
    it('should return mock data in mock mode', async () => {
      // Mock mode
      apiClient.setMockMode(true);
      
      const result = await apiClient.getFulfillment('test_intent_id');
      
      // Should not make API call
      expect(mockGet).not.toHaveBeenCalled();
      
      // Should return mock data
      expect(result).toEqual(expect.objectContaining({
        intent_id: 'test_intent_id',
        tx_hash: expect.stringContaining('0x')
      }));
    });
  });
}); 