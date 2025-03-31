import axios from 'axios';
import { Intent, Fulfillment, CreateIntentRequest, CreateFulfillmentRequest } from '../types';

const API_BASE_URL = process.env.REACT_APP_API_BASE_URL || 'http://localhost:8080/api/v1';

const api = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Add API key to requests
api.interceptors.request.use((config) => {
  const apiKey = process.env.REACT_APP_API_KEY;
  if (apiKey) {
    config.headers.Authorization = `Bearer ${apiKey}`;
  }
  return config;
});

export const intentService = {
  create: async (data: CreateIntentRequest): Promise<Intent> => {
    const response = await api.post('/intents', data);
    return response.data.intent;
  },

  get: async (id: string): Promise<Intent> => {
    const response = await api.get(`/intents/${id}`);
    return response.data.intent;
  },

  list: async (params?: {
    status?: string;
    source_chain?: string;
    target_chain?: string;
    page?: number;
    limit?: number;
  }): Promise<{ intents: Intent[]; total: number; page: number; limit: number }> => {
    const response = await api.get('/intents', { params });
    return response.data;
  },
};

export const fulfillmentService = {
  create: async (data: CreateFulfillmentRequest): Promise<Fulfillment> => {
    const response = await api.post('/fulfillments', data);
    return response.data.fulfillment;
  },

  get: async (id: string): Promise<Fulfillment> => {
    const response = await api.get(`/fulfillments/${id}`);
    return response.data.fulfillment;
  },
}; 