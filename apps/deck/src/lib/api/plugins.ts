import { get } from './client';
import type { Plugin } from './types';

export const plugins = {
	list: (): Promise<Plugin[] | null> => get<Plugin[]>('/plugins')
};
