import { defineConfig } from 'vite';

import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { federation } from '@module-federation/vite';

import kitConfig from './kit.config.js';

export default defineConfig({
	plugins: [
		tailwindcss(),
		sveltekit(kitConfig),
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
