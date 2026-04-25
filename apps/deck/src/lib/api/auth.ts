import { get } from './client';
import type { User } from './types';

export const auth = {
	me: (): Promise<User | null> => get<User>('/auth/me')
};
