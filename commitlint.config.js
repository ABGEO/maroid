import { existsSync, readdirSync } from 'node:fs';

const moduleParents = ['apps', 'libs', 'plugins', 'firmware'];

const moduleScopes = moduleParents
	.filter((parent) => existsSync(parent))
	.flatMap((parent) =>
		readdirSync(parent, { withFileTypes: true })
			.filter((entry) => entry.isDirectory())
			.map((entry) => entry.name)
	);

/** @type {import('@commitlint/types').UserConfig} */
export default {
	extends: ['@commitlint/config-conventional'],
	rules: {
		'type-enum': [
			2,
			'always',
			['feat', 'fix', 'refactor', 'perf', 'test', 'docs', 'build', 'chore']
		],
		'scope-enum': [2, 'always', [...new Set([...moduleScopes, 'specs', 'deps'])].sort()],
		'header-max-length': [2, 'always', 72],
		'body-empty': [2, 'always'],
		'footer-empty': [2, 'always']
	}
};
