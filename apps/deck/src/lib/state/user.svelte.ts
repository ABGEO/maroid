import { api, type User } from '$lib/api';

export const userState = $state<{ user: User | null }>({
	user: null
});

export async function loadUser(): Promise<void> {
	try {
		userState.user = await api.auth.me();
	} catch (error) {
		console.error('Failed to load current user', error);
		userState.user = null;
	}
}
