// Set environment variable before importing the component
process.env.NEXT_PUBLIC_WALLETCONNECT_PROJECT_ID = 'test-project-id';

import React from 'react';
import { render } from '@testing-library/react';

// Mock the dependencies
jest.mock('@rainbow-me/rainbowkit', () => ({
  RainbowKitProvider: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="mock-rainbow-kit">{children}</div>
  ),
}));

jest.mock('wagmi', () => ({
  WagmiConfig: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="mock-wagmi-config">{children}</div>
  ),
}));

jest.mock('../../components/RpcTest', () => ({
  RpcTest: () => <div data-testid="mock-rpc-test">RPC Test</div>,
}));

jest.mock('../../utils/web3Config', () => ({
  config: {},
  chains: [],
}));

jest.mock('../../utils/rainbowKitTheme', () => ({
  arcadeTheme: {},
}));

describe('Web3Provider', () => {
  const originalEnv = process.env;

  beforeEach(() => {
    jest.resetModules();
    process.env = { ...originalEnv };
  });

  afterEach(() => {
    process.env = originalEnv;
  });

  it('renders children within the Web3Provider when environment variable is set', () => {
    process.env.NEXT_PUBLIC_WALLETCONNECT_PROJECT_ID = 'test-project-id';
    const { Web3Provider } = require('../../components/Web3Provider');

    const { getByTestId, getByText } = render(
      <Web3Provider>
        <div>Test Child</div>
      </Web3Provider>
    );
    
    // Check that the WagmiConfig is rendered
    expect(getByTestId('mock-wagmi-config')).toBeInTheDocument();
    
    // Check that the RainbowKitProvider is rendered
    expect(getByTestId('mock-rainbow-kit')).toBeInTheDocument();
    
    // Check that the RpcTest component is rendered
    expect(getByTestId('mock-rpc-test')).toBeInTheDocument();
    
    // Check that the child component is rendered
    expect(getByText('Test Child')).toBeInTheDocument();
  });

  it('throws an error when environment variable is not set', () => {
    delete process.env.NEXT_PUBLIC_WALLETCONNECT_PROJECT_ID;
    
    expect(() => {
      require('../../components/Web3Provider');
    }).toThrow('NEXT_PUBLIC_WALLETCONNECT_PROJECT_ID is not defined');
  });
}); 