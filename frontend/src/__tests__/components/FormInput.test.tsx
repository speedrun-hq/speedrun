import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { FormInput } from '../../components/FormInput';

describe('FormInput', () => {
  it('renders correctly with default props', () => {
    const mockOnChange = jest.fn();
    render(<FormInput value="" onChange={mockOnChange} />);
    
    const input = screen.getByRole('textbox');
    expect(input).toBeInTheDocument();
    expect(input).toHaveAttribute('type', 'text');
  });

  it('renders with a label when provided', () => {
    const mockOnChange = jest.fn();
    render(<FormInput label="Test Label" value="" onChange={mockOnChange} />);
    
    expect(screen.getByText('Test Label')).toBeInTheDocument();
  });

  it('calls onChange when input value changes', () => {
    const mockOnChange = jest.fn();
    render(<FormInput value="" onChange={mockOnChange} />);
    
    const input = screen.getByRole('textbox');
    fireEvent.change(input, { target: { value: 'test value' } });
    
    expect(mockOnChange).toHaveBeenCalledWith('test value');
  });

  it('displays error message when provided', () => {
    const mockOnChange = jest.fn();
    render(<FormInput value="" onChange={mockOnChange} error="This is an error" />);
    
    expect(screen.getByText('This is an error')).toBeInTheDocument();
  });

  it('renders with custom type', () => {
    const mockOnChange = jest.fn();
    render(<FormInput value="test" onChange={mockOnChange} type="password" />);
    
    const input = screen.getByDisplayValue('test');
    expect(input).toHaveAttribute('type', 'password');
  });

  it('renders with number attributes when type is number', () => {
    const mockOnChange = jest.fn();
    render(
      <FormInput 
        value="" 
        onChange={mockOnChange} 
        type="number" 
        min="0" 
        max="100" 
        step="1" 
      />
    );
    
    const input = screen.getByRole('spinbutton');
    expect(input).toHaveAttribute('type', 'number');
    expect(input).toHaveAttribute('min', '0');
    expect(input).toHaveAttribute('max', '100');
    expect(input).toHaveAttribute('step', '1');
  });

  it('is disabled when disabled prop is true', () => {
    const mockOnChange = jest.fn();
    render(<FormInput value="" onChange={mockOnChange} disabled={true} />);
    
    const input = screen.getByRole('textbox');
    expect(input).toBeDisabled();
  });

  it('applies custom className when provided', () => {
    const mockOnChange = jest.fn();
    render(<FormInput value="" onChange={mockOnChange} className="custom-class" />);
    
    const input = screen.getByRole('textbox');
    expect(input).toHaveClass('custom-class');
  });
}); 