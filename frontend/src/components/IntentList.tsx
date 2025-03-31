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
        return 'bg-yellow-100 text-yellow-800';
      case 'completed':
        return 'bg-green-100 text-green-800';
      case 'failed':
        return 'bg-red-100 text-red-800';
      case 'cancelled':
        return 'bg-gray-100 text-gray-800';
      default:
        return 'bg-gray-100 text-gray-800';
    }
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleString();
  };

  if (loading) {
    return (
      <div className="flex justify-center items-center h-64">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary-600"></div>
      </div>
    );
  }

  if (error) {
    return <ErrorMessage error={error} className="mb-6" />;
  }

  return (
    <div className="w-full max-w-4xl mx-auto">
      <h2 className="text-2xl font-bold mb-4">Transfer Intents</h2>
      {intents.length === 0 ? (
        <p className="text-gray-500">No intents found</p>
      ) : (
        <div className="space-y-4">
          {intents.map((intent) => (
            <div
              key={intent.id}
              className="p-4 border rounded-lg shadow-sm hover:shadow-md transition-shadow"
            >
              <div className="flex justify-between items-start">
                <div>
                  <p className="font-medium">ID: {intent.id}</p>
                  <p className="text-sm text-gray-600">
                    {intent.source_chain} → {intent.destination_chain}
                  </p>
                  <p className="text-sm text-gray-600">
                    Amount: {intent.amount} {intent.token}
                  </p>
                  <p className="text-sm text-gray-500">
                    Created: {formatDate(intent.created_at)}
                  </p>
                </div>
                <span className={`px-2 py-1 text-sm rounded-full ${getStatusColor(intent.status)}`}>
                  {intent.status}
                </span>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Pagination */}
      {total > ITEMS_PER_PAGE && (
        <div className="mt-6 flex justify-center space-x-2">
          <button
            onClick={() => setPage(p => Math.max(1, p - 1))}
            disabled={page === 1}
            className="px-4 py-2 border rounded-md text-sm font-medium text-gray-700 bg-white hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            Previous
          </button>
          <button
            onClick={() => setPage(p => p + 1)}
            disabled={page * ITEMS_PER_PAGE >= total}
            className="px-4 py-2 border rounded-md text-sm font-medium text-gray-700 bg-white hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            Next
          </button>
        </div>
      )}
    </div>
  );
};

export default IntentList; 