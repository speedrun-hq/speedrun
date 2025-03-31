import React, { useState } from 'react';
import { Chain, CreateIntentRequest } from '../types';
import { intentService } from '../services/api';

const CHAINS: Chain[] = ['ethereum', 'base', 'zetachain'];

export const CreateIntentForm: React.FC = () => {
  const [formData, setFormData] = useState<CreateIntentRequest>({
    source_chain: 'ethereum',
    source_address: '',
    target_chain: 'zetachain',
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
      await intentService.create(formData);
      setSuccess(true);
      setFormData({
        source_chain: 'ethereum',
        source_address: '',
        target_chain: 'zetachain',
        target_address: '',
        amount: '',
      });
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create intent');
    } finally {
      setLoading(false);
    }
  };

  const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) => {
    const { name, value } = e.target;
    setFormData((prev) => ({ ...prev, [name]: value }));
  };

  return (
    <div className="card">
      <h2 className="text-2xl font-bold mb-6">Create New Intent</h2>
      
      {error && (
        <div className="mb-4 p-4 bg-red-50 text-red-700 rounded-md">
          {error}
        </div>
      )}
      
      {success && (
        <div className="mb-4 p-4 bg-green-50 text-green-700 rounded-md">
          Intent created successfully!
        </div>
      )}

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
            className="input mt-1"
          >
            {CHAINS.map((chain) => (
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
            className="input mt-1"
            placeholder="0x..."
            required
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
            className="input mt-1"
          >
            {CHAINS.map((chain) => (
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
            className="input mt-1"
            placeholder="0x..."
            required
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
            className="input mt-1"
            placeholder="1000000"
            required
            min="0"
            step="1"
          />
          <p className="mt-1 text-sm text-gray-500">
            Amount in USDC (6 decimal places). Example: 1 USDC = 1000000
          </p>
        </div>

        <button
          type="submit"
          disabled={loading}
          className="btn btn-primary w-full"
        >
          {loading ? 'Creating...' : 'Create Intent'}
        </button>
      </form>
    </div>
  );
}; 