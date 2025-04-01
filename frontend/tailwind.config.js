/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    './src/pages/**/*.{js,ts,jsx,tsx,mdx}',
    './src/components/**/*.{js,ts,jsx,tsx,mdx}',
    './src/app/**/*.{js,ts,jsx,tsx,mdx}',
  ],
  theme: {
    extend: {
      colors: {
        primary: {
          50: 'hsl(var(--primary) / 0.1)',
          100: 'hsl(var(--primary) / 0.2)',
          200: 'hsl(var(--primary) / 0.3)',
          300: 'hsl(var(--primary) / 0.4)',
          400: 'hsl(var(--primary) / 0.5)',
          500: 'hsl(var(--primary) / 0.6)',
          600: 'hsl(var(--primary) / 0.7)',
          700: 'hsl(var(--primary) / 0.8)',
          800: 'hsl(var(--primary) / 0.9)',
          900: 'hsl(var(--primary) / 1)',
        },
        secondary: {
          50: 'hsl(var(--secondary) / 0.1)',
          100: 'hsl(var(--secondary) / 0.2)',
          200: 'hsl(var(--secondary) / 0.3)',
          300: 'hsl(var(--secondary) / 0.4)',
          400: 'hsl(var(--secondary) / 0.5)',
          500: 'hsl(var(--secondary) / 0.6)',
          600: 'hsl(var(--secondary) / 0.7)',
          700: 'hsl(var(--secondary) / 0.8)',
          800: 'hsl(var(--secondary) / 0.9)',
          900: 'hsl(var(--secondary) / 1)',
        },
        accent: {
          50: 'hsl(var(--accent) / 0.1)',
          100: 'hsl(var(--accent) / 0.2)',
          200: 'hsl(var(--accent) / 0.3)',
          300: 'hsl(var(--accent) / 0.4)',
          400: 'hsl(var(--accent) / 0.5)',
          500: 'hsl(var(--accent) / 0.6)',
          600: 'hsl(var(--accent) / 0.7)',
          700: 'hsl(var(--accent) / 0.8)',
          800: 'hsl(var(--accent) / 0.9)',
          900: 'hsl(var(--accent) / 1)',
        },
        yellow: {
          50: 'hsl(var(--yellow) / 0.1)',
          100: 'hsl(var(--yellow) / 0.2)',
          200: 'hsl(var(--yellow) / 0.3)',
          300: 'hsl(var(--yellow) / 0.4)',
          400: 'hsl(var(--yellow) / 0.5)',
          500: 'hsl(var(--yellow) / 0.6)',
          600: 'hsl(var(--yellow) / 0.7)',
          700: 'hsl(var(--yellow) / 0.8)',
          800: 'hsl(var(--yellow) / 0.9)',
          900: 'hsl(var(--yellow) / 1)',
        },
        'neon-green': '#00ff00',
      },
      fontFamily: {
        arcade: ['"Press Start 2P"', 'cursive'],
      },
      animation: {
        'neon-pulse': 'neon-pulse 2s ease-in-out infinite',
      },
      keyframes: {
        'neon-pulse': {
          '0%, 100%': {
            textShadow: '0 0 7px #00ff00, 0 0 10px #00ff00, 0 0 21px #00ff00',
          },
          '50%': {
            textShadow: '0 0 10px #00ff00, 0 0 15px #00ff00, 0 0 30px #00ff00',
          },
        },
      },
    },
  },
  plugins: [],
} 