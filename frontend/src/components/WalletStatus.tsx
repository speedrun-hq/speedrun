import { useAccount, useDisconnect } from 'wagmi';
import { ConnectWallet } from './ConnectWallet';

export function WalletStatus() {
  const { isConnected, address } = useAccount();
  const { disconnect } = useDisconnect();

  if (!isConnected) {
    return <ConnectWallet />;
  }

  return (
    <div className="flex items-center gap-4">
      <span className="text-yellow-500 font-mono text-sm">
        {address?.slice(0, 6)}...{address?.slice(-4)}
      </span>
    </div>
  );
}

export function DisconnectButton() {
  const { isConnected } = useAccount();
  const { disconnect } = useDisconnect();

  if (!isConnected) {
    return null;
  }

  return (
    <button
      onClick={() => disconnect()}
      className="px-4 py-2 bg-red-500 text-black font-mono font-bold border-2 border-red-600 rounded-lg hover:bg-red-600 hover:border-red-700 transition-all duration-200 shadow-[0_0_10px_rgba(239,68,68,0.5)] hover:shadow-[0_0_15px_rgba(239,68,68,0.7)]"
    >
      DISCONNECT
    </button>
  );
} 