"use client";
import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useState, useEffect, useRef } from "react";
import { SUPPORTED_CHAINS, COMING_SOON_SOURCE_CHAINS, COMING_SOON_DESTINATION_CHAINS, CHAIN_LOGO_MAP, CHAIN_COLOR_MAP, } from "@speedrun/utils";
// Default border color for all selectors
const BORDER_COLOR = "border-yellow-500";
const SHADOW_COLOR = "shadow-yellow-500/50";
const TEXT_COLOR = "text-yellow-500";
export function ChainSelector({ value, onChange, label = "SELECT CHAIN", disabled, selectorType = "from", // Default to 'from' if not specified
 }) {
    const [isOpen, setIsOpen] = useState(false);
    // Get the appropriate coming soon chains based on the selector type
    const comingSoonChains = selectorType === "from"
        ? COMING_SOON_SOURCE_CHAINS
        : COMING_SOON_DESTINATION_CHAINS;
    const selectorRef = useRef(null);
    const handleClick = () => {
        if (disabled)
            return;
        setIsOpen(!isOpen);
    };
    // Handle click outside to close dropdown
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
    const selectedChain = SUPPORTED_CHAINS.find((chain) => chain.id === value);
    return (_jsxs("div", { className: "relative w-full", ref: selectorRef, children: [_jsxs("button", { type: "button", onClick: handleClick, disabled: disabled, className: `w-full px-4 py-2 bg-black border-2 ${BORDER_COLOR} rounded-lg ${TEXT_COLOR} arcade-text text-xs focus:outline-none flex justify-between items-center cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed`, children: [_jsxs("span", { className: "flex items-center", children: [selectedChain &&
                                (CHAIN_LOGO_MAP[selectedChain.id] ? (_jsx("img", { src: CHAIN_LOGO_MAP[selectedChain.id], alt: selectedChain.name, className: "w-5 h-5 mr-2" })) : (_jsx("span", { className: `mr-2 inline-block text-xl leading-none ${CHAIN_COLOR_MAP[selectedChain.id]}`, children: "\u2022" }))), (selectedChain === null || selectedChain === void 0 ? void 0 : selectedChain.name) || label] }), _jsx("span", { className: "ml-2", children: isOpen ? "▲" : "▼" })] }), isOpen && (_jsx("div", { className: "absolute top-full left-0 right-0 mt-2 z-[1000]", children: _jsxs("div", { className: `bg-black border-2 ${BORDER_COLOR} rounded-lg overflow-hidden shadow-lg ${SHADOW_COLOR} max-h-96 overflow-y-auto`, children: [SUPPORTED_CHAINS.map((chain) => (_jsxs("button", { type: "button", onClick: () => {
                                onChange(chain.id);
                                setIsOpen(false);
                            }, className: `w-full px-4 py-3 text-left arcade-text text-xs hover:bg-yellow-400 hover:text-black transition-colors cursor-pointer flex items-center ${chain.id === value ? "bg-black/50" : ""}`, children: [CHAIN_LOGO_MAP[chain.id] ? (_jsx("img", { src: CHAIN_LOGO_MAP[chain.id], alt: chain.name, className: "w-5 h-5 mr-2" })) : (_jsx("span", { className: `mr-2 inline-block text-xl leading-none ${CHAIN_COLOR_MAP[chain.id]}`, children: "\u2022" })), _jsx("span", { className: `${TEXT_COLOR}`, children: chain.name })] }, chain.id))), comingSoonChains.map((chain) => (_jsxs("div", { className: "w-full px-4 py-3 text-left arcade-text text-xs text-gray-500 cursor-not-allowed flex items-center justify-between", children: [_jsxs("div", { className: "flex items-center mr-2", children: [CHAIN_LOGO_MAP[chain.id] ? (_jsx("img", { src: CHAIN_LOGO_MAP[chain.id], alt: chain.name, className: "w-5 h-5 mr-2 opacity-50" })) : (_jsx("span", { className: `mr-2 inline-block text-xl leading-none text-gray-500 flex-shrink-0`, children: "\u2022" })), _jsx("span", { children: chain.name })] }), _jsx("span", { className: "text-gray-500 opacity-70 text-[10px] whitespace-nowrap flex-shrink-0", children: "COMING SOON" })] }, chain.id)))] }) }))] }));
}
export default ChainSelector;
//# sourceMappingURL=ChainSelector.js.map