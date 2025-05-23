export const Intent = {
  address: {
    [1]: process.env.NEXT_PUBLIC_INTENT_CONTRACT_ETHEREUM as `0x${string}`, // Ethereum
    [56]: process.env.NEXT_PUBLIC_INTENT_CONTRACT_BSC as `0x${string}`, // BNB Chain
    [137]: process.env.NEXT_PUBLIC_INTENT_CONTRACT_POLYGON as `0x${string}`, // Polygon
    [8453]: process.env.NEXT_PUBLIC_INTENT_CONTRACT_BASE as `0x${string}`, // Base
    [42161]: process.env.NEXT_PUBLIC_INTENT_CONTRACT_ARBITRUM as `0x${string}`, // Arbitrum
    [43114]: process.env.NEXT_PUBLIC_INTENT_CONTRACT_AVALANCHE as `0x${string}`, // Avalanche
    [7000]: process.env.NEXT_PUBLIC_INTENT_CONTRACT_ZETACHAIN as `0x${string}`, // ZetaChain
  },
  abi: [
    {
      inputs: [
        { name: "asset", type: "address" },
        { name: "amount", type: "uint256" },
        { name: "targetChain", type: "uint256" },
        { name: "receiver", type: "bytes" },
        { name: "tip", type: "uint256" },
        { name: "salt", type: "uint256" },
      ],
      name: "initiate",
      outputs: [{ name: "", type: "bytes32" }],
      stateMutability: "nonpayable",
      type: "function",
    },
    {
      anonymous: false,
      inputs: [
        { indexed: true, name: "intentId", type: "bytes32" },
        { indexed: true, name: "asset", type: "address" },
        { indexed: false, name: "amount", type: "uint256" },
        { indexed: false, name: "targetChain", type: "uint256" },
        { indexed: false, name: "receiver", type: "bytes" },
        { indexed: false, name: "tip", type: "uint256" },
        { indexed: false, name: "salt", type: "uint256" },
      ],
      name: "IntentInitiated",
      type: "event",
    },
  ],
} as const;
