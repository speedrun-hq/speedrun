import { Chain } from "wagmi";
import { base, arbitrum, mainnet, bsc, polygon, avalanche } from "wagmi/chains";
export declare const BITCOIN_CHAIN_ID = 8332;
export declare const SOLANA_CHAIN_ID = 10001;
export declare const ZETACHAIN_CHAIN_ID = 7000;
export type ChainId = typeof mainnet.id | typeof bsc.id | typeof polygon.id | typeof base.id | typeof arbitrum.id | typeof avalanche.id | typeof BITCOIN_CHAIN_ID | typeof SOLANA_CHAIN_ID | typeof ZETACHAIN_CHAIN_ID;
export type ChainName = "ETHEREUM" | "BSC" | "POLYGON" | "BASE" | "ARBITRUM" | "AVALANCHE" | "BITCOIN" | "SOLANA" | "ZETACHAIN";
export type TokenSymbol = "USDC" | "USDT" | "BTC" | "ZETA";
export interface Token {
    address: `0x${string}`;
    decimals: number;
    symbol: TokenSymbol;
    name: string;
}
export interface TokenConfig {
    [key: string]: Token;
}
export declare const SUPPORTED_CHAINS: {
    id: number;
    name: ChainName;
}[];
export declare const COMING_SOON_SOURCE_CHAINS: {
    id: number;
    name: ChainName;
}[];
export declare const COMING_SOON_DESTINATION_CHAINS: {
    id: number;
    name: ChainName;
}[];
export declare const CHAIN_LOGO_MAP: Record<number, string>;
export declare const CHAIN_COLOR_MAP: Record<number, string>;
export declare const TOKENS: Record<number, TokenConfig>;
export declare const CHAIN_ID_TO_NAME: Record<number, ChainName>;
export declare const CHAIN_NAME_TO_ID: Record<ChainName, number>;
export declare function getChainId(chainName: ChainName): number;
export declare function getChainName(chainId: number): ChainName;
export declare function isValidChainId(chainId: number): boolean;
export declare function getChainRpcUrl(chainId: number): string;
export declare function getExplorerUrl(chainId: number, txHash: string): string;
export declare const getCustomChains: (alchemyId: string) => Chain[];
