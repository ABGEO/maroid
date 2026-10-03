import { api, type Workspace } from '$lib/api';

const STORAGE_KEY = 'maroid.workspace';

export type WorkspaceStatus = 'idle' | 'loading' | 'ready' | 'error';

export const workspaceState = $state<{ status: WorkspaceStatus; workspaces: Workspace[] }>({
	status: 'idle',
	workspaces: []
});

/** Loads the workspaces of the person. A second call reloads them after a change. */
export async function loadWorkspaces(): Promise<void> {
	workspaceState.status = 'loading';

	try {
		const list = await api.workspaces.list();
		if (list === null) {
			return;
		}

		workspaceState.workspaces = list;
		workspaceState.status = 'ready';
	} catch (error) {
		console.error('Failed to load the workspaces', error);
		workspaceState.workspaces = [];
		workspaceState.status = 'error';
	}
}

/** Remembers the workspace for this browser. A browser that refuses the storage remembers nothing. */
export function rememberWorkspace(workspaceId: string): void {
	try {
		localStorage.setItem(STORAGE_KEY, workspaceId);
	} catch {
		// The storage is unavailable in a private window. The first workspace opens instead.
	}
}

function remembered(): string | null {
	try {
		return localStorage.getItem(STORAGE_KEY);
	} catch {
		return null;
	}
}

/**
 * The workspace to open after a sign in: the one this browser remembers, while the
 * person is still a member of it, or else the first of the list.
 */
export function landingWorkspace(workspaces: Workspace[]): Workspace | null {
	const rememberedId = remembered();

	return workspaces.find((workspace) => workspace.id === rememberedId) ?? workspaces[0] ?? null;
}

/** The workspace of the list with the identifier, or undefined. */
export function workspaceById(workspaceId: string | undefined): Workspace | undefined {
	return workspaceState.workspaces.find((workspace) => workspace.id === workspaceId);
}

/**
 * The workspace that a page acts in: the one that its address names, or else the
 * one to land on. A person with no workspace has none.
 */
export function actingWorkspaceId(addressed: string | undefined): string | null {
	return addressed ?? landingWorkspace(workspaceState.workspaces)?.id ?? null;
}
