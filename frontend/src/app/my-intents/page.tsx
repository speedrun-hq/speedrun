"use client";

import React from "react";
import dynamic from "next/dynamic";
import { ConnectWallet } from "@/components/ConnectWallet";

// Make sure to use a relative import path, which is more reliable
const UserIntentList = dynamic(
  () => import("../../components/UserIntentList"),
  {
    ssr: false,
    loading: () => (
      <div className="w-full max-w-4xl mx-auto mt-8">
        <div className="flex justify-center items-center h-64">
          <div className="arcade-text text-primary-500 animate-pulse">
            LOADING COMPONENT...
          </div>
        </div>
      </div>
    ),
  },
);

const MyIntentsPage: React.FC = () => {
  return (
    <main className="w-full max-w-6xl mx-auto px-4 py-12">
      <div className="flex flex-col items-center justify-center mb-8">
        <h1 className="arcade-text text-3xl text-primary-500 mb-4">
          MY TRANSFERS
        </h1>
        <p className="arcade-text text-sm text-gray-400 mb-6 text-center">
          VIEW ALL TRANSFERS YOU HAVE SENT OR RECEIVED
        </p>
      </div>

      {/* Use the dynamic component directly, with its own loading state */}
      <UserIntentList />
    </main>
  );
};

export default MyIntentsPage;
