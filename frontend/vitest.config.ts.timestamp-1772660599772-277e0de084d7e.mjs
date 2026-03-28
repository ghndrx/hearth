// vitest.config.ts
import { defineConfig } from "file:///home/administrator/clawd/hearth/frontend/node_modules/vitest/dist/config.js";
import { svelte } from "file:///home/administrator/clawd/hearth/frontend/node_modules/@sveltejs/vite-plugin-svelte/src/index.js";
import { svelteTesting } from "file:///home/administrator/clawd/hearth/frontend/node_modules/@testing-library/svelte/src/vite.js";
var vitest_config_default = defineConfig({
  plugins: [
    svelte({
      // Disable CSS preprocessing in tests to avoid Vite 6 PartialEnvironment issues
      // Tests don't need actual CSS processing
      compilerOptions: {
        css: "injected"
      },
      // Skip vitePreprocess for tests - it requires Vite's server environment
      preprocess: []
    }),
    svelteTesting()
  ],
  test: {
    environment: "jsdom",
    globals: true,
    setupFiles: ["./src/lib/test-setup.ts"],
    include: ["src/**/*.{test,spec}.{js,ts}"],
    // Ensure CSS imports are handled as empty modules in tests
    css: false
  },
  resolve: {
    alias: {
      "$lib": "/src/lib",
      "$app": "/src/lib/__mocks__/$app"
    }
  }
});
export {
  vitest_config_default as default
};
//# sourceMappingURL=data:application/json;base64,ewogICJ2ZXJzaW9uIjogMywKICAic291cmNlcyI6IFsidml0ZXN0LmNvbmZpZy50cyJdLAogICJzb3VyY2VzQ29udGVudCI6IFsiY29uc3QgX192aXRlX2luamVjdGVkX29yaWdpbmFsX2Rpcm5hbWUgPSBcIi9ob21lL2FkbWluaXN0cmF0b3IvY2xhd2QvaGVhcnRoL2Zyb250ZW5kXCI7Y29uc3QgX192aXRlX2luamVjdGVkX29yaWdpbmFsX2ZpbGVuYW1lID0gXCIvaG9tZS9hZG1pbmlzdHJhdG9yL2NsYXdkL2hlYXJ0aC9mcm9udGVuZC92aXRlc3QuY29uZmlnLnRzXCI7Y29uc3QgX192aXRlX2luamVjdGVkX29yaWdpbmFsX2ltcG9ydF9tZXRhX3VybCA9IFwiZmlsZTovLy9ob21lL2FkbWluaXN0cmF0b3IvY2xhd2QvaGVhcnRoL2Zyb250ZW5kL3ZpdGVzdC5jb25maWcudHNcIjtpbXBvcnQgeyBkZWZpbmVDb25maWcgfSBmcm9tICd2aXRlc3QvY29uZmlnJztcbmltcG9ydCB7IHN2ZWx0ZSB9IGZyb20gJ0BzdmVsdGVqcy92aXRlLXBsdWdpbi1zdmVsdGUnO1xuaW1wb3J0IHsgc3ZlbHRlVGVzdGluZyB9IGZyb20gJ0B0ZXN0aW5nLWxpYnJhcnkvc3ZlbHRlL3ZpdGUnO1xuXG5leHBvcnQgZGVmYXVsdCBkZWZpbmVDb25maWcoe1xuICBwbHVnaW5zOiBbXG4gICAgc3ZlbHRlKHtcbiAgICAgIC8vIERpc2FibGUgQ1NTIHByZXByb2Nlc3NpbmcgaW4gdGVzdHMgdG8gYXZvaWQgVml0ZSA2IFBhcnRpYWxFbnZpcm9ubWVudCBpc3N1ZXNcbiAgICAgIC8vIFRlc3RzIGRvbid0IG5lZWQgYWN0dWFsIENTUyBwcm9jZXNzaW5nXG4gICAgICBjb21waWxlck9wdGlvbnM6IHtcbiAgICAgICAgY3NzOiAnaW5qZWN0ZWQnXG4gICAgICB9LFxuICAgICAgLy8gU2tpcCB2aXRlUHJlcHJvY2VzcyBmb3IgdGVzdHMgLSBpdCByZXF1aXJlcyBWaXRlJ3Mgc2VydmVyIGVudmlyb25tZW50XG4gICAgICBwcmVwcm9jZXNzOiBbXVxuICAgIH0pLFxuICAgIHN2ZWx0ZVRlc3RpbmcoKVxuICBdLFxuICB0ZXN0OiB7XG4gICAgZW52aXJvbm1lbnQ6ICdqc2RvbScsXG4gICAgZ2xvYmFsczogdHJ1ZSxcbiAgICBzZXR1cEZpbGVzOiBbJy4vc3JjL2xpYi90ZXN0LXNldHVwLnRzJ10sXG4gICAgaW5jbHVkZTogWydzcmMvKiovKi57dGVzdCxzcGVjfS57anMsdHN9J10sXG4gICAgLy8gRW5zdXJlIENTUyBpbXBvcnRzIGFyZSBoYW5kbGVkIGFzIGVtcHR5IG1vZHVsZXMgaW4gdGVzdHNcbiAgICBjc3M6IGZhbHNlXG4gIH0sXG4gIHJlc29sdmU6IHtcbiAgICBhbGlhczoge1xuICAgICAgJyRsaWInOiAnL3NyYy9saWInLFxuICAgICAgJyRhcHAnOiAnL3NyYy9saWIvX19tb2Nrc19fLyRhcHAnXG4gICAgfVxuICB9XG59KTtcbiJdLAogICJtYXBwaW5ncyI6ICI7QUFBaVQsU0FBUyxvQkFBb0I7QUFDOVUsU0FBUyxjQUFjO0FBQ3ZCLFNBQVMscUJBQXFCO0FBRTlCLElBQU8sd0JBQVEsYUFBYTtBQUFBLEVBQzFCLFNBQVM7QUFBQSxJQUNQLE9BQU87QUFBQTtBQUFBO0FBQUEsTUFHTCxpQkFBaUI7QUFBQSxRQUNmLEtBQUs7QUFBQSxNQUNQO0FBQUE7QUFBQSxNQUVBLFlBQVksQ0FBQztBQUFBLElBQ2YsQ0FBQztBQUFBLElBQ0QsY0FBYztBQUFBLEVBQ2hCO0FBQUEsRUFDQSxNQUFNO0FBQUEsSUFDSixhQUFhO0FBQUEsSUFDYixTQUFTO0FBQUEsSUFDVCxZQUFZLENBQUMseUJBQXlCO0FBQUEsSUFDdEMsU0FBUyxDQUFDLDhCQUE4QjtBQUFBO0FBQUEsSUFFeEMsS0FBSztBQUFBLEVBQ1A7QUFBQSxFQUNBLFNBQVM7QUFBQSxJQUNQLE9BQU87QUFBQSxNQUNMLFFBQVE7QUFBQSxNQUNSLFFBQVE7QUFBQSxJQUNWO0FBQUEsRUFDRjtBQUNGLENBQUM7IiwKICAibmFtZXMiOiBbXQp9Cg==
