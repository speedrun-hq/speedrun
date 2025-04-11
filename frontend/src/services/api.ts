"use client";

import {
  Intent,
  Fulfillment,
  CreateIntentRequest,
  CreateFulfillmentRequest,
  Runner,
  PaginationParams,
  PaginatedResponse,
  PaginatedIntentsResponse,
  PaginatedFulfillmentsResponse
} from "@/types";
import { ApiError } from "@/utils/errors";

// Get API base URL from environment variable, fallback to localhost in development
const API_BASE_URL =
  process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api/v1";

const DEFAULT_PAGE_SIZE = 20;

class ApiService {
  private async fetchApi<T>(
    endpoint: string,
    options: RequestInit = {},
  ): Promise<T> {
    // Only execute fetch in browser environment
    if (typeof window === "undefined") {
      return Promise.resolve([] as unknown as T);
    }

    try {
      const response = await fetch(`${API_BASE_URL}${endpoint}`, {
        ...options,
        headers: {
          "Content-Type": "application/json",
          ...options.headers,
        },
      });

      let data;
      const text = await response.text();
      try {
        data = JSON.parse(text);
      } catch (parseError) {
        console.error("Failed to parse JSON response:", text);
        const errorMessage =
          parseError instanceof Error
            ? parseError.message
            : "Unknown parsing error";
        throw new ApiError(
          `Invalid JSON response from server: ${errorMessage}`,
          response.status,
          "INVALID_JSON",
        );
      }

      if (!response.ok) {
        throw new ApiError(
          data.error || "An error occurred",
          response.status,
          data.code,
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
      throw new ApiError("An unexpected error occurred");
    }
  }

  // Intent endpoints
  async listIntents(pagination?: PaginationParams): Promise<PaginatedIntentsResponse> {
    const queryString = this.getPaginationQueryString(pagination);
    console.log("Calling listIntents API endpoint:", `${API_BASE_URL}/intents${queryString}`);
    
    const response = await this.fetchApi<PaginatedIntentsResponse>(`/intents${queryString}`, {
      method: "GET",
    });
    
    console.log("listIntents raw response:", response);
    return response;
  }

  async listIntentsBySender(senderAddress: string, pagination?: PaginationParams): Promise<PaginatedIntentsResponse> {
    const queryString = this.getPaginationQueryString(pagination);
    console.log(
      "Calling listIntentsBySender API endpoint:",
      `${API_BASE_URL}/intents/sender/${senderAddress}${queryString}`,
    );
    
    const response = await this.fetchApi<PaginatedIntentsResponse>(
      `/intents/sender/${senderAddress}${queryString}`,
      {
        method: "GET",
      },
    );
    
    console.log("listIntentsBySender raw response:", response);
    return response;
  }

  async listIntentsByRecipient(recipientAddress: string, pagination?: PaginationParams): Promise<PaginatedIntentsResponse> {
    const queryString = this.getPaginationQueryString(pagination);
    console.log(
      "Calling listIntentsByRecipient API endpoint:",
      `${API_BASE_URL}/intents/recipient/${recipientAddress}${queryString}`,
    );
    
    const response = await this.fetchApi<PaginatedIntentsResponse>(
      `/intents/recipient/${recipientAddress}${queryString}`,
      {
        method: "GET",
      },
    );
    
    console.log("listIntentsByRecipient raw response:", response);
    return response;
  }

  async getIntent(id: string): Promise<Intent> {
    return this.fetchApi<Intent>(`/intents/${id}`);
  }

  async createIntent(data: CreateIntentRequest): Promise<Intent> {
    return this.fetchApi<Intent>("/intents", {
      method: "POST",
      body: JSON.stringify(data),
    });
  }

  // Fulfillment endpoints
  async createFulfillment(
    data: CreateFulfillmentRequest,
  ): Promise<Fulfillment> {
    return this.fetchApi<Fulfillment>("/fulfillments", {
      method: "POST",
      body: JSON.stringify(data),
    });
  }

  async getFulfillment(id: string): Promise<Fulfillment> {
    return this.fetchApi<Fulfillment>(`/fulfillments/${id}`);
  }

  async listFulfillments(pagination?: PaginationParams): Promise<PaginatedFulfillmentsResponse> {
    const queryString = this.getPaginationQueryString(pagination);
    return this.fetchApi<PaginatedFulfillmentsResponse>(`/fulfillments${queryString}`, {
      method: "GET",
    });
  }

  // Leaderboard endpoints
  async getLeaderboard(chainId: number): Promise<Runner[]> {
    return this.fetchApi<Runner[]>(`/leaderboard/${chainId}`, {
      method: "GET",
    });
  }

  // Helper to build query string from pagination params
  private getPaginationQueryString(pagination?: PaginationParams): string {
    if (!pagination) return '';
    const { page, page_size } = pagination;
    const params = new URLSearchParams();
    if (page) params.append('page', page.toString());
    if (page_size) params.append('page_size', page_size.toString());
    const queryString = params.toString();
    return queryString ? `?${queryString}` : '';
  }
}

export const apiService = new ApiService();
