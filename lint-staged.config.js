/** @type {import('lint-staged').Configuration} */
export default {
	'*.{js,cjs,mjs,ts,svelte,css,html,json,yaml,yml,md}': 'prettier --write --ignore-unknown',
	'*.go': 'golangci-lint fmt'
};
