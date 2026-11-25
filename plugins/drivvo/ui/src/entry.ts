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
const entry = () => import('./pages/Entry.svelte');

/** The navigation entry, then its page for one vehicle, then the detail of a listed entry. */
function vehiclePages(
	page: string,
	title: string,
	load: () => Promise<{ default: Component<never> }>,
	fixed: Record<string, unknown> = {},
	listsEntries = true
): Record<string, RouteMounter> {
	const routes: Record<string, RouteMounter> = {
		[`/${page}`]: routeWith(landing, { page, title }),
		[`/${page}/:vehicle`]: routeWith(load, fixed)
	};

	if (listsEntries) {
		routes[`/${page}/:vehicle/:entry`] = routeWith(entry, { page, title });
	}

	return routes;
}

export const routes = {
	...vehiclePages('overview', 'Overview', () => import('./pages/Overview.svelte'), {}, false),
	...vehiclePages('history', 'History', () => import('./pages/History.svelte')),
	...vehiclePages('refuels', 'Refuels', entryList, { kind: 'refuel' }),
	...vehiclePages('maintenances', 'Maintenance', entryList, { kind: 'maintenance' }),
	...vehiclePages('expenses', 'Expenses', entryList, { kind: 'expense' }),
	...vehiclePages('incomes', 'Income', entryList, { kind: 'income' }),
	...vehiclePages('reminders', 'Reminders', () => import('./pages/Reminders.svelte'), {}, false),
	'/account': defineRoute(() => import('./pages/Account.svelte'))
};
