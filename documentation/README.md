# Speedrun Documentation

This is the documentation site for Speedrun, built with [Docusaurus](https://docusaurus.io/).

## Development

### Prerequisites

- Node.js version 18 or above (use nvm to manage Node versions)
- Yarn package manager

### Setup

1. Make sure you're using Node.js 18 or higher:
   ```
   nvm use
   ```

2. Install dependencies:
   ```
   yarn
   ```

3. Start the development server:
   ```
   yarn start
   ```

This will start a local development server and open up a browser window. Most changes are reflected live without having to restart the server.

## Build

```
yarn build
```

This command generates static content into the `build` directory that can be served by any static content hosting service.

## Deployment

The documentation site is configured to be deployed to `docs.speedrun.exchange`.
