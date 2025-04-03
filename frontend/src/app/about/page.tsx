'use client';

export default function About() {
  return (
    <main className="flex min-h-screen flex-col items-center p-8 relative overflow-hidden">
      {/* Retro grid background */}
      <div className="fixed inset-0 bg-[linear-gradient(transparent_1px,_transparent_1px),_linear-gradient(90deg,_transparent_1px,_transparent_1px)] bg-[size:4rem_4rem] [mask-image:radial-gradient(ellipse_80%_50%_at_50%_0%,_#000_70%,_transparent_100%)] opacity-20" />
      
      {/* Neon glow effects */}
      <div className="fixed inset-0 bg-[radial-gradient(circle_at_50%_50%,_rgba(255,255,0,0.1)_0%,_transparent_50%)]" />
      
      <div className="z-10 max-w-5xl w-full relative">
        <div className="text-center mb-12">
          <h1 className="arcade-text text-3xl text-primary-500 relative mb-4">
            <span className="absolute inset-0 blur-sm opacity-50">ABOUT SPEEDRUN</span>
            ABOUT SPEEDRUN
          </h1>
          <p className="arcade-text text-xl text-primary-300 relative">
            Intent-based token transfers backed by ZetaChain
          </p>
        </div>

        <div className="mt-8 space-y-8 relative">
          {/* ARCHITECTURE Section (moved up) */}
          <div className="arcade-container border-primary-500 relative group">
            <div className="absolute inset-0 bg-primary-500/10 blur-sm group-hover:bg-primary-500/20 transition-all duration-300" />
            <div className="relative">
              <h3 className="arcade-text text-lg mb-4 text-primary-500 text-center">ARCHITECTURE</h3>
              <div className="space-y-4">
                <p className="arcade-text text-xs text-gray-300">
                  SPEEDRUN is powered by ZetaChain's cross-chain intent settlement protocol, enabling seamless token transfers across multiple blockchains.
                </p>
                <p className="arcade-text text-xs text-gray-300">
                  The platform uses an intent-based architecture where users create transfer intents that get fulfilled by "speedrunners" - liquidity providers who compete to execute transfers as quickly as possible.
                </p>
                <p className="arcade-text text-xs text-gray-300">
                  By leveraging ZetaChain's interoperability features, SPEEDRUN eliminates the need for bridges or wrapped tokens, making cross-chain transfers simple and efficient.
                </p>
              </div>
            </div>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
            <div className="arcade-container border-yellow-500 relative group">
              <div className="absolute inset-0 bg-yellow-500/10 blur-sm group-hover:bg-yellow-500/20 transition-all duration-300" />
              <div className="relative">
                <h3 className="arcade-text text-lg mb-2 text-yellow-500">FAST</h3>
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
      </div>
    </main>
  );
} 