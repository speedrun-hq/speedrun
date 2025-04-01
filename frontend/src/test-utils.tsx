import React from 'react';
import { render as rtlRender } from '@testing-library/react';

// Create a wrapper component that provides the router context
function AllTheProviders({ children }: { children: React.ReactNode }) {
  return (
    <div>
      {children}
    </div>
  );
}

// Override the default render method
function render(ui: React.ReactElement, options = {}) {
  return rtlRender(ui, { wrapper: AllTheProviders, ...options });
}

// Re-export everything
export * from '@testing-library/react';

// Override render method
export { render }; 