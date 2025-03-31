import Link from 'next/link';

export default function Home() {
  return (
    <main className="flex min-h-screen flex-col items-center justify-between p-24">
      <div className="z-10 max-w-5xl w-full items-center justify-between font-mono text-sm">
        <div className="text-center">
          <h1 className="text-4xl font-bold mb-8">ZetaFast</h1>
          <p className="text-xl mb-8">Cross-chain USDC Transfer Service</p>
          <div className="space-y-4">
            <p className="text-gray-600">
              Seamlessly transfer USDC between different blockchain networks with our fast and secure service.
            </p>
            <div className="flex justify-center space-x-4 mt-8">
              <Link
                href="/create"
                className="bg-blue-600 text-white hover:bg-blue-700 px-6 py-3 rounded-md text-sm font-medium"
              >
                Create Transfer
              </Link>
              <Link
                href="/intents"
                className="bg-white text-gray-700 hover:bg-gray-50 px-6 py-3 rounded-md text-sm font-medium border"
              >
                View Transfers
              </Link>
            </div>
          </div>
        </div>

        <div className="mt-16 grid grid-cols-1 md:grid-cols-3 gap-8">
          <div className="p-6 border rounded-lg">
            <h3 className="text-lg font-semibold mb-2">Fast Transfers</h3>
            <p className="text-gray-600">
              Quick and efficient cross-chain transfers powered by ZetaChain.
            </p>
          </div>
          <div className="p-6 border rounded-lg">
            <h3 className="text-lg font-semibold mb-2">Secure</h3>
            <p className="text-gray-600">
              Built with security in mind, ensuring your assets are safe.
            </p>
          </div>
          <div className="p-6 border rounded-lg">
            <h3 className="text-lg font-semibold mb-2">Easy to Use</h3>
            <p className="text-gray-600">
              Simple interface for creating and managing your transfers.
            </p>
          </div>
        </div>
      </div>
    </main>
  );
} 