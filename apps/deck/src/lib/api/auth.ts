import { client } from './client';
import type { SignOut, User } from './types';

export const auth = {
	me: (): Promise<User | null> => client.get<User>('/auth/me'),

	logout: (redirect: string): Promise<SignOut | null> =>
		client.post<SignOut>('/auth/logout', undefined, { params: { redirect } })
};
