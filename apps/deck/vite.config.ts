import { defineConfig } from 'vite';

import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { federation } from '@module-federation/vite';

export default defineConfig({
	plugins: [
		tailwindcss(),
		sveltekit(),
		federation({
			dts: false,
			dev: { remoteHmr: true },
			name: 'deck',
			filename: 'remoteEntry.js',
			remotes: {},
			exposes: {},
			shared: []
		})
	]
});
