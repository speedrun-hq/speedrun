'use client';

import { useState } from 'react';
import CreateIntentForm from '@/components/CreateIntentForm';

export default function Home() {
  const [showMore, setShowMore] = useState(false);

  return (
    <main className="flex min-h-screen flex-col items-center p-8">
      <div className="z-10 max-w-5xl w-full">
        <div className="text-center mb-12">
          <h1 className="arcade-text text-4xl mb-4 text-yellow-500">SPEEDRUN</h1>
          <p className="arcade-text text-xl text-primary-500">
            Intent-based token transfers backed by ZetaChain
          </p>
        </div>

        <div className="mb-8">
          <CreateIntentForm />
        </div>

        <div className="text-center">
          <button
            onClick={() => setShowMore(!showMore)}
            className="arcade-btn border-yellow-500 text-yellow-500 hover:bg-yellow-500"
          >
            {showMore ? 'SHOW LESS' : 'ABOUT SPEEDRUN'}
          </button>
        </div>

        {showMore && (
          <div className="mt-8 space-y-8">
            <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
              <div className="arcade-container border-yellow-500">
                <h3 className="arcade-text text-lg mb-2 text-yellow-500">Fast Transfers</h3>
                <p className="arcade-text text-xs text-gray-300">
                  Speedrunners race to fulfill your transfers quickly for competitive rewards.
                </p>
              </div>
              <div className="arcade-container border-primary-500">
                <h3 className="arcade-text text-lg mb-2 text-primary-500">Secure</h3>
                <p className="arcade-text text-xs text-gray-300">
                  Built with security in mind, ensuring your assets are safe while speedrunners compete.
                </p>
              </div>
              <div className="arcade-container">
                <h3 className="arcade-text text-lg mb-2 text-primary-500">Easy to Use</h3>
                <p className="arcade-text text-xs text-gray-300">
                  Set your transfer fee to attract speedrunners and get faster execution.
                </p>
              </div>
            </div>

            <div className="arcade-container border-yellow-500">
              <h3 className="arcade-text text-lg mb-4 text-yellow-500 text-center">HOW IT WORKS</h3>
              <div className="space-y-4">
                <p className="arcade-text text-xs text-gray-300">
                  1. CREATE A TRANSFER WITH YOUR DESIRED FEE
                </p>
                <p className="arcade-text text-xs text-gray-300">
                  2. SPEEDRUNNERS COMPETE TO FULFILL YOUR TRANSFER
                </p>
                <p className="arcade-text text-xs text-gray-300">
                  3. FASTEST SPEEDRUNNER WINS THE FEE REWARD
                </p>
                <p className="arcade-text text-xs text-yellow-500">
                  HIGHER FEES ATTRACT MORE SPEEDRUNNERS = FASTER TRANSFERS!
                </p>
              </div>
            </div>
          </div>
        )}
      </div>
    </main>
  );
} 