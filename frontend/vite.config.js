import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import { readFileSync } from 'fs';
import { resolve } from 'path';
import obfuscator from 'vite-plugin-javascript-obfuscator';

const packageJson = JSON.parse(readFileSync(resolve(__dirname, 'package.json'), 'utf-8'));

// https://vite.dev/config/
export default defineConfig(({ mode }) => {
  return {
    plugins: [
      react(),
      mode === 'production' ? obfuscator({
        options: {
          compact: true,
          controlFlowFlattening: true,
          deadCodeInjection: true,
          numbersToExpressions: true,
          simplify: true,
          stringArrayShuffle: true,
        },
      }) : null,
    ].filter(Boolean),
    root: '.',
    define: {
      'process.env.APP_VERSION': JSON.stringify(packageJson.version),
    },
    server: {
      host: true,
      port: 5000,
    },
    preview: {
      port: 5000,
      host: '0.0.0.0',
    },
    build: {
      outDir: './dist',
      emptyOutDir: true,
      rollupOptions: {
        input: {
          main: resolve(__dirname, 'index.html'),
          privacy: resolve(__dirname, 'privacy.html'),
        },
      },
    },
    test: {
      globals: true,
      environment: 'jsdom',
      setupFiles: './src/setupTests.js',
    },
  };
})
