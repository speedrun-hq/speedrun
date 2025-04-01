import Link from 'next/link';

interface LayoutProps {
  children: React.ReactNode;
}

export default function Layout({ children }: LayoutProps) {
  return (
    <div className="min-h-screen bg-black">
      <nav className="border-b-4 border-primary-500 bg-black">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between h-16">
            <div className="flex">
              <div className="flex-shrink-0 flex items-center">
                <Link href="/" className="arcade-text text-xl text-primary-500">
                  SPEEDRUN
                </Link>
              </div>
              <div className="hidden sm:ml-6 sm:flex sm:space-x-8">
                <Link
                  href="/"
                  className="arcade-text border-transparent text-primary-500 hover:text-white hover:border-primary-500 inline-flex items-center px-1 pt-1 border-b-2 text-xs"
                >
                  HOME
                </Link>
                <Link
                  href="/intents"
                  className="arcade-text border-transparent text-primary-500 hover:text-white hover:border-primary-500 inline-flex items-center px-1 pt-1 border-b-2 text-xs"
                >
                  INTENTS
                </Link>
                <Link
                  href="/create"
                  className="arcade-text border-transparent text-primary-500 hover:text-white hover:border-primary-500 inline-flex items-center px-1 pt-1 border-b-2 text-xs"
                >
                  CREATE
                </Link>
              </div>
            </div>
          </div>
        </div>
      </nav>

      <main className="max-w-7xl mx-auto py-6 sm:px-6 lg:px-8">
        <div className="px-4 py-6 sm:px-0">
          {children}
        </div>
      </main>
    </div>
  );
} 