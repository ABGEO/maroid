import type { Page, Tagged } from '@maroid/api-client';

import { client } from './client';
import type { Candidate, Member, Workspace } from './types';

function path(workspaceId: string): string {
	return `/workspaces/${encodeURIComponent(workspaceId)}`;
}

function memberPath(workspaceId: string, userId: string): string {
	return `${path(workspaceId)}/members/${encodeURIComponent(userId)}`;
}

export const workspaces = {
	list: (): Promise<Workspace[] | null> =>
		client.get<Page<Workspace>>('/workspaces').then((page) => page?.items ?? null),

	get: (workspaceId: string): Promise<Tagged<Workspace> | null> =>
		client.getTagged<Workspace>(path(workspaceId)),

	create: (name: string, idempotencyKey: string): Promise<Workspace | null> =>
		client.post<Workspace>('/workspaces', { name }, { idempotencyKey }),

	rename: (workspaceId: string, name: string, ifMatch?: string): Promise<Workspace | null> =>
		client.patch<Workspace>(path(workspaceId), { name }, { ifMatch }),

	members: (workspaceId: string): Promise<Member[] | null> =>
		client.get<Page<Member>>(`${path(workspaceId)}/members`).then((page) => page?.items ?? null),

	candidates: (workspaceId: string): Promise<Candidate[] | null> =>
		client
			.get<Page<Candidate>>(`${path(workspaceId)}/member-candidates`)
			.then((page) => page?.items ?? null),

	addMember: (
		workspaceId: string,
		userId: string,
		idempotencyKey: string
	): Promise<Member | null> =>
		client.post<Member>(`${path(workspaceId)}/members`, { user_id: userId }, { idempotencyKey }),

	removeMember: (workspaceId: string, userId: string): Promise<void> =>
		client.del<void>(memberPath(workspaceId, userId)).then(() => undefined)
};
