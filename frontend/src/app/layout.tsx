import type { Metadata } from 'next';
import './globals.css';
import Navigation from '@/components/Navigation';
import { Web3Provider } from '@/components/Web3Provider';

export const metadata: Metadata = {
  title: 'Speedrun',
  description: 'Fast Cross-Chain Transfers Powered by Speedrunners',
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <body className="bg-black text-white min-h-screen font-arcade">
        <Web3Provider>
          <Navigation />
          <main className="container mx-auto px-4 py-8">
            {children}
          </main>
        </Web3Provider>
      </body>
    </html>
  );
} 