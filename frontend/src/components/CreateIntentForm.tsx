'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { CreateIntentRequest } from '@/types';
import { intentService } from '@/services/api';

const CHAINS = ['ethereum', 'base', 'zetachain'] as const;

export default function CreateIntentForm() {
  const router = useRouter();
  const [formData, setFormData] = useState<CreateIntentRequest>({
    source_chain: 'ethereum',
    source_address: '',
    target_chain: 'base',
    target_address: '',
    amount: '',
  });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError(null);
    setSuccess(false);

    try {
      await intentService.createIntent(formData);
      setSuccess(true);
      setTimeout(() => {
        router.push('/intents');
      }, 2000);
    } catch (err) {
      setError('Failed to create intent');
      console.error('Error creating intent:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) => {
    const { name, value } = e.target;
    setFormData(prev => ({
      ...prev,
      [name]: value
    }));
  };

  return (
    <div className="max-w-2xl mx-auto">
      <div className="card">
        <div className="px-4 py-5 sm:p-6">
          <h3 className="text-lg leading-6 font-medium text-gray-900">Create New Intent</h3>
          <form onSubmit={handleSubmit} className="mt-5 space-y-6">
            <div>
              <label htmlFor="source_chain" className="block text-sm font-medium text-gray-700">
                Source Chain
              </label>
              <select
                id="source_chain"
                name="source_chain"
                value={formData.source_chain}
                onChange={handleChange}
                className="select"
              >
                {CHAINS.map(chain => (
                  <option key={chain} value={chain}>
                    {chain.charAt(0).toUpperCase() + chain.slice(1)}
                  </option>
                ))}
              </select>
            </div>

            <div>
              <label htmlFor="source_address" className="block text-sm font-medium text-gray-700">
                Source Address
              </label>
              <input
                type="text"
                id="source_address"
                name="source_address"
                value={formData.source_address}
                onChange={handleChange}
                required
                className="input"
                placeholder="0x..."
              />
            </div>

            <div>
              <label htmlFor="target_chain" className="block text-sm font-medium text-gray-700">
                Target Chain
              </label>
              <select
                id="target_chain"
                name="target_chain"
                value={formData.target_chain}
                onChange={handleChange}
                className="select"
              >
                {CHAINS.map(chain => (
                  <option key={chain} value={chain}>
                    {chain.charAt(0).toUpperCase() + chain.slice(1)}
                  </option>
                ))}
              </select>
            </div>

            <div>
              <label htmlFor="target_address" className="block text-sm font-medium text-gray-700">
                Target Address
              </label>
              <input
                type="text"
                id="target_address"
                name="target_address"
                value={formData.target_address}
                onChange={handleChange}
                required
                className="input"
                placeholder="0x..."
              />
            </div>

            <div>
              <label htmlFor="amount" className="block text-sm font-medium text-gray-700">
                Amount (USDC)
              </label>
              <input
                type="number"
                id="amount"
                name="amount"
                value={formData.amount}
                onChange={handleChange}
                required
                min="0"
                step="0.01"
                className="input"
                placeholder="0.00"
              />
            </div>

            {error && (
              <div className="bg-red-50 border border-red-200 rounded-md p-4">
                <p className="text-sm text-red-800">{error}</p>
              </div>
            )}

            {success && (
              <div className="bg-green-50 border border-green-200 rounded-md p-4">
                <p className="text-sm text-green-800">Intent created successfully! Redirecting...</p>
              </div>
            )}

            <div>
              <button
                type="submit"
                disabled={loading}
                className="btn-primary w-full disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {loading ? 'Creating...' : 'Create Intent'}
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  );
} 