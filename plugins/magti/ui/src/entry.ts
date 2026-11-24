import { defineRoute } from '@maroid/plugin-sdk';

export const routes = {
	'/subscriber': defineRoute(() => import('./pages/subscriber/Subscriber.svelte')),
	'/traffic': defineRoute(() => import('./pages/traffic/Traffic.svelte')),
	'/money': defineRoute(() => import('./pages/money/Money.svelte')),
	'/account': defineRoute(() => import('./pages/account/Account.svelte'))
};
