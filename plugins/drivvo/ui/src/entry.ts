import { defineRoute, type RouteMounter } from '@maroid/plugin-sdk';
import type { Component } from 'svelte';

/** A route that mounts the page with the props of the deck and the fixed props. */
function routeWith(
	load: () => Promise<{ default: Component<never> }>,
	fixed: Record<string, unknown>
): RouteMounter {
	const mount = defineRoute(load as () => Promise<{ default: Component }>);

	return (target, props = {}) => mount(target, { ...props, ...fixed });
}

const landing = () => import('./pages/Landing.svelte');
const entryList = () => import('./pages/EntryList.svelte');

export const routes = {
	'/overview': routeWith(landing, { page: 'overview', title: 'Overview' }),
	'/history': routeWith(landing, { page: 'history', title: 'History' }),
	'/refuels': routeWith(landing, { page: 'refuels', title: 'Refuels' }),
	'/maintenances': routeWith(landing, { page: 'maintenances', title: 'Maintenance' }),
	'/expenses': routeWith(landing, { page: 'expenses', title: 'Expenses' }),
	'/incomes': routeWith(landing, { page: 'incomes', title: 'Income' }),
	'/reminders': routeWith(landing, { page: 'reminders', title: 'Reminders' }),
	'/account': defineRoute(() => import('./pages/Account.svelte')),
	'/vehicles/:vehicle/overview': defineRoute(() => import('./pages/Overview.svelte')),
	'/vehicles/:vehicle/history': defineRoute(() => import('./pages/History.svelte')),
	'/vehicles/:vehicle/refuels': routeWith(entryList, { kind: 'refuel' }),
	'/vehicles/:vehicle/maintenances': routeWith(entryList, { kind: 'maintenance' }),
	'/vehicles/:vehicle/expenses': routeWith(entryList, { kind: 'expense' }),
	'/vehicles/:vehicle/incomes': routeWith(entryList, { kind: 'income' }),
	'/vehicles/:vehicle/entries/:entry': defineRoute(() => import('./pages/Entry.svelte')),
	'/vehicles/:vehicle/reminders': defineRoute(() => import('./pages/Reminders.svelte'))
};
