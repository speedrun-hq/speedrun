"use client";

import {
  Intent,
  Fulfillment,
  CreateIntentRequest,
  CreateFulfillmentRequest,
  Runner,
} from "@/types";
import { ApiError } from "@/utils/errors";

// Get API base URL from environment variable, fallback to localhost in development
const API_BASE_URL =
  process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api/v1";

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
  async listIntents(): Promise<Intent[]> {
    console.log("Calling listIntents API endpoint:", `${API_BASE_URL}/intents`);
    const response = await this.fetchApi<Intent[]>("/intents", {
      method: "GET",
    });
    console.log("listIntents raw response:", response);
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

  // Leaderboard endpoints
  async getLeaderboard(chainId: number): Promise<Runner[]> {
    return this.fetchApi<Runner[]>(`/leaderboard/${chainId}`, {
      method: "GET",
    });
  }
}

export const apiService = new ApiService();
