'use client';

import { CreateNewIntentWrapper } from '@/components/CreateNewIntentWrapper';

export default function Home() {
  return (
    <main className="flex min-h-[calc(100vh-150px)] flex-col items-center p-4 md:p-8 pt-12 md:pt-16 relative overflow-hidden">
      {/* Retro grid background */}
      <div className="fixed inset-0 bg-[linear-gradient(transparent_1px,_transparent_1px),_linear-gradient(90deg,_transparent_1px,_transparent_1px)] bg-[size:4rem_4rem] [mask-image:radial-gradient(ellipse_80%_50%_at_50%_0%,_#000_70%,_transparent_100%)] opacity-20" />
      
      {/* Neon glow effects */}
      <div className="fixed inset-0 bg-[radial-gradient(circle_at_50%_50%,_rgba(255,255,0,0.1)_0%,_transparent_50%)]" />
      
      <div className="z-10 max-w-5xl w-full relative flex flex-col items-center">
        <div className="text-center mb-10 md:mb-16">
          <p className="arcade-text text-lg md:text-xl text-primary-500 relative">
            <span className="absolute inset-0 blur-sm opacity-50">Blazing-Fast Cross-Chain Transfers</span>
            Blazing-Fast Cross-Chain Transfers
          </p>
        </div>

        <div className="relative w-full max-w-2xl mx-auto flex-grow flex items-center justify-center">
          <div className="w-full">
            <div className="absolute inset-0 bg-[radial-gradient(circle_at_50%_50%,_rgba(0,255,255,0.05)_0%,_transparent_70%)] blur-2xl" />
            <div className="absolute inset-0 bg-[linear-gradient(45deg,_transparent_0%,_rgba(0,255,255,0.02)_50%,_transparent_100%)] animate-pulse" />
            <CreateNewIntentWrapper />
          </div>
        </div>
      </div>
    </main>
  );
} 