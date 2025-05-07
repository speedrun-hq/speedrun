# Speedrun Widget Development Setup

This document provides setup instructions for developers working on the Speedrun Widget.

## Setting Up Development Environment

### Prerequisites
- Node.js (v16+)
- npm, yarn, or pnpm

### Recommended Approach: Standalone Development

To avoid workspace conflicts, we recommend working on the widget in a separate directory outside the main Speedrun repository:

```bash
# Clone the repository
git clone https://github.com/speedrun-hq/speedrun.git

# Create a separate directory for widget development
mkdir speedrun-widget-dev
cd speedrun-widget-dev

# Copy widget files
cp -R ../speedrun/widget/* .

# Initialize a new git repository (optional)
git init

# Install dependencies
npm install --legacy-peer-deps
# or
yarn install
# or
pnpm install --no-strict-peer-dependencies
```

### Alternative: Working Within the Monorepo

If you need to work within the monorepo, use these steps to avoid conflicts:

```bash
# Navigate to the widget directory
cd speedrun/widget

# Install dependencies with flags to avoid conflicts
npm install --legacy-peer-deps --no-workspaces
# or
yarn install --ignore-workspace-root-check
# or
pnpm install --ignore-workspace
```

## Development Workflow

### Running in Development Mode

```bash
# Start the development server
npm run dev
# or
yarn dev
# or
pnpm dev
```

### Building the Package

```bash
# Build the widget
npm run build
# or
yarn build
# or
pnpm build
```

### Running Tests

```bash
# Run tests
npm test
# or
yarn test
# or
pnpm test

# Run tests in watch mode
npm run test:watch
# or
yarn test:watch
# or
pnpm test:watch

# Run tests with coverage
npm run test:coverage
# or
yarn test:coverage
# or
pnpm test:coverage
```

## Troubleshooting Common Issues

### Dependency Conflicts

If you encounter dependency conflicts, try one of these approaches:

1. Use the `--legacy-peer-deps` flag with npm:
   ```bash
   npm install --legacy-peer-deps
   ```

2. Update the peerDependencies in package.json to be more flexible:
   ```json
   "peerDependencies": {
     "react": "^18.0.0 || ^17.0.0 || ^16.9.0",
     "react-dom": "^18.0.0 || ^17.0.0 || ^16.9.0",
     "wagmi": "^1.0.0"
   }
   ```

3. Add resolutions to package.json (for yarn):
   ```json
   "resolutions": {
     "@testing-library/react-hooks": {
       "react": "^18.0.0 || ^17.0.0 || ^16.9.0"
     }
   }
   ```

### Testing Issues

If Jest tests are failing because of environment issues:

1. Ensure your Node.js version is compatible (v16+)
2. Try clearing the Jest cache:
   ```bash
   npx jest --clearCache
   ```
3. Check if the problem is related to the test environment:
   ```bash
   npm test -- --env=jsdom
   ```

## Publishing to NPM

For maintainers with publishing rights:

```bash
# Ensure you're logged in to npm
npm login

# Update the version in package.json (patch, minor, or major)
npm version patch
# or
npm version minor
# or 
npm version major

# Publish the package
npm publish
```

The GitHub Actions workflow will also automatically publish to npm when changes are pushed to the main branch and the version has been updated.

## Integration Testing

To test the widget in a real application:

1. Build the widget:
   ```bash
   npm run build
   ```

2. Link it locally:
   ```bash
   npm link
   ```

3. In your test application:
   ```bash
   npm link @speedrun/widget
   ```

4. Import and use the widget in your test application 