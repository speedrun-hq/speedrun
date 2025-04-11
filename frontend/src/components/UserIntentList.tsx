"use client";

import React, { useEffect, useState } from "react";
import { Intent } from "@/types";
import { apiService } from "@/services/api";
import ErrorMessage from "@/components/ErrorMessage";
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
  const [offset, setOffset] = useState(0);
  const [hasMore, setHasMore] = useState(true);

  useEffect(() => {
    const fetchUserIntents = async () => {
      if (!address || !isConnected) return;

      try {
        setLoading(true);
        setError(null);

        // Fetch sent intents
        const sentData = await apiService.listIntentsBySender(address);
        setSentIntents(sentData || []);

        // Fetch received intents
        const receivedData = await apiService.listIntentsByRecipient(address);
        setReceivedIntents(receivedData || []);

        // Update pagination based on current view
        const currentData = viewMode === "sent" ? sentData : receivedData;
        setHasMore((currentData?.length || 0) > offset + ITEMS_PER_PAGE);
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
  }, [address, isConnected, viewMode]);

  // Reset offset when address or view mode changes
  useEffect(() => {
    setOffset(0);
  }, [address, viewMode]);

  // Update pagination when view mode changes
  useEffect(() => {
    const currentData = viewMode === "sent" ? sentIntents : receivedIntents;
    setHasMore((currentData?.length || 0) > ITEMS_PER_PAGE);
  }, [viewMode, sentIntents, receivedIntents]);

  // Update hasMore flag when offset changes
  useEffect(() => {
    const currentData = viewMode === "sent" ? sentIntents : receivedIntents;
    setHasMore((currentData?.length || 0) > offset + ITEMS_PER_PAGE);
  }, [offset, viewMode, sentIntents, receivedIntents]);

  const currentIntents = viewMode === "sent" ? sentIntents : receivedIntents;
  const displayedIntents = currentIntents.slice(
    offset,
    offset + ITEMS_PER_PAGE,
  );

  const getStatusColor = (status: string) => {
    switch (status.toLowerCase()) {
      case "pending":
        return "text-primary-500 border-primary-500";
      case "fulfilled":
        return "text-secondary-500 border-secondary-500";
      case "processing":
        return "text-yellow-500 border-yellow-500";
      case "settled":
        return "text-green-500 border-green-500";
      case "failed":
        return "text-accent-500 border-accent-500";
      default:
        return "text-gray-500 border-gray-500";
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
      <div className="flex justify-center mb-6 space-x-4">
        <button
          onClick={() => setViewMode("sent")}
          className={`arcade-btn ${viewMode === "sent" ? "bg-primary-500 text-black" : ""}`}
        >
          SENT
        </button>
        <button
          onClick={() => setViewMode("received")}
          className={`arcade-btn ${viewMode === "received" ? "bg-primary-500 text-black" : ""}`}
        >
          RECEIVED
        </button>
      </div>

      {displayedIntents.length === 0 ? (
        <p className="arcade-text text-gray-500 text-center">
          NO {viewMode.toUpperCase()} INTENTS FOUND
        </p>
      ) : (
        <div className="arcade-container">
          {displayedIntents.map((intent, index) => (
            <div key={intent.id} className="arcade-card relative">
              <span
                className={`arcade-status ${getStatusColor(intent.status)} border-2 absolute top-4 right-4`}
              >
                {intent.status}
              </span>
              <div className="space-y-3">
                <div className="flex items-center space-x-2">
                  <span className="arcade-text text-sm text-yellow-500">
                    INTENT
                  </span>
                  <span className="arcade-text text-sm text-cyan-500">
                    #{index + 1 + offset}
                  </span>
                </div>
                <div className="space-y-1">
                  <div className="flex flex-col">
                    <span className="arcade-text text-xs text-gray-500">
                      INTENT ID
                    </span>
                    <span
                      className="arcade-text text-xs text-magenta-500 break-all font-mono"
                      style={{ textTransform: "none" }}
                    >
                      {intent.id}
                    </span>
                  </div>
                  <div className="flex flex-col">
                    <span className="arcade-text text-xs text-gray-500">
                      ROUTE
                    </span>
                    <span className="arcade-text text-xs text-cyan-500">
                      CHAIN{" "}
                      <span className="text-orange-500">
                        {intent.source_chain}
                      </span>{" "}
                      → CHAIN{" "}
                      <span className="text-orange-500">
                        {intent.destination_chain}
                      </span>
                    </span>
                  </div>
                  <div className="flex flex-col">
                    <span className="arcade-text text-xs text-gray-500">
                      TOKEN
                    </span>
                    <span
                      className="arcade-text text-xs text-yellow-500 break-all font-mono"
                      style={{ textTransform: "none" }}
                    >
                      {intent.token}
                    </span>
                  </div>
                  <div className="flex flex-col">
                    <span className="arcade-text text-xs text-gray-500">
                      AMOUNT
                    </span>
                    <span className="arcade-text text-xs text-primary-500">
                      {intent.amount}
                    </span>
                  </div>
                  {viewMode === "sent" ? (
                    <div className="flex flex-col">
                      <span className="arcade-text text-xs text-gray-500">
                        RECIPIENT
                      </span>
                      <span
                        className="arcade-text text-xs text-magenta-500 break-all font-mono"
                        style={{ textTransform: "none" }}
                      >
                        {intent.recipient}
                      </span>
                    </div>
                  ) : (
                    <div className="flex flex-col">
                      <span className="arcade-text text-xs text-gray-500">
                        SENDER
                      </span>
                      <span
                        className="arcade-text text-xs text-magenta-500 break-all font-mono"
                        style={{ textTransform: "none" }}
                      >
                        {intent.sender}
                      </span>
                    </div>
                  )}
                  <div className="flex flex-col">
                    <span className="arcade-text text-xs text-gray-500">
                      CREATED AT
                    </span>
                    <span className="arcade-text text-xs text-green-500">
                      {new Date(intent.created_at).toLocaleString()}
                    </span>
                  </div>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Pagination */}
      {currentIntents.length > 0 && (
        <div className="flex justify-center items-center space-x-4 mt-6">
          <button
            onClick={() => setOffset((o) => Math.max(0, o - ITEMS_PER_PAGE))}
            disabled={offset === 0}
            className="arcade-btn disabled:opacity-50 disabled:cursor-not-allowed"
          >
            PREV
          </button>

          <div className="arcade-text text-primary-500 px-2">
            {offset + 1}-
            {Math.min(offset + displayedIntents.length, currentIntents.length)}{" "}
            OF {currentIntents.length}
          </div>

          <button
            onClick={() => setOffset((o) => o + ITEMS_PER_PAGE)}
            disabled={offset + ITEMS_PER_PAGE >= currentIntents.length}
            className="arcade-btn disabled:opacity-50 disabled:cursor-not-allowed"
          >
            NEXT
          </button>
        </div>
      )}
    </div>
  );
}

export default UserIntentList;
