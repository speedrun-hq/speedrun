import { base, arbitrum, mainnet, bsc, polygon, avalanche } from "wagmi/chains";
// Custom chain IDs for upcoming chains
export const BITCOIN_CHAIN_ID = 8332;
export const SOLANA_CHAIN_ID = 10001;
export const ZETACHAIN_CHAIN_ID = 7000;
// Supported chain data
export const SUPPORTED_CHAINS = [
    { id: mainnet.id, name: "ETHEREUM" },
    { id: bsc.id, name: "BSC" },
    { id: polygon.id, name: "POLYGON" },
    { id: base.id, name: "BASE" },
    { id: arbitrum.id, name: "ARBITRUM" },
    { id: avalanche.id, name: "AVALANCHE" },
    { id: ZETACHAIN_CHAIN_ID, name: "ZETACHAIN" },
];
// Coming soon chains
export const COMING_SOON_SOURCE_CHAINS = [
    { id: BITCOIN_CHAIN_ID, name: "BITCOIN" },
    { id: SOLANA_CHAIN_ID, name: "SOLANA" },
];
export const COMING_SOON_DESTINATION_CHAINS = [
    { id: BITCOIN_CHAIN_ID, name: "BITCOIN" },
    { id: SOLANA_CHAIN_ID, name: "SOLANA" },
];
// Chain logo mapping for UI
export const CHAIN_LOGO_MAP = {
    [mainnet.id]: "/images/eth.png",
    [bsc.id]: "/images/bnb.png",
    [polygon.id]: "/images/pol.png",
    [base.id]: "/images/base.png",
    [arbitrum.id]: "/images/arb.png",
    [avalanche.id]: "/images/ava.png",
    [BITCOIN_CHAIN_ID]: "/images/btc.png",
    [SOLANA_CHAIN_ID]: "/images/sol.png",
    [ZETACHAIN_CHAIN_ID]: "/images/zeta.png",
};
// Chain color map for UI
export const CHAIN_COLOR_MAP = {
    [mainnet.id]: "text-gray-400",
    [bsc.id]: "text-yellow-400",
    [polygon.id]: "text-purple-500",
    [base.id]: "text-blue-400",
    [arbitrum.id]: "text-blue-600",
    [avalanche.id]: "text-red-600",
    [BITCOIN_CHAIN_ID]: "text-orange-500",
    [SOLANA_CHAIN_ID]: "text-purple-400",
    [ZETACHAIN_CHAIN_ID]: "text-green-500",
};
// Token configuration
export const TOKENS = {
    [base.id]: {
        USDC: {
            address: "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913",
            decimals: 6,
            symbol: "USDC",
            name: "USD Coin",
        },
        USDT: {
            address: "0x50c5725949A6F0c72E6C4a641F24049A917DB0Cb",
            decimals: 6,
            symbol: "USDT",
            name: "Tether USD",
        },
    },
    [arbitrum.id]: {
        USDC: {
            address: "0xaf88d065e77c8cC2239327C5EDb3A432268e5831",
            decimals: 6,
            symbol: "USDC",
            name: "USD Coin",
        },
        USDT: {
            address: "0xFd086bC7CD5C481DCC9C85ebE478A1C0b69FCbb9",
            decimals: 6,
            symbol: "USDT",
            name: "Tether USD",
        },
    },
    [mainnet.id]: {
        USDC: {
            address: "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48",
            decimals: 6,
            symbol: "USDC",
            name: "USD Coin",
        },
        USDT: {
            address: "0xdAC17F958D2ee523a2206206994597C13D831ec7",
            decimals: 6,
            symbol: "USDT",
            name: "Tether USD",
        },
    },
    [polygon.id]: {
        USDC: {
            address: "0x3c499c542cEF5E3811e1192ce70d8cC03d5c3359",
            decimals: 6,
            symbol: "USDC",
            name: "USD Coin",
        },
        USDT: {
            address: "0xc2132D05D31c914a87C6611C10748AEb04B58e8F",
            decimals: 6,
            symbol: "USDT",
            name: "Tether USD",
        },
    },
    [bsc.id]: {
        USDC: {
            address: "0x8AC76a51cc950d9822D68b83fE1Ad97B32Cd580d",
            decimals: 18,
            symbol: "USDC",
            name: "USD Coin",
        },
        USDT: {
            address: "0x55d398326f99059fF775485246999027B3197955",
            decimals: 18,
            symbol: "USDT",
            name: "Tether USD",
        },
    },
    [avalanche.id]: {
        USDC: {
            address: "0xB97EF9Ef8734C71904D8002F8b6Bc66Dd9c48a6E",
            decimals: 6,
            symbol: "USDC",
            name: "USD Coin",
        },
        USDT: {
            address: "0xde3a24028580884448a5397872046a019649b084",
            decimals: 6,
            symbol: "USDT",
            name: "Tether USD",
        },
    },
    [ZETACHAIN_CHAIN_ID]: {
        USDC: {
            address: "0x0cbe0dF132a6c6B4a2974Fa1b7Fb953CF0Cc798a",
            decimals: 6,
            symbol: "USDC",
            name: "USD Coin",
        },
        USDT: {
            address: "0x7c8dDa80bbBE1254a7aACf3219EBe1481c6E01d7",
            decimals: 6,
            symbol: "USDT",
            name: "Tether USD",
        },
    },
};
// Chain ID to name mapping
export const CHAIN_ID_TO_NAME = SUPPORTED_CHAINS.reduce((acc, chain) => (Object.assign(Object.assign({}, acc), { [chain.id]: chain.name })), {});
// Chain name to ID mapping
export const CHAIN_NAME_TO_ID = SUPPORTED_CHAINS.reduce((acc, chain) => (Object.assign(Object.assign({}, acc), { [chain.name]: chain.id })), {});
// Helper functions
export function getChainId(chainName) {
    const chain = SUPPORTED_CHAINS.find((c) => c.name === chainName);
    if (chain)
        return chain.id;
    // Check if it's one of the coming soon chains
    const comingSoonChain = [
        ...COMING_SOON_SOURCE_CHAINS,
        ...COMING_SOON_DESTINATION_CHAINS,
    ].find((c) => c.name === chainName);
    if (comingSoonChain)
        return comingSoonChain.id;
    return 0; // Return 0 for unknown chains
}
export function getChainName(chainId) {
    const chain = SUPPORTED_CHAINS.find((c) => c.id === chainId);
    if (chain)
        return chain.name;
    // Check if it's one of the coming soon chains
    const comingSoonChain = [
        ...COMING_SOON_SOURCE_CHAINS,
        ...COMING_SOON_DESTINATION_CHAINS,
    ].find((c) => c.id === chainId);
    if (comingSoonChain)
        return comingSoonChain.name;
    return "ETHEREUM"; // Default to Ethereum for unknown chains
}
export function isValidChainId(chainId) {
    return SUPPORTED_CHAINS.some((chain) => chain.id === chainId);
}
export function getChainRpcUrl(chainId) {
    switch (chainId) {
        case mainnet.id:
            return (process.env.NEXT_PUBLIC_ETHEREUM_RPC_URL || "https://eth.llamarpc.com");
        case bsc.id:
            return (process.env.NEXT_PUBLIC_BSC_RPC_URL ||
                "https://bsc-dataseed.bnbchain.org");
        case polygon.id:
            return (process.env.NEXT_PUBLIC_POLYGON_RPC_URL || "https://polygon-rpc.com");
        case base.id:
            return process.env.NEXT_PUBLIC_BASE_RPC_URL || "https://mainnet.base.org";
        case arbitrum.id:
            return (process.env.NEXT_PUBLIC_ARBITRUM_RPC_URL ||
                "https://arb1.arbitrum.io/rpc");
        case avalanche.id:
            return (process.env.NEXT_PUBLIC_AVALANCHE_RPC_URL ||
                "https://avalanche-c-chain-rpc.publicnode.com");
        case ZETACHAIN_CHAIN_ID:
            return (process.env.NEXT_PUBLIC_ZETACHAIN_RPC_URL ||
                "https://zetachain-mainnet.g.allthatnode.com/archive/evm");
        default:
            return "";
    }
}
export function getExplorerUrl(chainId, txHash) {
    switch (chainId) {
        case mainnet.id:
            return `https://etherscan.io/tx/${txHash}`;
        case bsc.id:
            return `https://bscscan.com/tx/${txHash}`;
        case polygon.id:
            return `https://polygonscan.com/tx/${txHash}`;
        case base.id:
            return `https://basescan.org/tx/${txHash}`;
        case arbitrum.id:
            return `https://arbiscan.io/tx/${txHash}`;
        case avalanche.id:
            return `https://snowtrace.io/tx/${txHash}`;
        case ZETACHAIN_CHAIN_ID:
            return `https://explorer.zetachain.com/tx/${txHash}`;
        default:
            return "";
    }
}
// Create custom chain configurations for wagmi
export const getCustomChains = (alchemyId) => {
    // Configure Ethereum chain with custom RPC URL
    const customMainnet = Object.assign(Object.assign({}, mainnet), { rpcUrls: Object.assign(Object.assign({}, mainnet.rpcUrls), { default: {
                http: [`https://eth-mainnet.g.alchemy.com/v2/${alchemyId}`],
            }, public: {
                http: ["https://eth.llamarpc.com"],
            } }) });
    // Configure BNB Chain with custom RPC URL
    const customBsc = Object.assign(Object.assign({}, bsc), { rpcUrls: Object.assign(Object.assign({}, bsc.rpcUrls), { default: {
                http: ["https://bsc-dataseed.bnbchain.org"],
            }, public: {
                http: ["https://bsc-dataseed.bnbchain.org"],
            } }) });
    // Configure Polygon chain with custom RPC URL
    const customPolygon = Object.assign(Object.assign({}, polygon), { rpcUrls: Object.assign(Object.assign({}, polygon.rpcUrls), { default: {
                http: [`https://polygon-mainnet.g.alchemy.com/v2/${alchemyId}`],
            }, public: {
                http: ["https://polygon-rpc.com"],
            } }) });
    // Configure Base chain with custom RPC URL
    const customBase = Object.assign(Object.assign({}, base), { rpcUrls: Object.assign(Object.assign({}, base.rpcUrls), { default: {
                http: [`https://base-mainnet.g.alchemy.com/v2/${alchemyId}`],
            }, public: {
                http: ["https://mainnet.base.org"],
            } }) });
    // Configure Arbitrum chain with custom RPC URL
    const customArbitrum = Object.assign(Object.assign({}, arbitrum), { rpcUrls: Object.assign(Object.assign({}, arbitrum.rpcUrls), { default: {
                http: [`https://arb-mainnet.g.alchemy.com/v2/${alchemyId}`],
            }, public: {
                http: ["https://arb1.arbitrum.io/rpc"],
            } }) });
    // Configure Avalanche chain with custom RPC URL
    const customAvalanche = Object.assign(Object.assign({}, avalanche), { rpcUrls: Object.assign(Object.assign({}, avalanche.rpcUrls), { default: {
                http: ["https://avalanche-c-chain-rpc.publicnode.com"],
            }, public: {
                http: ["https://avalanche-c-chain-rpc.publicnode.com"],
            } }) });
    // Configure ZetaChain
    const customZetaChain = {
        id: ZETACHAIN_CHAIN_ID,
        name: "ZetaChain",
        network: "zetachain",
        nativeCurrency: {
            name: "Zeta",
            symbol: "ZETA",
            decimals: 18,
        },
        rpcUrls: {
            default: {
                http: [
                    process.env.NEXT_PUBLIC_ZETACHAIN_RPC_URL ||
                        "https://zetachain-mainnet.g.allthatnode.com/archive/evm",
                ],
            },
            public: {
                http: [
                    process.env.NEXT_PUBLIC_ZETACHAIN_RPC_URL ||
                        "https://zetachain-mainnet.g.allthatnode.com/archive/evm",
                ],
            },
        },
        blockExplorers: {
            default: {
                name: "ZetaChain Explorer",
                url: "https://explorer.zetachain.com",
            },
        },
    };
    return [
        customMainnet,
        customBsc,
        customPolygon,
        customBase,
        customArbitrum,
        customAvalanche,
        customZetaChain,
    ];
};
//# sourceMappingURL=chain.js.map