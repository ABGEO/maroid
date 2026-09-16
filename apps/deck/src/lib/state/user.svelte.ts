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

/** The two names of the record, joined for display. Falls back when the owner set neither. */
export function displayName(user: User | null): string {
	const name = [user?.first_name, user?.last_name].filter(Boolean).join(' ');

	return name || 'Maroid User';
}
