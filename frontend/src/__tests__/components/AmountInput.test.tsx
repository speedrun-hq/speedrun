import React from "react";
import { render, screen, fireEvent } from "@testing-library/react";
import { AmountInput } from "../../components/AmountInput";

describe("AmountInput", () => {
  it("renders correctly", () => {
    const mockOnChange = jest.fn();
    render(<AmountInput value="" onChange={mockOnChange} />);

    const input = screen.getByPlaceholderText("0.00");
    expect(input).toBeInTheDocument();
    expect(input).toHaveAttribute("type", "number");
    expect(input).toHaveAttribute("min", "0");
    expect(input).toHaveAttribute("step", "0.01");
  });

  it("calls onChange when input value changes", () => {
    const mockOnChange = jest.fn();
    render(<AmountInput value="" onChange={mockOnChange} />);

    const input = screen.getByPlaceholderText("0.00");
    fireEvent.change(input, { target: { value: "10.5" } });

    expect(mockOnChange).toHaveBeenCalledWith("10.5");
  });

  it("displays error message for invalid input", () => {
    const mockOnChange = jest.fn();
    render(<AmountInput value="-5" onChange={mockOnChange} />);

    expect(screen.getByText("Please enter a valid amount")).toBeInTheDocument();
  });

  it("displays error message when amount exceeds max", () => {
    const mockOnChange = jest.fn();
    render(<AmountInput value="15" onChange={mockOnChange} max="10" />);

    expect(screen.getByText("Amount exceeds your balance")).toBeInTheDocument();
  });

  it("prevents input when disabled", () => {
    const mockOnChange = jest.fn();
    render(<AmountInput value="5" onChange={mockOnChange} disabled={true} />);

    const input = screen.getByPlaceholderText("0.00");
    expect(input).toBeDisabled();
  });

  it("does not call onChange when amount exceeds max", () => {
    const mockOnChange = jest.fn();
    render(<AmountInput value="5" onChange={mockOnChange} max="10" />);

    const input = screen.getByPlaceholderText("0.00");
    fireEvent.change(input, { target: { value: "15" } });

    expect(mockOnChange).not.toHaveBeenCalled();
  });
});
