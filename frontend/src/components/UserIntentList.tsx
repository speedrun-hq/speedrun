"use client";

import React, { useEffect, useState } from "react";
import { Intent } from "@/types";
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

      {displayedIntents.length === 0 ? (
        <p className="arcade-text text-gray-500 text-center">
          NO {viewMode.toUpperCase()} INTENTS FOUND
        </p>
      ) : (
        <div className="arcade-container">
          {displayedIntents.map((intent, index) => (
            <IntentTile 
              key={intent.id}
              intent={intent}
              index={index}
              offset={offset}
              label="INTENT"
              showSender={viewMode === "received"}
            />
          ))}
        </div>
      )}

      {/* Pagination */}
      {currentIntents.length > 0 && (
        <div className="flex justify-center items-center space-x-4 mt-6">
          <button
            onClick={() => setOffset((o) => Math.max(0, o - ITEMS_PER_PAGE))}
            disabled={offset === 0}
            className="arcade-btn-sm border-green-400 text-green-400 hover:bg-green-400 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            PREV
          </button>
          <button
            onClick={() => setOffset((o) => o + ITEMS_PER_PAGE)}
            disabled={!hasMore}
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
