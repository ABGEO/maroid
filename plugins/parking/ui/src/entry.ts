import { defineRoute } from '@maroid/plugin-sdk';

export const routes = {
	'/overview': defineRoute(() => import('./pages/overview/Overview.svelte')),
	'/sessions': defineRoute(() => import('./pages/sessions/Sessions.svelte')),
	'/transactions': defineRoute(() => import('./pages/transactions/Transactions.svelte')),
	'/account': defineRoute(() => import('./pages/account/Account.svelte'))
};
