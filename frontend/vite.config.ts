import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [sveltekit()],
	
	server: {
		port: 5173,
		proxy: {
			'/api': {
				target: 'http://localhost:8080',
				changeOrigin: true
			},
			'/gateway': {
				target: 'ws://localhost:8080',
				ws: true
			}
		}
	},
	
	build: {
		target: 'esnext',
		minify: 'esbuild',
		sourcemap: true,
		rollupOptions: {
			output: {
				manualChunks: {
					crypto: ['./src/lib/crypto/keys.ts', './src/lib/crypto/encryption.ts']
				}
			}
		}
	},
	
	optimizeDeps: {
		exclude: ['@sveltejs/kit']
	}
});
