'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { CreateIntentRequest } from '@/types';
import { apiService } from '@/services/api';
import { ApiError } from '@/utils/errors';

const CHAINS = ['base', 'arbitrum'] as const;
const TOKENS = ['USDC'] as const;

export default function CreateIntentForm() {
  const router = useRouter();
  const [step, setStep] = useState(1);
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [formData, setFormData] = useState<CreateIntentRequest>({
    source_chain: 'base',
    destination_chain: 'base',
    token: 'USDC',
    amount: '',
    recipient: '',
    intent_fee: '0',
  });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<ApiError | null>(null);
  const [success, setSuccess] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError(null);
    setSuccess(false);

    try {
      await apiService.createIntent(formData);
      setSuccess(true);
      setTimeout(() => {
        router.push('/intents');
      }, 2000);
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err);
      } else if (err instanceof Error) {
        setError(new ApiError(err.message));
      } else {
        setError(new ApiError('An unexpected error occurred'));
      }
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

  const nextStep = () => {
    setStep(prev => Math.min(prev + 1, 3));
  };

  const prevStep = () => {
    setStep(prev => Math.max(prev - 1, 1));
  };

  return (
    <div className="max-w-2xl mx-auto">
      <div className="arcade-container">
        <div className="px-4 py-5 sm:p-6">
          <h3 className="arcade-text text-lg text-center text-yellow-500 mb-6">CREATE NEW RUN</h3>
          <form onSubmit={handleSubmit} className="mt-5 space-y-6">
            {step === 1 && (
              <div className="space-y-4">
                <div>
                  <label htmlFor="token" className="arcade-text block text-xs text-primary-500 mb-2">
                    SELECT TOKEN
                  </label>
                  <select
                    id="token"
                    name="token"
                    value={formData.token}
                    onChange={handleChange}
                    className="arcade-select"
                  >
                    {TOKENS.map(token => (
                      <option key={token} value={token}>
                        {token}
                      </option>
                    ))}
                  </select>
                </div>

                <div>
                  <label htmlFor="amount" className="arcade-text block text-xs text-primary-500 mb-2">
                    AMOUNT
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
                    className="arcade-input"
                    placeholder="0.00"
                  />
                </div>

                <button
                  type="button"
                  onClick={nextStep}
                  className="arcade-btn w-full mt-6"
                >
                  NEXT
                </button>
              </div>
            )}

            {step === 2 && (
              <div className="space-y-4">
                <div>
                  <label htmlFor="source_chain" className="arcade-text block text-xs text-primary-500 mb-2">
                    SOURCE CHAIN
                  </label>
                  <select
                    id="source_chain"
                    name="source_chain"
                    value={formData.source_chain}
                    onChange={handleChange}
                    className="arcade-select"
                  >
                    {CHAINS.map(chain => (
                      <option key={chain} value={chain}>
                        {chain.toUpperCase()}
                      </option>
                    ))}
                  </select>
                </div>

                <div>
                  <label htmlFor="destination_chain" className="arcade-text block text-xs text-primary-500 mb-2">
                    DESTINATION CHAIN
                  </label>
                  <select
                    id="destination_chain"
                    name="destination_chain"
                    value={formData.destination_chain}
                    onChange={handleChange}
                    className="arcade-select"
                  >
                    {CHAINS.map(chain => (
                      <option key={chain} value={chain}>
                        {chain.toUpperCase()}
                      </option>
                    ))}
                  </select>
                </div>

                <div className="flex space-x-4">
                  <button
                    type="button"
                    onClick={prevStep}
                    className="arcade-btn flex-1"
                  >
                    BACK
                  </button>
                  <button
                    type="button"
                    onClick={nextStep}
                    className="arcade-btn flex-1"
                  >
                    NEXT
                  </button>
                </div>
              </div>
            )}

            {step === 3 && (
              <div className="space-y-4">
                <div>
                  <label htmlFor="recipient" className="arcade-text block text-xs text-primary-500 mb-2">
                    RECIPIENT ADDRESS
                  </label>
                  <input
                    type="text"
                    id="recipient"
                    name="recipient"
                    value={formData.recipient}
                    onChange={handleChange}
                    required
                    className="arcade-input"
                    placeholder="0x..."
                  />
                </div>

                <div>
                  <label htmlFor="intent_fee" className="arcade-text block text-xs text-primary-500 mb-2">
                    INTENT FEE (OPTIONAL)
                  </label>
                  <input
                    type="number"
                    id="intent_fee"
                    name="intent_fee"
                    value={formData.intent_fee}
                    onChange={handleChange}
                    min="0"
                    step="0.01"
                    className="arcade-input"
                    placeholder="0.00"
                  />
                </div>

                <div className="flex space-x-4">
                  <button
                    type="button"
                    onClick={prevStep}
                    className="arcade-btn flex-1"
                  >
                    BACK
                  </button>
                  <button
                    type="submit"
                    disabled={loading}
                    className="arcade-btn flex-1"
                  >
                    {loading ? 'PROCESSING...' : 'START TRANSFER'}
                  </button>
                </div>
              </div>
            )}

            {error && (
              <div className="arcade-text bg-red-900/50 border-2 border-red-500 p-4 text-red-500">
                <p className="text-xs">{error.message}</p>
              </div>
            )}

            {success && (
              <div className="arcade-text bg-green-900/50 border-2 border-green-500 p-4 text-green-500">
                <p className="text-xs">INTENT CREATED SUCCESSFULLY! REDIRECTING...</p>
              </div>
            )}
          </form>
        </div>
      </div>
    </div>
  );
} 