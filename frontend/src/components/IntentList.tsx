"use client";

import React from "react";
import { useEffect, useState } from "react";
import { Intent, PaginationParams } from "@/types";
import { apiService } from "@/services/api";
import ErrorMessage from "@/components/ErrorMessage";
import IntentTile from "@/components/IntentTile";

const ITEMS_PER_PAGE = 10;

const IntentList: React.FC = () => {
  const [intents, setIntents] = useState<Intent[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<unknown>(null);
  const [currentPage, setCurrentPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const [totalCount, setTotalCount] = useState(0);

  useEffect(() => {
    const fetchIntents = async () => {
      try {
        setLoading(true);
        setError(null);

        const pagination: PaginationParams = {
          page: currentPage,
          page_size: ITEMS_PER_PAGE,
        };

        const response = await apiService.listIntents(pagination);
        setIntents(response.data);
        setTotalPages(response.total_pages);
        setTotalCount(response.total_count);
      } catch (err) {
        setError(err);
      } finally {
        setLoading(false);
      }
    };

    fetchIntents();
  }, [currentPage]);

  const handlePrevPage = () => {
    setCurrentPage((prev) => Math.max(1, prev - 1));
  };

  const handleNextPage = () => {
    setCurrentPage((prev) => (prev < totalPages ? prev + 1 : prev));
  };

  const getStatusColor = (status: string) => {
    switch (status.toLowerCase()) {
      case "pending":
        return "text-primary-500 border-primary-500";
      case "completed":
        return "text-yellow-500 border-yellow-500";
      case "failed":
        return "text-gray-500 border-gray-500";
      case "cancelled":
        return "text-gray-500 border-gray-500";
      default:
        return "text-gray-500 border-gray-500";
    }
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleString();
  };

  if (loading) {
    return (
      <div className="flex justify-center items-center h-64">
        <div className="arcade-text text-primary-500 animate-pulse">
          LOADING...
        </div>
      </div>
    );
  }

  if (error) {
    return <ErrorMessage error={error} className="mb-6" />;
  }

  return (
    <div className="w-full max-w-4xl mx-auto">
      <h2 className="arcade-text text-2xl mb-6 text-center text-primary-500">
        RUNS
      </h2>
      {intents.length === 0 ? (
        <p className="arcade-text text-gray-500 text-center">
          NO RECORDS FOUND
        </p>
      ) : (
        <div className="arcade-container">
          {intents.map((intent, index) => (
            <IntentTile
              key={intent.id}
              intent={intent}
              index={index}
              offset={(currentPage - 1) * ITEMS_PER_PAGE}
              label="RUN"
            />
          ))}
        </div>
      )}

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="flex justify-center space-x-4 mt-6">
          <button
            onClick={handlePrevPage}
            disabled={currentPage === 1}
            className="arcade-btn-sm border-green-400 text-green-400 hover:bg-green-400 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            PREV
          </button>
          <span className="arcade-text text-green-400">
            {currentPage} / {totalPages}
          </span>
          <button
            onClick={handleNextPage}
            disabled={currentPage >= totalPages}
            className="arcade-btn-sm border-green-400 text-green-400 hover:bg-green-400 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            NEXT
          </button>
        </div>
      )}
    </div>
  );
};

export default IntentList;
