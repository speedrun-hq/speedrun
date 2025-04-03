'use client';

import { useState } from 'react';
import { CreateNewIntentWrapper } from '@/components/CreateNewIntentWrapper';

export default function Home() {
  const [showMore, setShowMore] = useState(false);

  return (
    <main className="flex min-h-screen flex-col items-center p-8 relative overflow-hidden">
      {/* Retro grid background */}
      <div className="fixed inset-0 bg-[linear-gradient(transparent_1px,_transparent_1px),_linear-gradient(90deg,_transparent_1px,_transparent_1px)] bg-[size:4rem_4rem] [mask-image:radial-gradient(ellipse_80%_50%_at_50%_0%,_#000_70%,_transparent_100%)] opacity-20" />
      
      {/* Neon glow effects */}
      <div className="fixed inset-0 bg-[radial-gradient(circle_at_50%_50%,_rgba(255,255,0,0.1)_0%,_transparent_50%)]" />
      
      <div className="z-10 max-w-5xl w-full relative">
        <div className="text-center mb-12">
          <p className="arcade-text text-xl text-primary-500 relative">
            <span className="absolute inset-0 blur-sm opacity-50">Cheap and fast cross-chain transfers</span>
            Cheap and fast cross-chain transfers
          </p>
        </div>

        <div className="mb-8 relative">
          <div className="absolute inset-0 bg-[radial-gradient(circle_at_50%_50%,_rgba(0,255,255,0.05)_0%,_transparent_70%)] blur-2xl" />
          <div className="absolute inset-0 bg-[linear-gradient(45deg,_transparent_0%,_rgba(0,255,255,0.02)_50%,_transparent_100%)] animate-pulse" />
          <CreateNewIntentWrapper />
        </div>

        <div className="text-center">
          <button
            onClick={() => setShowMore(!showMore)}
            className="arcade-btn border-yellow-500 text-yellow-500 hover:bg-yellow-500 relative group"
          >
            <span className="absolute inset-0 bg-yellow-500/20 blur-sm group-hover:bg-yellow-500/30 transition-all duration-300" />
            <span className="relative">
              {showMore ? 'SHOW LESS' : 'ABOUT SPEEDRUN'}
            </span>
          </button>
        </div>

        {showMore && (
          <div className="mt-8 space-y-8 relative">
            <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
              <div className="arcade-container border-yellow-500 relative group">
                <div className="absolute inset-0 bg-yellow-500/10 blur-sm group-hover:bg-yellow-500/20 transition-all duration-300" />
                <div className="relative">
                  <h3 className="arcade-text text-lg mb-2 text-yellow-500">Fast Transfers</h3>
                  <p className="arcade-text text-xs text-gray-300">
                    Speedrunners race to fulfill your transfers quickly for competitive rewards.
                  </p>
                </div>
              </div>
              <div className="arcade-container border-primary-500 relative group">
                <div className="absolute inset-0 bg-primary-500/10 blur-sm group-hover:bg-primary-500/20 transition-all duration-300" />
                <div className="relative">
                  <h3 className="arcade-text text-lg mb-2 text-primary-500">Secure</h3>
                  <p className="arcade-text text-xs text-gray-300">
                    Built with security in mind, ensuring your assets are safe while speedrunners compete.
                  </p>
                </div>
              </div>
              <div className="arcade-container relative group">
                <div className="absolute inset-0 bg-primary-500/10 blur-sm group-hover:bg-primary-500/20 transition-all duration-300" />
                <div className="relative">
                  <h3 className="arcade-text text-lg mb-2 text-primary-500">Easy to Use</h3>
                  <p className="arcade-text text-xs text-gray-300">
                    Set your transfer fee to attract speedrunners and get faster execution.
                  </p>
                </div>
              </div>
            </div>

            <div className="arcade-container border-yellow-500 relative group">
              <div className="absolute inset-0 bg-yellow-500/10 blur-sm group-hover:bg-yellow-500/20 transition-all duration-300" />
              <div className="relative">
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
          </div>
        )}
      </div>
      <div className="z-10 pt-8 text-center">
        <p className="arcade-text text-sm text-primary-300 relative">
          <span className="absolute inset-0 blur-sm opacity-50">Backed by ZetaChain</span>
          Backed by ZetaChain
        </p>
      </div>
    </main>
  );
} 