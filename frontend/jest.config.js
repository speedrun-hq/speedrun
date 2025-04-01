const nextJest = require('next/jest');

const createJestConfig = nextJest({
  dir: './',
});

const customJestConfig = {
  setupFilesAfterEnv: ['<rootDir>/src/test/setup.ts'],
  testEnvironment: 'jest-environment-jsdom',
  moduleNameMapper: {
    '^@/(.*)$': '<rootDir>/src/$1',
  },
  transformIgnorePatterns: [
    'node_modules/(?!(wagmi|viem|@wagmi|isows|@rainbow-me/rainbowkit)/)',
  ],
  testMatch: ['**/__tests__/**/*.test.(ts|tsx)'],
};

module.exports = createJestConfig(customJestConfig); 