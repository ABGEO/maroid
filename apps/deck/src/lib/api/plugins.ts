import type { Page } from '@maroid/api-client';

import { client } from './client';
import type { Plugin } from './types';

export const plugins = {
	list: (): Promise<Plugin[] | null> =>
		client.get<Page<Plugin>>('/plugins').then((page) => page?.items ?? null)
};
