import Link from 'next/link';

export default function Home() {
  return (
    <main className="flex min-h-screen flex-col items-center justify-between p-24">
      <div className="z-10 max-w-5xl w-full items-center justify-between">
        <div className="text-center">
          <h1 className="arcade-text text-4xl mb-8 text-yellow-500">ZetaFast</h1>
          <p className="arcade-text text-xl mb-8 text-primary-500">Cross-chain Token Transfer Service</p>
          <div className="space-y-4">
            <p className="arcade-text text-sm text-gray-300">
              Seamlessly transfer tokens between different blockchain networks with our fast and secure service.
            </p>
            <p className="arcade-text text-sm text-yellow-500 mt-4">
              SPEEDRUNNERS COMPETE TO ACCELERATE YOUR TRANSFERS FOR REWARDS!
            </p>
            <div className="flex justify-center space-x-4 mt-8">
              <Link
                href="/create"
                className="arcade-btn border-yellow-500 text-yellow-500 hover:bg-yellow-500"
              >
                CREATE TRANSFER
              </Link>
              <Link
                href="/intents"
                className="arcade-btn"
              >
                VIEW TRANSFERS
              </Link>
            </div>
          </div>
        </div>

        <div className="mt-16 grid grid-cols-1 md:grid-cols-3 gap-8">
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

        <div className="mt-16 arcade-container border-yellow-500">
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
    </main>
  );
} 