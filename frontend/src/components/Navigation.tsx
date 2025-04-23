"use client";

import Link from "next/link";
import { ConnectWallet } from "./ConnectWallet";
import { useState, useEffect } from "react";

const Navigation = () => {
  const [isMenuOpen, setIsMenuOpen] = useState(false);
  const [isMobile, setIsMobile] = useState(false);

  // Handle window resize to determine mobile view
  useEffect(() => {
    const handleResize = () => {
      setIsMobile(window.innerWidth < 1024);
    };

    // Set initial value
    handleResize();

    // Add event listener
    window.addEventListener("resize", handleResize);

    // Clean up
    return () => window.removeEventListener("resize", handleResize);
  }, []);

  // Toggle menu open/closed
  const toggleMenu = () => {
    setIsMenuOpen(!isMenuOpen);
  };

  // Close menu when clicking outside
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (
        isMenuOpen &&
        !(event.target as Element).closest(".mobile-menu") &&
        !(event.target as Element).closest(".hamburger-btn")
      ) {
        setIsMenuOpen(false);
      }
    };

    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, [isMenuOpen]);

  return (
    <nav className="bg-black border-b-4 border-primary-500 relative z-50">
      <div className="container mx-auto px-2">
        <div className="flex justify-between h-16 items-center">
          <Link
            href="/"
            className="arcade-text text-2xl text-primary-500 hover:text-primary-400 relative z-10 font-bold pl-2 flex items-center"
          >
            <img 
              src="/images/speed.png" 
              alt="Speedrun Logo" 
              className="h-10 mr-2" 
            />
            SPEEDRUN
          </Link>

          {/* Desktop Menu */}
          <div className="hidden lg:flex items-center space-x-2 relative z-10">
            <Link
              href="/"
              className="arcade-btn-sm border-green-400 text-green-400 hover:bg-green-400 transition-none min-w-[120px] text-center justify-center"
            >
              MAKE TRANSFER
            </Link>
            <Link
              href="/my-intents"
              className="arcade-btn-sm border-green-400 text-green-400 hover:bg-green-400 transition-none min-w-[120px] text-center justify-center"
            >
              MY TRANSFERS
            </Link>
            <Link
              href="/leaderboard"
              className="arcade-btn-sm border-green-400 text-green-400 hover:bg-green-400 transition-none min-w-[120px] text-center justify-center"
            >
              LEADERBOARD
            </Link>
            <Link
              href="/about"
              className="arcade-btn-sm border-green-400 text-green-400 hover:bg-green-400 transition-none min-w-[120px] text-center justify-center"
            >
              LEARN MORE
            </Link>
            <div className="flex justify-center">
              <ConnectWallet />
            </div>
          </div>

          {/* Mobile Menu Button */}
          <div className="lg:hidden flex items-center z-20">
            <div className="relative z-10">
              <ConnectWallet />
            </div>
            <button
              className="hamburger-btn ml-4 p-2 rounded border-2 border-green-400 text-green-400 hover:bg-green-400/20 transition-none z-50 relative cursor-pointer"
              onClick={toggleMenu}
              aria-label="Toggle menu"
              type="button"
            >
              <svg
                xmlns="http://www.w3.org/2000/svg"
                className="h-5 w-5 pointer-events-none"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                {isMenuOpen ? (
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M6 18L18 6M6 6l12 12"
                  />
                ) : (
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M4 6h16M4 12h16M4 18h16"
                  />
                )}
              </svg>
            </button>
          </div>
        </div>
      </div>

      {/* Mobile Menu Panel */}
      <div
        className={`mobile-menu lg:hidden absolute top-full left-0 right-0 bg-black border-b-4 border-primary-500 shadow-lg transform transition-transform duration-300 z-40 ${
          isMenuOpen
            ? "translate-y-0 opacity-100"
            : "-translate-y-full opacity-0 pointer-events-none"
        }`}
      >
        <div className="container mx-auto px-4 py-4 space-y-3">
          <Link
            href="/"
            className="arcade-btn-sm w-full block py-2 text-center border-green-400 text-green-400 hover:bg-green-400/20 transition-none"
            onClick={() => setIsMenuOpen(false)}
          >
            MAKE TRANSFER
          </Link>
          <Link
            href="/my-intents"
            className="arcade-btn-sm w-full block py-2 text-center border-green-400 text-green-400 hover:bg-green-400/20 transition-none"
            onClick={() => setIsMenuOpen(false)}
          >
            MY TRANSFERS
          </Link>
          <Link
            href="/leaderboard"
            className="arcade-btn-sm w-full block py-2 text-center border-green-400 text-green-400 hover:bg-green-400/20 transition-none"
            onClick={() => setIsMenuOpen(false)}
          >
            LEADERBOARD
          </Link>
          <Link
            href="/about"
            className="arcade-btn-sm w-full block py-2 text-center border-green-400 text-green-400 hover:bg-green-400/20 transition-none"
            onClick={() => setIsMenuOpen(false)}
          >
            LEARN MORE
          </Link>
        </div>
      </div>
    </nav>
  );
};

export default Navigation;
