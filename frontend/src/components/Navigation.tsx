import Link from 'next/link';
import { ConnectWallet } from './ConnectWallet';

const Navigation = () => {
  return (
    <nav className="bg-black border-b-4 border-primary-500 relative z-50">
      <div className="container mx-auto px-4">
        <div className="flex justify-between h-16 items-center">
          <Link href="/" className="arcade-text text-2xl text-primary-500 hover:text-primary-400 relative z-10 font-bold">
          SPEEDRUN
          </Link>
          <div className="flex space-x-4 items-center relative z-10">
            <Link
              href="/"
              className="arcade-btn border-yellow-500 text-yellow-500 hover:bg-yellow-500 relative"
            >
              MAKE TRANSFER
            </Link>
            <Link
              href="/leaderboard"
              className="arcade-btn border-yellow-500 text-yellow-500 hover:bg-yellow-500 relative"
            >
              LEADERBOARD
            </Link>
            <Link
              href="/about"
              className="arcade-btn border-yellow-500 text-yellow-500 hover:bg-yellow-500 relative"
            >
              LEARN MORE
            </Link>
            <div className="relative">
              <ConnectWallet />
            </div>
          </div>
        </div>
      </div>
    </nav>
  );
};

export default Navigation; 