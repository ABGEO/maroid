import adapter from '@sveltejs/adapter-node';

/** @type {import('@sveltejs/kit/vite').Config} */
const config = {
	adapter: adapter(),
	// SvelteKit 3 no longer provides $lib. Remove this alias once the imports use #lib.
	alias: {
		$lib: 'src/lib'
	},
	compilerOptions: {
		// Force runes mode for the project, except for libraries. Can be removed in svelte 6.
		runes: ({ filename }) => (filename.split(/[/\\]/).includes('node_modules') ? undefined : true)
	}
};

export default config;
