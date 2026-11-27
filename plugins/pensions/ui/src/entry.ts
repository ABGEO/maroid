import { defineRoute } from '@maroid/plugin-sdk';

export const routes = {
	'/overview': defineRoute(() => import('./pages/overview/Overview.svelte')),
	'/contributions': defineRoute(() => import('./pages/contributions/Contributions.svelte')),
	'/portfolios': defineRoute(() => import('./pages/portfolios/Portfolios.svelte')),
	'/account': defineRoute(() => import('./pages/account/Account.svelte'))
};
