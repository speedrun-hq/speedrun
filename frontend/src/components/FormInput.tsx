import React from "react";

interface FormInputProps {
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

export function FormInput({
  label,
  value,
  onChange,
  placeholder,
  disabled = false,
  error,
  type = "text",
  max,
  min,
  step,
  className = "",
  labelClassName = "block text-[hsl(var(--yellow))] mb-2 font-mono",
}: FormInputProps) {
  return (
    <div className="space-y-2 relative z-10">
      {label && <label className={labelClassName}>{label}</label>}
      <input
        type={type}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        disabled={disabled}
        max={max}
        min={min}
        step={step}
        className={`w-full px-4 py-2 bg-black border-2 border-[hsl(var(--yellow))] rounded-lg text-[#00ff00] arcade-text text-xs normal-case focus:outline-none focus:border-[hsl(var(--yellow)/0.8)] disabled:opacity-50 disabled:cursor-not-allowed placeholder:text-[hsl(var(--yellow)/0.5)] pointer-events-auto ${className}`}
        style={{ textTransform: "none" }}
      />
      {error && <p className="text-red-500 text-xs arcade-text">{error}</p>}
    </div>
  );
}
