"use client";
import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useState, useEffect, useRef } from "react";
export function TokenSelector({ value, onChange, label = "SELECT TOKEN", disabled = false, sourceChain = "BASE", destinationChain = "ARBITRUM", }) {
    const [isOpen, setIsOpen] = useState(false);
    const isBaseChainInvolved = sourceChain === "BASE" || destinationChain === "BASE";
    const tokens = isBaseChainInvolved
        ? ["USDC"]
        : ["USDC", "USDT"];
    const comingSoonTokens = isBaseChainInvolved
        ? ["BTC", "USDT", "ZETA"]
        : ["BTC", "ZETA"];
    const selectorRef = useRef(null);
    useEffect(() => {
        if (isBaseChainInvolved && value === "USDT") {
            onChange("USDC");
        }
    }, [sourceChain, destinationChain, value, onChange, isBaseChainInvolved]);
    const handleClick = () => {
        if (disabled)
            return;
        setIsOpen(!isOpen);
    };
    useEffect(() => {
        function handleOutsideClick(event) {
            if (selectorRef.current &&
                !selectorRef.current.contains(event.target)) {
                setIsOpen(false);
            }
        }
        if (isOpen) {
            document.addEventListener("mousedown", handleOutsideClick);
        }
        return () => {
            document.removeEventListener("mousedown", handleOutsideClick);
        };
    }, [isOpen]);
    return (_jsxs("div", { className: "relative w-full", ref: selectorRef, children: [_jsxs("button", { type: "button", onClick: handleClick, disabled: disabled, className: "w-full px-4 py-2 bg-black border-2 border-yellow-500 rounded-lg text-yellow-500 arcade-text text-xs focus:outline-none focus:border-yellow-400 flex justify-between items-center cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed", children: [_jsx("span", { children: value || label }), _jsx("span", { className: "ml-2", children: isOpen ? "▲" : "▼" })] }), isOpen && (_jsx("div", { className: "absolute top-full left-0 right-0 mt-2 z-[100]", children: _jsxs("div", { className: "bg-black border-2 border-yellow-500 rounded-lg overflow-hidden shadow-lg shadow-yellow-500/50", children: [tokens.map((token) => (_jsx("button", { type: "button", onClick: () => {
                                onChange(token);
                                setIsOpen(false);
                            }, className: `w-full px-4 py-3 text-left arcade-text text-xs hover:bg-yellow-500 hover:text-black transition-colors cursor-pointer ${token === value
                                ? "text-[#00ff00] bg-black/50"
                                : "text-yellow-500"}`, children: token }, token))), comingSoonTokens.map((token) => (_jsxs("div", { className: "w-full px-4 py-3 text-left arcade-text text-xs text-gray-500 cursor-not-allowed flex items-center justify-between", children: [_jsx("span", { children: token }), _jsx("span", { className: "text-gray-500 opacity-70 text-[10px] whitespace-nowrap flex-shrink-0", children: "COMING SOON" })] }, token)))] }) }))] }));
}
export default TokenSelector;
//# sourceMappingURL=TokenSelector.js.map