import { TokenSymbol, ChainName } from "@speedrun/utils";
export interface TokenSelectorProps {
    value: TokenSymbol;
    onChange: (value: TokenSymbol) => void;
    label?: string;
    disabled?: boolean;
    sourceChain?: ChainName;
    destinationChain?: ChainName;
}
export declare function TokenSelector({ value, onChange, label, disabled, sourceChain, destinationChain, }: TokenSelectorProps): import("react/jsx-runtime").JSX.Element;
export default TokenSelector;
