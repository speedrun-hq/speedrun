/** @type {import('jest').Config} */
module.exports = {
  preset: 'ts-jest',
  testEnvironment: 'jsdom',
  setupFilesAfterEnv: ['<rootDir>/src/setupTests.ts'],
  moduleNameMapper: {
    '\\.(css|less|scss|sass)$': 'identity-obj-proxy',
    // Mock the @speedrun packages
    '^@speedrun/utils$': '<rootDir>/src/__mocks__/speedrunUtils.ts',
    '^@speedrun/components$': '<rootDir>/src/__mocks__/speedrunComponents.ts'
  },
  collectCoverageFrom: [
    'src/**/*.{ts,tsx}',
    '!src/**/*.d.ts',
    '!src/setupTests.ts',
    '!src/__mocks__/**'
  ],
  testPathIgnorePatterns: ['/node_modules/', '/dist/']
}; 