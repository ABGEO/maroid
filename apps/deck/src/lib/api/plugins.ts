import { client } from './client';
import type { Plugin } from './types';

export const plugins = {
	list: (): Promise<Plugin[] | null> => client.get<Plugin[]>('/plugins')
};
