import { defineConfig } from 'vitest/config';
import { svelte } from '@sveltejs/vite-plugin-svelte';
import { svelteTesting } from '@testing-library/svelte/vite';

export default defineConfig({
  plugins: [
    svelte({
      // Disable CSS preprocessing in tests to avoid Vite 6 PartialEnvironment issues
      // Tests don't need actual CSS processing
      compilerOptions: {
        css: 'injected'
      },
      // Skip vitePreprocess for tests - it requires Vite's server environment
      preprocess: []
    }),
    svelteTesting()
  ],
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/lib/test-setup.ts'],
    include: ['src/**/*.{test,spec}.{js,ts}'],
    // Ensure CSS imports are handled as empty modules in tests
    css: false
  },
  resolve: {
    alias: {
      '$lib': '/src/lib',
      '$app': '/src/lib/__mocks__/$app'
    }
  }
});
