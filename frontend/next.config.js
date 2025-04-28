/** @type {import('next').NextConfig} */
// Note: This config is also defined in next.config.ts for type safety
// If you modify this file, please update the TypeScript version as well
const nextConfig = {
  reactStrictMode: true,
  webpack: (config) => {
    config.resolve.alias = {
      ...config.resolve.alias,
      "@": require("path").resolve(__dirname, "src"),
    };
    return config;
  },
};

// This file is a thin wrapper around the TypeScript configuration
// It's needed because Next.js expects a .js file by default

module.exports = nextConfig;
