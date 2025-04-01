import { ConnectButton } from '@rainbow-me/rainbowkit';

export function ConnectWallet() {
  return (
    <div className="flex justify-center items-center">
      <div className="arcade-btn border-yellow-500 text-yellow-500 hover:bg-yellow-500 hover:text-black transition-all duration-200">
        <ConnectButton
          chainStatus="icon"
          showBalance={false}
          label="CONNECT WALLET"
          accountStatus="address"
        />
      </div>
    </div>
  );
} 