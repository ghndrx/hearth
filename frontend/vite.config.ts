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
		},
		// COOP/COEP headers required for SharedArrayBuffer (used by libsignal-client WASM)
		headers: {
			'Cross-Origin-Opener-Policy': 'same-origin',
			'Cross-Origin-Embedder-Policy': 'require-corp'
		}
	},
	
	build: {
		target: 'esnext',
		minify: 'esbuild',
		sourcemap: true,
		chunkSizeWarningLimit: 600,
		rollupOptions: {
			output: {
				manualChunks(id) {
					// Split livekit-client into its own chunk (large SDK)
					if (id.includes('livekit-client')) {
						return 'livekit';
					}
					// Keep crypto separate
					if (id.includes('/lib/crypto/')) {
						return 'crypto';
					}
					// Voice-related components in their own chunk
					if (id.includes('/voice/LiveKit') || id.includes('VoiceChannel') || 
					    id.includes('VoiceControls') || id.includes('VoiceConnectedBar')) {
						return 'voice';
					}
				}
			}
		}
	},
	
	optimizeDeps: {
		exclude: ['@sveltejs/kit']
	},
	
	ssr: {
		external: ['livekit-client']
	}
});
