'use client';

import { useEffect } from 'react';
import { usePublicClient, useNetwork } from 'wagmi';

export function RpcTest() {
  const { chain } = useNetwork();
  const publicClient = usePublicClient();

  useEffect(() => {
    async function testConnection() {
      if (!publicClient) return;
      
      try {
        const blockNumber = await publicClient.getBlockNumber();
        console.log('RPC Connection Test:', {
          chain: chain?.name,
          chainId: chain?.id,
          blockNumber: blockNumber.toString(),
          rpcUrl: chain?.rpcUrls?.default?.http[0],
        });
      } catch (error) {
        console.error('RPC Connection Error:', {
          chain: chain?.name,
          chainId: chain?.id,
          error,
        });
      }
    }

    testConnection();
  }, [publicClient, chain]);

  return null;
} 