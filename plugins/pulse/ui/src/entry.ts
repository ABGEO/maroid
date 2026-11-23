import { defineRoute } from '@maroid/plugin-sdk';

export const routes = {
	'/data': defineRoute(() => import('./pages/data/Data.svelte')),
	'/history': defineRoute(() => import('./pages/history/History.svelte')),
	'/history/:checkin': defineRoute(() => import('./pages/history/History.svelte')),
	'/checkins': defineRoute(() => import('./pages/checkins/Checkins.svelte')),
	'/questions': defineRoute(() => import('./pages/questions/Questions.svelte')),
	'/preferences': defineRoute(() => import('./pages/settings/Settings.svelte'))
};
