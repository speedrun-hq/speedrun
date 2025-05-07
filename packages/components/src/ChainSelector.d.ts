export interface ChainSelectorProps {
    value: number;
    onChange: (value: number) => void;
    label?: string;
    disabled?: boolean;
    selectorType?: "from" | "to";
}
export declare function ChainSelector({ value, onChange, label, disabled, selectorType, }: ChainSelectorProps): import("react/jsx-runtime").JSX.Element;
export default ChainSelector;
