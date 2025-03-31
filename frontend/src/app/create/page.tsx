'use client';

import React, { useState } from 'react';
import { useRouter } from 'next/navigation';
import { apiService } from '@/services/api';
import { CreateIntentRequest } from '@/types';
import ErrorMessage from '@/components/ErrorMessage';

const SUPPORTED_CHAINS = ['base', 'polygon'];
const SUPPORTED_TOKENS = ['USDC'];

export default function CreateIntentPage() {
  const router = useRouter();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const [formData, setFormData] = useState<CreateIntentRequest>({
    source_chain: '',
    destination_chain: '',
    token: 'USDC',
    amount: '',
    recipient: '',
    intent_fee: '0',
  });

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError(null);

    try {
      await apiService.createIntent(formData);
      router.push('/intents');
    } catch (err) {
      setError(err);
    } finally {
      setLoading(false);
    }
  };

  const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) => {
    const { name, value } = e.target;
    setFormData(prev => ({ ...prev, [name]: value }));
  };

  return (
    <main className="container mx-auto px-4 py-8">
      <div className="max-w-2xl mx-auto">
        <h1 className="text-2xl font-bold mb-6">Create Transfer Intent</h1>
        
        {error && <ErrorMessage error={error} className="mb-6" />}

        <form onSubmit={handleSubmit} className="space-y-6">
          <div>
            <label htmlFor="source_chain" className="block text-sm font-medium text-gray-700">
              Source Chain
            </label>
            <select
              id="source_chain"
              name="source_chain"
              value={formData.source_chain}
              onChange={handleChange}
              required
              className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-primary-500 focus:ring-primary-500"
            >
              <option value="">Select a chain</option>
              {SUPPORTED_CHAINS.map(chain => (
                <option key={chain} value={chain}>
                  {chain.charAt(0).toUpperCase() + chain.slice(1)}
                </option>
              ))}
            </select>
          </div>

          <div>
            <label htmlFor="destination_chain" className="block text-sm font-medium text-gray-700">
              Destination Chain
            </label>
            <select
              id="destination_chain"
              name="destination_chain"
              value={formData.destination_chain}
              onChange={handleChange}
              required
              className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-primary-500 focus:ring-primary-500"
            >
              <option value="">Select a chain</option>
              {SUPPORTED_CHAINS.map(chain => (
                <option key={chain} value={chain}>
                  {chain.charAt(0).toUpperCase() + chain.slice(1)}
                </option>
              ))}
            </select>
          </div>

          <div>
            <label htmlFor="token" className="block text-sm font-medium text-gray-700">
              Token
            </label>
            <select
              id="token"
              name="token"
              value={formData.token}
              onChange={handleChange}
              required
              className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-primary-500 focus:ring-primary-500"
            >
              {SUPPORTED_TOKENS.map(token => (
                <option key={token} value={token}>
                  {token}
                </option>
              ))}
            </select>
          </div>

          <div>
            <label htmlFor="amount" className="block text-sm font-medium text-gray-700">
              Amount
            </label>
            <input
              type="text"
              id="amount"
              name="amount"
              value={formData.amount}
              onChange={handleChange}
              required
              pattern="^[0-9]+\.?[0-9]{0,18}$"
              placeholder="0.00"
              className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-primary-500 focus:ring-primary-500"
            />
          </div>

          <div>
            <label htmlFor="recipient" className="block text-sm font-medium text-gray-700">
              Recipient Address
            </label>
            <input
              type="text"
              id="recipient"
              name="recipient"
              value={formData.recipient}
              onChange={handleChange}
              required
              pattern="^0x[a-fA-F0-9]{40}$"
              placeholder="0x..."
              className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-primary-500 focus:ring-primary-500"
            />
          </div>

          <div>
            <label htmlFor="intent_fee" className="block text-sm font-medium text-gray-700">
              Intent Fee
            </label>
            <input
              type="text"
              id="intent_fee"
              name="intent_fee"
              value={formData.intent_fee}
              onChange={handleChange}
              required
              pattern="^[0-9]+\.?[0-9]{0,18}$"
              placeholder="0.00"
              className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-primary-500 focus:ring-primary-500"
            />
          </div>

          <div className="flex justify-end space-x-4">
            <button
              type="button"
              onClick={() => router.back()}
              className="px-4 py-2 border rounded-md text-sm font-medium text-gray-700 bg-white hover:bg-gray-50"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={loading}
              className="px-4 py-2 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-primary-600 hover:bg-primary-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary-500 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {loading ? 'Creating...' : 'Create Intent'}
            </button>
          </div>
        </form>
      </div>
    </main>
  );
} 