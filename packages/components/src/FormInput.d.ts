export interface FormInputProps {
    label?: string;
    value: string;
    onChange: (value: string) => void;
    placeholder?: string;
    disabled?: boolean;
    error?: string;
    type?: string;
    max?: string;
    min?: string;
    step?: string;
    className?: string;
    labelClassName?: string;
}
export declare function FormInput({ label, value, onChange, placeholder, disabled, error, type, max, min, step, className, labelClassName, }: FormInputProps): import("react/jsx-runtime").JSX.Element;
export default FormInput;
