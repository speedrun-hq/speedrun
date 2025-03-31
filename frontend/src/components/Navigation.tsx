import Link from 'next/link';

const Navigation = () => {
  return (
    <nav className="bg-black border-b-4 border-primary-500">
      <div className="container mx-auto px-4">
        <div className="flex justify-between h-16 items-center">
          <Link href="/" className="arcade-text text-xl text-primary-500 hover:text-primary-400">
            ZETAFAST
          </Link>
          <div className="flex space-x-4">
            <Link
              href="/intents"
              className="arcade-text text-primary-500 hover:text-primary-400 px-3 py-2 text-sm"
            >
              HIGH SCORES
            </Link>
            <Link
              href="/create"
              className="arcade-btn"
            >
              NEW GAME
            </Link>
          </div>
        </div>
      </div>
    </nav>
  );
};

export default Navigation; 