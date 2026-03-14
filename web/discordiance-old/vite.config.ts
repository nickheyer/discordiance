import tailwindcss from '@tailwindcss/vite';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [tailwindcss(), sveltekit()],
	server: {
		proxy: {
			'/discordiance.v1': {
				target: 'http://localhost:8067',
				changeOrigin: true
			}
		}
	}
});
