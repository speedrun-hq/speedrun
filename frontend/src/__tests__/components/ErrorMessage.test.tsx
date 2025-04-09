import React from 'react';
import { render, screen } from '@testing-library/react';
import ErrorMessage from '../../components/ErrorMessage';
import { ApiError } from '../../utils/errors';

// Mock the getErrorMessage function
jest.mock('../../utils/errors', () => ({
  ...jest.requireActual('../../utils/errors'),
  getErrorMessage: jest.fn((error) => {
    if (error instanceof ApiError) {
      return `API Error: ${error.message}`;
    }
    if (error instanceof Error) {
      return error.message;
    }
    return 'An unexpected error occurred';
  }),
}));

describe('ErrorMessage', () => {
  it('renders with a standard error message', () => {
    const error = new Error('Something went wrong');
    render(<ErrorMessage error={error} />);
    
    expect(screen.getByText('Something went wrong')).toBeInTheDocument();
  });

  it('renders with an API error message', () => {
    const error = new ApiError('API Error', 400);
    render(<ErrorMessage error={error} />);
    
    expect(screen.getByText('API Error: API Error')).toBeInTheDocument();
  });

  it('renders with a default error message for unknown errors', () => {
    const error = 'string error';
    render(<ErrorMessage error={error} />);
    
    expect(screen.getByText('An unexpected error occurred')).toBeInTheDocument();
  });

  it('applies custom className when provided', () => {
    const error = new Error('Test error');
    const { container } = render(<ErrorMessage error={error} className="custom-class" />);
    
    const errorContainer = container.firstChild;
    expect(errorContainer).toHaveClass('custom-class');
  });
}); 