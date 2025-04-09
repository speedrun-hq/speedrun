import React from "react";
import { render, screen, fireEvent } from "@testing-library/react";
import { WalletStatus, DisconnectButton } from "../../components/WalletStatus";
import { ConnectWallet } from "../../components/ConnectWallet";

// Mock the wagmi hooks
jest.mock("wagmi", () => ({
  useAccount: jest.fn(),
  useDisconnect: jest.fn(() => ({ disconnect: jest.fn() })),
}));

// Mock the ConnectWallet component
jest.mock("../../components/ConnectWallet", () => ({
  ConnectWallet: jest.fn(() => (
    <div data-testid="mock-connect-wallet">Connect Wallet</div>
  )),
}));

describe("WalletStatus", () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it("renders ConnectWallet when not connected", () => {
    // Mock useAccount to return not connected
    const { useAccount } = require("wagmi");
    useAccount.mockReturnValue({ isConnected: false, address: null });

    render(<WalletStatus />);

    expect(screen.getByTestId("mock-connect-wallet")).toBeInTheDocument();
  });

  it("renders truncated address when connected", () => {
    // Mock useAccount to return connected with an address
    const { useAccount } = require("wagmi");
    useAccount.mockReturnValue({
      isConnected: true,
      address: "0x1234567890abcdef1234567890abcdef12345678",
    });

    render(<WalletStatus />);

    expect(screen.getByText("0x1234...5678")).toBeInTheDocument();
  });
});

describe("DisconnectButton", () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it("renders nothing when not connected", () => {
    // Mock useAccount to return not connected
    const { useAccount } = require("wagmi");
    useAccount.mockReturnValue({ isConnected: false });

    const { container } = render(<DisconnectButton />);

    expect(container).toBeEmptyDOMElement();
  });

  it("renders disconnect button when connected", () => {
    // Mock useAccount to return connected
    const { useAccount } = require("wagmi");
    useAccount.mockReturnValue({ isConnected: true });

    // Mock useDisconnect to return disconnect function
    const mockDisconnect = jest.fn();
    const { useDisconnect } = require("wagmi");
    useDisconnect.mockReturnValue({ disconnect: mockDisconnect });

    render(<DisconnectButton />);

    const disconnectButton = screen.getByText("DISCONNECT");
    expect(disconnectButton).toBeInTheDocument();

    fireEvent.click(disconnectButton);
    expect(mockDisconnect).toHaveBeenCalled();
  });
});
