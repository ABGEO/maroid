import { client } from './client';
import type { User } from './types';

export const auth = {
	me: (): Promise<User | null> => client.get<User>('/auth/me')
};
