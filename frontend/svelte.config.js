import adapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	preprocess: vitePreprocess(),

	kit: {
		adapter: adapter({
			pages: 'build',
			assets: 'build',
			fallback: 'index.html',
			precompress: false,
			strict: true
		}),
		alias: {
			$lib: 'src/lib',
			$stores: 'src/lib/stores'
		},
		csp: {
			mode: 'auto',
			directives: {
				'default-src': ['self'],
				'script-src': ['self'],
				'style-src': ['self', 'unsafe-inline'],
				'img-src': ['self', 'data:', 'https:'],
				'connect-src': ['self', 'wss:', 'https:'],
				'font-src': ['self'],
				'object-src': ['none'],
				'frame-ancestors': ['self'],
				'base-uri': ['self'],
				'form-action': ['self']
			}
		},
		paths: {
			base: ''
		},
		prerender: {
			entries: []
		}
	}
};

export default config;
