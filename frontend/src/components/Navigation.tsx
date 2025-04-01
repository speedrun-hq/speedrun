import Link from 'next/link';
import { ConnectWallet } from './ConnectWallet';

const Navigation = () => {
  return (
    <nav className="bg-black border-b-4 border-primary-500">
      <div className="container mx-auto px-4">
        <div className="flex justify-between h-16 items-center">
          <Link href="/" className="arcade-text text-xl text-primary-500 hover:text-primary-400">
            SPEEDRUN
          </Link>
          <div className="flex space-x-4 items-center">
            <Link
              href="/intents"
              className="arcade-btn border-yellow-500 text-yellow-500 hover:bg-yellow-500"
            >
              RUNS
            </Link>
            <Link
              href="/create"
              className="arcade-btn"
            >
              NEW RUN
            </Link>
            <ConnectWallet />
          </div>
        </div>
      </div>
    </nav>
  );
};

export default Navigation; 