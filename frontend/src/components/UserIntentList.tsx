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
  const [pageInput, setPageInput] = useState("");

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

  const handlePageInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const value = e.target.value;
    if (value === "" || /^\d+$/.test(value)) {
      setPageInput(value);
    }
  };

  const handlePageInputSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const pageNum = parseInt(pageInput);
    if (pageNum >= 1 && pageNum <= totalPages) {
      setCurrentPage(pageNum);
      setPageInput("");
    }
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
      <div className="flex justify-between items-center mb-6">
        <div className="flex space-x-4">
          <button
            onClick={() => setViewMode("sent")}
            className={`arcade-btn-sm ${
              viewMode === "sent"
                ? "bg-yellow-500 text-black"
                : "border-yellow-500 text-yellow-500"
            }`}
          >
            SENT
          </button>
          <button
            onClick={() => setViewMode("received")}
            className={`arcade-btn-sm ${
              viewMode === "received"
                ? "bg-yellow-500 text-black"
                : "border-yellow-500 text-yellow-500"
            }`}
          >
            RECEIVED
          </button>
        </div>
        <div className="flex items-center space-x-4">
          <form onSubmit={handlePageInputSubmit} className="flex items-center">
            <input
              type="text"
              value={pageInput}
              onChange={handlePageInputChange}
              placeholder={`1-${totalPages}`}
              className="w-20 px-2 py-1 bg-black border-2 border-yellow-500 rounded text-yellow-500 arcade-text text-xs focus:outline-none focus:border-yellow-400"
            />
            <button
              type="submit"
              className="ml-2 arcade-btn-sm border-yellow-500 text-yellow-500"
            >
              GO
            </button>
          </form>
          <div className="flex items-center space-x-2">
            <button
              onClick={handlePrevPage}
              disabled={currentPage === 1}
              className="arcade-btn-sm border-yellow-500 text-yellow-500 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              ←
            </button>
            <span className="arcade-text text-yellow-500 text-xs">
              {currentPage} / {totalPages}
            </span>
            <button
              onClick={handleNextPage}
              disabled={currentPage === totalPages}
              className="arcade-btn-sm border-yellow-500 text-yellow-500 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              →
            </button>
          </div>
        </div>
      </div>

      <div className="space-y-4">
        {currentIntents.map((intent, index) => (
          <IntentTile
            key={intent.id}
            intent={intent}
            index={totalCount - ((currentPage - 1) * ITEMS_PER_PAGE + index)}
            offset={0}
            showSender={viewMode === "received"}
            label={viewMode === "sent" ? "SENT" : "RECEIVED"}
          />
        ))}
      </div>
    </div>
  );
}

export default UserIntentList;
