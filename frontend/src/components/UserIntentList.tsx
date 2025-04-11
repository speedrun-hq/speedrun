"use client";

import React, { useEffect, useState } from "react";
import { Intent, PaginationParams } from "@/types";
import { apiService } from "@/services/api";
import ErrorMessage from "@/components/ErrorMessage";
import IntentTile from "@/components/IntentTile";
import { useAccount } from "wagmi";

const ITEMS_PER_PAGE = 10;

type ViewMode = "sent" | "received";

function UserIntentList() {
  const { address, isConnected } = useAccount();
  const [viewMode, setViewMode] = useState<ViewMode>("sent");
  const [sentIntents, setSentIntents] = useState<Intent[]>([]);
  const [receivedIntents, setReceivedIntents] = useState<Intent[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<unknown>(null);
  const [currentPage, setCurrentPage] = useState(1);
  const [sentTotalPages, setSentTotalPages] = useState(1);
  const [receivedTotalPages, setReceivedTotalPages] = useState(1);
  const [sentTotalCount, setSentTotalCount] = useState(0);
  const [receivedTotalCount, setReceivedTotalCount] = useState(0);

  useEffect(() => {
    const fetchUserIntents = async () => {
      if (!address || !isConnected) return;

      try {
        setLoading(true);
        setError(null);

        const pagination: PaginationParams = {
          page: currentPage,
          page_size: ITEMS_PER_PAGE,
        };

        if (viewMode === "sent") {
          // Fetch sent intents
          const sentResponse = await apiService.listIntentsBySender(
            address,
            pagination,
          );
          setSentIntents(sentResponse.data || []);
          setSentTotalPages(sentResponse.total_pages);
          setSentTotalCount(sentResponse.total_count);
        } else {
          // Fetch received intents
          const receivedResponse = await apiService.listIntentsByRecipient(
            address,
            pagination,
          );
          setReceivedIntents(receivedResponse.data || []);
          setReceivedTotalPages(receivedResponse.total_pages);
          setReceivedTotalCount(receivedResponse.total_count);
        }
      } catch (err) {
        console.error("Error fetching intents:", err);
        setError(err);
      } finally {
        setLoading(false);
      }
    };

    if (isConnected && address) {
      fetchUserIntents();
    }
  }, [address, isConnected, viewMode, currentPage]);

  // Reset page when address or view mode changes
  useEffect(() => {
    setCurrentPage(1);
  }, [address, viewMode]);

  const currentIntents = viewMode === "sent" ? sentIntents : receivedIntents;
  const totalPages = viewMode === "sent" ? sentTotalPages : receivedTotalPages;
  const totalCount = viewMode === "sent" ? sentTotalCount : receivedTotalCount;

  const handlePrevPage = () => {
    setCurrentPage((prev) => Math.max(1, prev - 1));
  };

  const handleNextPage = () => {
    setCurrentPage((prev) => (prev < totalPages ? prev + 1 : prev));
  };

  if (!isConnected || !address) {
    return (
      <div className="flex justify-center items-center h-64">
        <div className="arcade-text text-primary-500">
          CONNECT WALLET TO VIEW YOUR INTENTS
        </div>
      </div>
    );
  }

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
      <div className="flex justify-center mb-6 space-x-4">
        <button
          onClick={() => setViewMode("sent")}
          className={`arcade-btn-sm border-green-400 text-green-400 hover:bg-green-400 ${viewMode === "sent" ? "bg-primary-500 text-black" : ""}`}
        >
          SENT
        </button>
        <button
          onClick={() => setViewMode("received")}
          className={`arcade-btn-sm border-green-400 text-green-400 hover:bg-green-400 ${viewMode === "received" ? "bg-primary-500 text-black" : ""}`}
        >
          RECEIVED
        </button>
      </div>

      {currentIntents.length === 0 ? (
        <p className="arcade-text text-gray-500 text-center">
          NO {viewMode.toUpperCase()} INTENTS FOUND
        </p>
      ) : (
        <div className="arcade-container">
          {currentIntents.map((intent, index) => (
            <IntentTile
              key={intent.id}
              intent={intent}
              index={index}
              offset={(currentPage - 1) * ITEMS_PER_PAGE}
              label="INTENT"
              showSender={viewMode === "received"}
            />
          ))}
        </div>
      )}

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="flex justify-center items-center space-x-4 mt-6">
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
}

export default UserIntentList;
