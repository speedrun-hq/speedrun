import axios, { AxiosInstance } from 'axios';

interface IntentRequest {
  sourceChainId: number;
  destinationChainId: number;
  tokenAddress: string;
  amount: string;
  recipient: string;
  tip: string;
  sender: string;
}

interface Intent {
  id: string;
  status: 'pending' | 'fulfilled' | 'settled' | 'failed';
  source_chain: number;
  destination_chain: number;
  token: string;
  amount: string;
  recipient: string;
  sender: string;
  intent_fee: string;
  created_at: string;
  updated_at: string;
}

interface Fulfillment {
  intent_id: string;
  tx_hash: string;
  fulfiller: string;
  amount: string;
  created_at: string;
}

interface IntentResponse {
  id: string;
  intentHash?: string;
}

// Default API URL - can be configured during initialization
const DEFAULT_API_URL = 'https://api.speedrun.exchange';

// Network timeouts for axios requests
const NETWORK_TIMEOUT = 10000; // 10 seconds

class ApiClient {
  private apiUrl: string;
  private axiosInstance: AxiosInstance;
  private mockMode: boolean;
  
  constructor(apiUrl = DEFAULT_API_URL, mockMode = false) {
    this.apiUrl = apiUrl;
    this.mockMode = mockMode;
    
    // Initialize axios with default config
    this.axiosInstance = axios.create({
      baseURL: this.apiUrl,
      timeout: NETWORK_TIMEOUT,
      headers: {
        'Content-Type': 'application/json',
        'Accept': 'application/json',
      }
    });

    // Add response interceptor for error handling
    this.axiosInstance.interceptors.response.use(
      response => response,
      error => {
        // Format the error for consistent handling
        const formattedError = new Error(
          error.response?.data?.message || 
          error.message || 
          'Unknown API error'
        );
        return Promise.reject(formattedError);
      }
    );
  }
  
  // Configure the API URL
  public configure(apiUrl: string, mockMode = false) {
    this.apiUrl = apiUrl;
    this.mockMode = mockMode;
    
    this.axiosInstance = axios.create({
      baseURL: this.apiUrl,
      timeout: NETWORK_TIMEOUT,
      headers: {
        'Content-Type': 'application/json',
        'Accept': 'application/json',
      }
    });
  }
  
  // Create an intent
  public async createIntent(params: IntentRequest): Promise<IntentResponse> {
    try {
      // If in mock mode, return simulated data (for testing)
      if (this.mockMode) {
        await new Promise(resolve => setTimeout(resolve, 800));
        return {
          id: `intent_${Math.random().toString(36).substring(2, 15)}`,
          intentHash: `0x${Math.random().toString(36).substring(2, 38)}`
        };
      }
      
      // Make real API call
      const response = await this.axiosInstance.post('/v1/intents', {
        source_chain_id: params.sourceChainId,
        destination_chain_id: params.destinationChainId,
        token_address: params.tokenAddress,
        amount: params.amount,
        recipient: params.recipient,
        tip: params.tip,
        sender: params.sender
      });
      
      return {
        id: response.data.id,
        intentHash: response.data.intent_hash
      };
    } catch (error) {
      throw new Error(`Failed to create intent: ${error instanceof Error ? error.message : 'Unknown error'}`);
    }
  }
  
  // Get an intent by ID
  public async getIntent(intentId: string): Promise<Intent> {
    try {
      // If in mock mode, return simulated data (for testing)
      if (this.mockMode) {
        await new Promise(resolve => setTimeout(resolve, 300));
        const random = Math.random();
        const status = random < 0.3 ? 'pending' : 'fulfilled';
        
        return {
          id: intentId,
          status,
          source_chain: 42161, // Arbitrum
          destination_chain: 8453, // Base
          token: '0xaf88d065e77c8cC2239327C5EDb3A432268e5831', // USDC on Arbitrum
          amount: '10000000', // 10 USDC with 6 decimals
          recipient: '0x' + '1'.repeat(40),
          sender: '0x' + '2'.repeat(40),
          intent_fee: '100000', // 0.1 USDC with 6 decimals
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString()
        };
      }
      
      // Make real API call
      const response = await this.axiosInstance.get(`/v1/intents/${intentId}`);
      
      return response.data;
    } catch (error) {
      throw new Error(`Failed to fetch intent: ${error instanceof Error ? error.message : 'Unknown error'}`);
    }
  }
  
  // Get fulfillment details for an intent
  public async getFulfillment(intentId: string): Promise<Fulfillment> {
    try {
      // If in mock mode, return simulated data (for testing)
      if (this.mockMode) {
        await new Promise(resolve => setTimeout(resolve, 300));
        
        return {
          intent_id: intentId,
          tx_hash: `0x${Math.random().toString(36).substring(2, 38)}`,
          fulfiller: '0x' + '3'.repeat(40),
          amount: '10000000', // 10 USDC with 6 decimals
          created_at: new Date().toISOString()
        };
      }
      
      // Make real API call
      const response = await this.axiosInstance.get(`/v1/intents/${intentId}/fulfillment`);
      
      return response.data;
    } catch (error) {
      throw new Error(`Failed to fetch fulfillment: ${error instanceof Error ? error.message : 'Unknown error'}`);
    }
  }

  // Set mock mode (useful for testing)
  public setMockMode(mockMode: boolean) {
    this.mockMode = mockMode;
  }
}

// Export a singleton instance (default mode is determined by environment)
// In a real production app, we'd check process.env.NODE_ENV
// For now, default to real API calls
const isMockMode = false;
export const apiClient = new ApiClient(DEFAULT_API_URL, isMockMode);

// For testing or custom configuration
export const createApiClient = (apiUrl: string, mockMode = false) => new ApiClient(apiUrl, mockMode); 