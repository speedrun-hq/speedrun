import {themes as prismThemes} from 'prism-react-renderer';
import type {Config} from '@docusaurus/types';
import type * as Preset from '@docusaurus/preset-classic';

// This runs in Node.js - Don't use client-side code here (browser APIs, JSX...)

const config: Config = {
  title: 'Speedrun Docs',
  tagline: 'Documentation for Speedrun',
  favicon: 'img/favicon.ico',

  // Set the production url of your site here
  url: 'https://docs.speedrun.exchange',
  // Set the /<baseUrl>/ pathname under which your site is served
  // For GitHub pages deployment, it is often '/<projectName>/'
  baseUrl: '/',

  // GitHub pages deployment config.
  // If you aren't using GitHub pages, you don't need these.
  organizationName: 'speedrun', // Usually your GitHub org/user name.
  projectName: 'speedrun', // Usually your repo name.

  onBrokenLinks: 'throw',
  onBrokenMarkdownLinks: 'warn',

  // Even if you don't use internationalization, you can use this field to set
  // useful metadata like html lang. For example, if your site is Chinese, you
  // may want to replace "en" with "zh-Hans".
  i18n: {
    defaultLocale: 'en',
    locales: ['en'],
  },

  presets: [
    [
      'classic',
      {
        docs: {
          sidebarPath: './sidebars.ts',
          // Remove the edit this page links
          editUrl: undefined,
          routeBasePath: '/', // Set docs as the homepage
        },
        theme: {
          customCss: './src/css/custom.css',
        },
      } satisfies Preset.Options,
    ],
  ],

  themeConfig: {
    // Replace with your project's social card
    image: 'img/speedrun-social-card.jpg',
    colorMode: {
      defaultMode: 'dark', // Default to dark mode
      disableSwitch: true, // Disable theme switcher
      respectPrefersColorScheme: false, // Don't respect system preferences
    },
    navbar: {
      title: 'Speedrun',
      logo: {
        alt: 'Speedrun Logo',
        src: 'img/logo.png', // Placeholder for the logo
      },
      items: [
        {
          href: 'https://speedrun.exchange',
          label: 'Main Site',
          position: 'right',
        },
      ],
    },
    footer: {
      style: 'dark',
      links: [], // No links in the footer sections
      copyright: `<span class="footer-left">© ${new Date().getFullYear()} <a href="https://speedrun.exchange">speedrun</a></span><span class="footer-separator">|</span><span class="footer-right">powered by <a href="https://www.zetachain.com/" target="_blank" rel="noopener noreferrer">zetachain</a></span>`,
    },
    prism: {
      theme: prismThemes.dracula, // Arcade-friendly dark theme
      darkTheme: prismThemes.dracula,
    },
  } satisfies Preset.ThemeConfig,
};

export default config;
