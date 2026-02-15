import { defineConfig } from 'vitest/config';
import { svelte } from '@sveltejs/vite-plugin-svelte';
import { svelteTesting } from '@testing-library/svelte/vite';

export default defineConfig({
  plugins: [
    svelte({
      // Disable vite preprocessing for CSS to avoid Vite 6 compatibility issue
      // See: https://github.com/sveltejs/vite-plugin-svelte/issues/1043
      configFile: false,
      compilerOptions: {
        css: 'injected'
      }
    }),
    svelteTesting()
  ],
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/lib/test-setup.ts'],
    include: ['src/**/*.{test,spec}.{js,ts}']
  },
  resolve: {
    alias: {
      '$lib': '/src/lib'
    }
  },
  css: {
    // Don't process CSS in test mode
    modules: false
  }
});
