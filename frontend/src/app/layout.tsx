import type { Metadata } from 'next';
import { Press_Start_2P } from 'next/font/google';
import './globals.css';
import Navigation from '@/components/Navigation';
import { Web3Provider } from '@/components/Web3Provider';

const arcade = Press_Start_2P({ 
  weight: '400',
  subsets: ['latin'],
  display: 'swap',
});

export const metadata: Metadata = {
  title: 'SpeedRun',
  description: 'Fast cross-chain token transfers powered by ZetaChain',
  icons: {
    icon: [
      { url: '/favicon.ico', sizes: 'any' },
      { url: '/favicon.svg', type: 'image/svg+xml' },
    ],
    apple: [
      { url: '/favicon.svg' },
    ],
  },
  manifest: '/site.webmanifest',
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <body className={`${arcade.className} bg-black text-white min-h-screen`}>
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