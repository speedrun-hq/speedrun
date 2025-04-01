'use client';

import React from 'react';
import { useEffect, useState } from 'react';
import { Intent } from '@/types';
import { apiService } from '@/services/api';
import ErrorMessage from '@/components/ErrorMessage';

const ITEMS_PER_PAGE = 10;

const IntentList: React.FC = () => {
  const [intents, setIntents] = useState<Intent[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<unknown>(null);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);

  useEffect(() => {
    const fetchIntents = async () => {
      try {
        setLoading(true);
        setError(null);
        const { intents: data, total: totalCount } = await apiService.listIntents(page, ITEMS_PER_PAGE);
        setIntents(data);
        setTotal(totalCount);
      } catch (err) {
        setError(err);
      } finally {
        setLoading(false);
      }
    };

    fetchIntents();
  }, [page]);

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'pending':
        return 'text-primary-500 border-primary-500';
      case 'completed':
        return 'text-secondary-500 border-secondary-500';
      case 'failed':
        return 'text-accent-500 border-accent-500';
      case 'cancelled':
        return 'text-gray-500 border-gray-500';
      default:
        return 'text-gray-500 border-gray-500';
    }
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleString();
  };

  if (loading) {
    return (
      <div className="flex justify-center items-center h-64">
        <div className="arcade-text text-primary-500 animate-pulse">LOADING...</div>
      </div>
    );
  }

  if (error) {
    return <ErrorMessage error={error} className="mb-6" />;
  }

  return (
    <div className="w-full max-w-4xl mx-auto">
      <h2 className="arcade-text text-2xl mb-6 text-center text-primary-500">HIGH SCORES</h2>
      {intents.length === 0 ? (
        <p className="arcade-text text-gray-500 text-center">NO RECORDS FOUND</p>
      ) : (
        <div className="arcade-container">
          {intents.map((intent, index) => (
            <div
              key={intent.id}
              className="arcade-card"
            >
              <div className="flex justify-between items-start">
                <div className="space-y-2">
                  <p className="arcade-text text-sm text-primary-500">#{index + 1}</p>
                  <p className="arcade-text text-xs text-primary-500">ID: {intent.id}</p>
                  <p className="arcade-text text-xs text-primary-500">
                    {intent.source_chain} → {intent.destination_chain}
                  </p>
                  <p className="arcade-text text-xs text-primary-500">
                    AMOUNT: {intent.amount} {intent.token}
                  </p>
                  <p className="arcade-text text-xs text-gray-500">
                    {formatDate(intent.created_at)}
                  </p>
                </div>
                <span className={`arcade-status ${getStatusColor(intent.status)} border-2`}>
                  {intent.status.toUpperCase()}
                </span>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Pagination */}
      {total > ITEMS_PER_PAGE && (
        <div className="flex justify-center space-x-4 mt-6">
          <button
            onClick={() => setPage(p => Math.max(1, p - 1))}
            disabled={page === 1}
            className="arcade-btn disabled:opacity-50 disabled:cursor-not-allowed"
          >
            PREV
          </button>
          <button
            onClick={() => setPage(p => p + 1)}
            disabled={page * ITEMS_PER_PAGE >= total}
            className="arcade-btn disabled:opacity-50 disabled:cursor-not-allowed"
          >
            NEXT
          </button>
        </div>
      )}
    </div>
  );
};

export default IntentList; 