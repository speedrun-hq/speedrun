import { createRequire } from 'module';
import resolve from '@rollup/plugin-node-resolve';
import commonjs from '@rollup/plugin-commonjs';
import typescript from '@rollup/plugin-typescript';
import peerDepsExternal from 'rollup-plugin-peer-deps-external';
import path from 'path';

const require = createRequire(import.meta.url);
const pkg = require('./package.json');

const external = [
  ...Object.keys(pkg.dependencies || {}),
  ...Object.keys(pkg.peerDependencies || {}),
  'react/jsx-runtime',
  '@speedrun/components',
  '@speedrun/utils',
];

const plugins = [
  // Automatically externalize peer dependencies
  peerDepsExternal(),
  
  // Resolve node_modules
  resolve({
    browser: true,
    preferBuiltins: false,
    dedupe: ['react', 'react-dom'],
  }),
  
  // Convert CommonJS modules to ES6
  commonjs({
    include: /node_modules/,
  }),
  
  // Compile TypeScript
  typescript({
    tsconfig: './tsconfig.json',
    exclude: [
      'node_modules',
      '**/__tests__/**',
      '**/__mocks__/**',
      '../packages/**/*',
    ],
  }),
];

const onwarn = (warning, warn) => {
  // Ignore certain warnings
  if (warning.code === 'THIS_IS_UNDEFINED') return;
  if (warning.code === 'UNUSED_EXTERNAL_IMPORT') return;
  if (warning.code === 'MODULE_LEVEL_DIRECTIVE') return;
  if (warning.code === 'TS6059') return; // Ignore rootDir warnings for external packages
  if (warning.code === 'TS6307') return; // Ignore file list warnings
  if (warning.code === 'TS2353') return; // Ignore Chain type warning
  if (warning.code === 'TS2322') return; // Ignore type assignment warnings
  
  // Use default for everything else
  warn(warning);
};

export default {
  input: 'src/index.ts',
  output: [
    {
      file: pkg.main,
      format: 'cjs',
      sourcemap: true,
      exports: 'named',
      preserveModules: false,
    },
    {
      file: pkg.module,
      format: 'esm',
      sourcemap: true,
      exports: 'named',
      preserveModules: false,
    },
  ],
  external,
  plugins,
  onwarn,
}; 