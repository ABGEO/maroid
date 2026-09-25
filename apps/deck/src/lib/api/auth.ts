import { client } from './client';
import type { SignOut, User } from './types';

export const auth = {
	me: (): Promise<User | null> => client.get<User>('/auth/sessions/self'),

	logout: (redirect: string): Promise<SignOut | null> =>
		client.del<SignOut>('/auth/sessions/self', { params: { redirect } })
};
