declare module './ChainSelector' {
  export interface ChainSelectorProps {
    value: number;
    onChange: (value: number) => void;
  }
  export function ChainSelector(props: ChainSelectorProps): JSX.Element;
}

declare module './AmountInput' {
  export interface AmountInputProps {
    value: string;
    onChange: (value: string) => void;
    max?: string;
  }
  export function AmountInput(props: AmountInputProps): JSX.Element;
}

declare module './TokenSelector' {
  export interface TokenSelectorProps {
    value: 'USDC' | 'USDT';
    onChange: (value: 'USDC' | 'USDT') => void;
  }
  export function TokenSelector(props: TokenSelectorProps): JSX.Element;
} 