// Mock implementation for @speedrun/components
import React from 'react';

// Mock components using React.createElement instead of JSX
export const ChainSelector = (props: any) => {
  return React.createElement('div', { 'data-testid': 'chain-selector' }, 
    React.createElement('select', {
      'data-testid': `chain-select-${props.label || ''}`,
      value: props.value || '',
      onChange: (e: any) => props.onChange && props.onChange(props.valueAsChainId ? parseInt(e.target.value) : e.target.value)
    }, [
      React.createElement('option', { value: '1', key: '1' }, 'ETHEREUM'),
      React.createElement('option', { value: '42161', key: '42161' }, 'ARBITRUM'),
      React.createElement('option', { value: '8453', key: '8453' }, 'BASE'),
      React.createElement('option', { value: '137', key: '137' }, 'POLYGON')
    ])
  );
};

export const TokenSelector = (props: any) => {
  return React.createElement('div', { 'data-testid': 'token-selector' },
    React.createElement('select', {
      'data-testid': `token-select-${props.label || ''}`,
      value: props.value || '',
      onChange: (e: any) => props.onChange && props.onChange(e.target.value)
    }, [
      React.createElement('option', { value: 'USDC', key: 'USDC' }, 'USDC'),
      React.createElement('option', { value: 'ETH', key: 'ETH' }, 'ETH'),
      React.createElement('option', { value: 'DAI', key: 'DAI' }, 'DAI')
    ])
  );
};

export const FormInput = (props: any) => {
  return React.createElement('div', { 'data-testid': 'form-input' },
    React.createElement('input', {
      'data-testid': `input-${props.placeholder || ''}`,
      value: props.value || '',
      onChange: (e: any) => props.onChange && props.onChange(e.target.value)
    })
  );
}; 