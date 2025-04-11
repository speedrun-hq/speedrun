import React from "react";
import { Intent } from "@/types";
import Link from "next/link";

interface IntentTileProps {
  intent: Intent;
  index: number;
  offset: number;
  showSender?: boolean;
  label?: string;
}

export const getStatusColor = (status: string) => {
  switch (status.toLowerCase()) {
    case "pending":
    case "processing":
    case "settled":
      return "text-primary-500 border-primary-500"; // green for all processing states
    case "completed":
    case "fulfilled":
      return "text-yellow-500 border-yellow-500"; // yellow for completed states
    case "failed":
    case "cancelled":
    default:
      return "text-gray-500 border-gray-500"; // gray for failed states
  }
};

const IntentTile: React.FC<IntentTileProps> = ({
  intent,
  index,
  offset,
  showSender = false,
  label = "RUN",
}) => {
  return (
    <div className="arcade-card relative border-yellow-500/30 hover:border-yellow-500 transition-all hover:bg-yellow-500/20 hover:shadow-[0_0_15px_rgba(234,179,8,0.3)] cursor-pointer">
      <span
        className={`arcade-status ${getStatusColor(intent.status)} border-2 absolute top-4 right-4 z-10`}
      >
        {intent.status}
      </span>
      <Link href={`/intent/${intent.id}`} className="block">
        <div className="space-y-3">
          <div className="flex items-center space-x-2">
            <div className="flex items-center space-x-2">
              <span className="arcade-text text-sm text-primary-500">
                {label}
              </span>
              <span className="arcade-text text-sm text-primary-500">
                #{index + 1 + offset}
              </span>
            </div>
          </div>
          <div className="space-y-1">
            <div className="flex flex-col">
              <span className="arcade-text text-xs text-yellow-500">
                INTENT ID
              </span>
              <span
                className="arcade-text text-[10px] text-gray-300 break-all"
                style={{ textTransform: "none" }}
              >
                {intent.id}
              </span>
            </div>
            <div className="flex flex-col">
              <span className="arcade-text text-xs text-yellow-500">
                ROUTE
              </span>
              <span className="arcade-text text-xs text-gray-300">
                CHAIN{" "}
                <span className="text-gray-300">
                  {intent.source_chain}
                </span>{" "}
                → CHAIN{" "}
                <span className="text-gray-300">
                  {intent.destination_chain}
                </span>
              </span>
            </div>
            <div className="flex flex-col">
              <span className="arcade-text text-xs text-yellow-500">
                TOKEN
              </span>
              <span
                className="arcade-text text-[10px] text-gray-300 break-all"
                style={{ textTransform: "none" }}
              >
                {intent.token}
              </span>
            </div>
            <div className="flex flex-col">
              <span className="arcade-text text-xs text-yellow-500">
                AMOUNT
              </span>
              <span className="arcade-text text-xs text-gray-300">
                {intent.amount}
              </span>
            </div>
            {showSender ? (
              <div className="flex flex-col">
                <span className="arcade-text text-xs text-yellow-500">
                  SENDER
                </span>
                <span
                  className="arcade-text text-[10px] text-gray-300 break-all"
                  style={{ textTransform: "none" }}
                >
                  {intent.sender}
                </span>
              </div>
            ) : (
              <div className="flex flex-col">
                <span className="arcade-text text-xs text-yellow-500">
                  RECIPIENT
                </span>
                <span
                  className="arcade-text text-[10px] text-gray-300 break-all"
                  style={{ textTransform: "none" }}
                >
                  {intent.recipient}
                </span>
              </div>
            )}
            {intent.created_at && (
              <div className="flex flex-col">
                <span className="arcade-text text-xs text-yellow-500">
                  CREATED AT
                </span>
                <span className="arcade-text text-xs text-gray-300">
                  {new Date(intent.created_at).toLocaleString()}
                </span>
              </div>
            )}
          </div>
        </div>
      </Link>
    </div>
  );
};

export default IntentTile; 