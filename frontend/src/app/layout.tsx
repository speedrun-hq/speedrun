import type { Metadata } from "next";
import { Press_Start_2P } from "next/font/google";
import "./globals.css";
import Navigation from "@/components/Navigation";
import { Web3Provider } from "@/components/Web3Provider";
import WarningBanner from "@/components/WarningBanner";

const arcade = Press_Start_2P({
  weight: "400",
  subsets: ["latin"],
  display: "swap",
});

export const metadata: Metadata = {
  title: "Speedrun",
  description: "Fast cross-chain token transfers powered by ZetaChain",
  icons: {
    icon: [
      { url: "/favicon.ico", sizes: "any" },
      { url: "/favicon.svg", type: "image/svg+xml" },
    ],
    apple: [{ url: "/favicon.svg" }],
  },
  manifest: "/site.webmanifest",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <body
        className={`${arcade.className} bg-black text-white min-h-screen flex flex-col`}
      >
        <Web3Provider>
          <Navigation />
          <WarningBanner />
          <main className="container mx-auto px-4 py-8 flex-grow">
            {children}
          </main>

          {/* Footer */}
          <footer className="w-full py-6 text-center arcade-text text-xs text-gray-500 border-t border-gray-900 relative z-20">
            <div className="container mx-auto">
              <div className="flex flex-col md:flex-row justify-center items-center gap-2 md:gap-4">
                <a
                  href="https://x.com/speedrun_hq"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="flex items-center text-green-400 hover:text-green-300 transition-none z-20 relative"
                >
                  <svg
                    className="h-4 w-4 mr-1"
                    fill="currentColor"
                    viewBox="0 0 24 24"
                  >
                    <path d="M18.901 1.153h3.68l-8.04 9.19L24 22.846h-7.406l-5.8-7.584-6.638 7.584H.474l8.6-9.83L0 1.154h7.594l5.243 6.932ZM17.61 20.644h2.039L6.486 3.24H4.298Z" />
                  </svg>
                  X
                </a>
                <a
                  href="https://github.com/speedrun-hq"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="flex items-center text-green-400 hover:text-green-300 transition-none z-20 relative"
                >
                  <svg
                    className="h-4 w-4 mr-1"
                    fill="currentColor"
                    viewBox="0 0 24 24"
                  >
                    <path
                      fillRule="evenodd"
                      d="M12 2C6.477 2 2 6.484 2 12.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.531 1.032 1.531 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0112 6.844c.85.004 1.705.115 2.504.337 1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.202 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.309.678.92.678 1.855 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.019 10.019 0 0022 12.017C22 6.484 17.522 2 12 2z"
                      clipRule="evenodd"
                    />
                  </svg>
                  GitHub
                </a>
                <div className="text-yellow-500 relative z-20 select-text">
                  ©️ 2025 SPEEDRUN
                </div>
                <a
                  href="https://www.zetachain.com/"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-green-400 hover:text-green-300 transition-none z-20 relative select-text"
                >
                  Powered by ZetaChain
                </a>
              </div>
            </div>
          </footer>
        </Web3Provider>
      </body>
    </html>
  );
}
