import '@testing-library/jest-dom';
import { TextEncoder, TextDecoder } from 'util';
import React from 'react';

// Polyfill for TextEncoder/TextDecoder
global.TextEncoder = TextEncoder;
global.TextDecoder = TextDecoder as any;

// Mock window.matchMedia
Object.defineProperty(window, 'matchMedia', {
  writable: true,
  value: jest.fn().mockImplementation(query => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: jest.fn(),
    removeListener: jest.fn(),
    addEventListener: jest.fn(),
    removeEventListener: jest.fn(),
    dispatchEvent: jest.fn(),
  })),
});

// Mock ResizeObserver
global.ResizeObserver = class ResizeObserver {
  observe() {}
  unobserve() {}
  disconnect() {}
};

// Mock wagmi hooks and chains
jest.mock('wagmi', () => ({
  useAccount: jest.fn(),
  useBalance: jest.fn(),
  useNetwork: jest.fn(),
  useContractRead: jest.fn().mockReturnValue({
    data: BigInt('1000000000'),
    isError: false,
    isLoading: false,
  }),
}));

// Mock wagmi/chains
jest.mock('wagmi/chains', () => ({
  base: {
    id: 8453,
    name: 'BASE',
  },
  arbitrum: {
    id: 42161,
    name: 'ARBITRUM',
  },
}));

// Mock RainbowKit components
jest.mock('@rainbow-me/rainbowkit', () => ({
  RainbowKitProvider: ({ children }: { children: React.ReactNode }) => React.createElement('div', null, children),
  ConnectButton: () => React.createElement('button', null, 'Connect Wallet'),
}));

// Mock next/image
jest.mock('next/image', () => ({
  __esModule: true,
  default: (props: any) => React.createElement('img', props),
}));

// Mock next/navigation
jest.mock('next/navigation', () => ({
  useRouter: () => ({
    push: jest.fn(),
    replace: jest.fn(),
    prefetch: jest.fn(),
  }),
  usePathname: () => '',
  useSearchParams: () => new URLSearchParams(),
}));

// Mock ethers
jest.mock('ethers', () => ({
  ethers: {
    providers: {
      JsonRpcProvider: jest.fn(),
    },
    Contract: jest.fn(),
  },
}));

// Mock viem
jest.mock('viem', () => ({
  createPublicClient: jest.fn(),
  http: jest.fn(),
  parseEther: jest.fn(),
  formatEther: jest.fn(),
  formatUnits: (value: bigint, decimals: number) => {
    return (Number(value) / Math.pow(10, decimals)).toString();
  },
})); 