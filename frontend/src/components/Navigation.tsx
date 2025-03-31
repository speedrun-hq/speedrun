import Link from 'next/link';

const Navigation = () => {
  return (
    <nav className="bg-white shadow-sm">
      <div className="container mx-auto px-4">
        <div className="flex justify-between h-16 items-center">
          <Link href="/" className="text-xl font-bold text-gray-800">
            ZetaFast
          </Link>
          <div className="flex space-x-4">
            <Link
              href="/intents"
              className="text-gray-600 hover:text-gray-900 px-3 py-2 rounded-md text-sm font-medium"
            >
              Intents
            </Link>
            <Link
              href="/create"
              className="bg-blue-600 text-white hover:bg-blue-700 px-4 py-2 rounded-md text-sm font-medium"
            >
              Create Intent
            </Link>
          </div>
        </div>
      </div>
    </nav>
  );
};

export default Navigation; 