import { Intent, Fulfillment, CreateIntentRequest, CreateFulfillmentRequest } from '@/types';
import { ApiError } from '@/utils/errors';

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api/v1';

class ApiService {
  private async fetchApi<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
    try {
      const response = await fetch(`${API_BASE_URL}${endpoint}`, {
        ...options,
        headers: {
          'Content-Type': 'application/json',
          ...options.headers,
        },
      });

      const data = await response.json();

      if (!response.ok) {
        throw new ApiError(
          data.error || 'An error occurred',
          response.status,
          data.code
        );
      }

      return data;
    } catch (error) {
      if (error instanceof ApiError) {
        throw error;
      }
      if (error instanceof Error) {
        throw new ApiError(error.message);
      }
      throw new ApiError('An unexpected error occurred');
    }
  }

  // Intent endpoints
  async listIntents(limit: number = 10, offset: number = 0, status?: string): Promise<Intent[]> {
    let url = `/intents?limit=${limit}&offset=${offset}`;
    if (status) {
      url += `&status=${status}`;
    }
    return this.fetchApi(url);
  }

  async getIntent(id: string): Promise<Intent> {
    return this.fetchApi(`/intents/${id}`);
  }

  async createIntent(data: CreateIntentRequest): Promise<Intent> {
    return this.fetchApi('/intents', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  // Fulfillment endpoints
  async createFulfillment(data: CreateFulfillmentRequest): Promise<Fulfillment> {
    return this.fetchApi('/fulfillments', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async getFulfillment(id: string): Promise<Fulfillment> {
    return this.fetchApi(`/fulfillments/${id}`);
  }
}

export const apiService = new ApiService(); 