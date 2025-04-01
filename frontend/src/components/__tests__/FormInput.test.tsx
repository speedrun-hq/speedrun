import { render, screen, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { FormInput } from '../FormInput';

describe('FormInput', () => {
  const defaultProps = {
    label: 'Test Label',
    value: '',
    onChange: jest.fn(),
  };

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('renders with label', () => {
    render(<FormInput {...defaultProps} />);
    expect(screen.getByText('Test Label')).toBeInTheDocument();
  });

  it('renders without label when not provided', () => {
    render(<FormInput value="" onChange={jest.fn()} />);
    expect(screen.queryByText('Test Label')).not.toBeInTheDocument();
  });

  it('displays the provided value', () => {
    render(<FormInput {...defaultProps} value="test value" />);
    const input = screen.getByRole('textbox');
    expect(input).toHaveValue('test value');
  });

  it('calls onChange when input value changes', async () => {
    render(<FormInput {...defaultProps} />);
    const input = screen.getByRole('textbox');
    
    await userEvent.type(input, 'new value');
    
    expect(defaultProps.onChange).toHaveBeenCalled();
  });

  it('applies disabled state correctly', () => {
    render(<FormInput {...defaultProps} disabled />);
    const input = screen.getByRole('textbox');
    expect(input).toBeDisabled();
  });

  it('applies number type correctly', () => {
    render(<FormInput {...defaultProps} type="number" />);
    const input = screen.getByRole('spinbutton');
    expect(input).toHaveAttribute('type', 'number');
  });

  it('applies max attribute for number inputs', () => {
    render(<FormInput {...defaultProps} type="number" max="100" />);
    const input = screen.getByRole('spinbutton');
    expect(input).toHaveAttribute('max', '100');
  });

  it('applies step attribute for number inputs', () => {
    render(<FormInput {...defaultProps} type="number" step="0.01" />);
    const input = screen.getByRole('spinbutton');
    expect(input).toHaveAttribute('step', '0.01');
  });

  it('applies placeholder text', () => {
    render(<FormInput {...defaultProps} placeholder="Enter value" />);
    const input = screen.getByRole('textbox');
    expect(input).toHaveAttribute('placeholder', 'Enter value');
  });

  it('applies custom className', () => {
    const customClass = 'my-custom-class';
    render(<FormInput className={customClass} />);
    const input = screen.getByRole('textbox');
    expect(input.className).toContain(customClass);
  });

  it('applies retro-futuristic styling', () => {
    render(<FormInput {...defaultProps} />);
    const input = screen.getByRole('textbox');
    const label = screen.getByText('Test Label');

    // Check for base styling classes
    expect(input.className).toMatch(/bg-black/);
    expect(input.className).toMatch(/border-\[hsl\(var\(--yellow\)\)\]/);
    expect(input.className).toMatch(/text-\[#00ff00\]/);
    expect(label.className).toMatch(/text-\[hsl\(var\(--yellow\)\)\]/);
    expect(label.className).toMatch(/font-mono/);
  });

  it('handles focus and blur states', async () => {
    render(<FormInput {...defaultProps} />);
    const input = screen.getByRole('textbox');

    // Focus state
    await userEvent.click(input);
    expect(input.className).toMatch(/focus:border-\[hsl\(var\(--yellow\)\/0\.8\)\]/);

    // Blur state
    await userEvent.tab();
    expect(input.className).toMatch(/border-\[hsl\(var\(--yellow\)\)\]/);
  });
}); 