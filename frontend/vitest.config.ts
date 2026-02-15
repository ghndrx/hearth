import { defineConfig } from 'vitest/config';
import { svelte } from '@sveltejs/vite-plugin-svelte';
import { svelteTesting } from '@testing-library/svelte/vite';

export default defineConfig({
  plugins: [
    svelte({
      compilerOptions: {
        dev: true
      },
      preprocess: []
    }),
    svelteTesting()
  ],
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/lib/test-setup.ts'],
    include: ['src/**/*.{test,spec}.{js,ts}'],
    css: {
      include: [],
      exclude: []
    },
    server: {
      deps: {
        inline: [/svelte/]
      }
    }
  },
  resolve: {
    alias: {
      '$lib': '/src/lib',
      '$app/environment': '/src/lib/__mocks__/environment.ts'
    }
  }
});
