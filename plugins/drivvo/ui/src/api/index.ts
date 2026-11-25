import type { PluginHost, Page, Tagged } from '@maroid/plugin-sdk';

import type {
	Connection,
	Kind,
	LogEntry,
	LogEntryItem,
	LogEntryTotals,
	Overview,
	Period,
	Reminder,
	Vehicle
} from './types';

export type * from './types';

const CONNECTION = '/connections/self';

/** The path with the period and each kind as its query. */
function withQuery(path: string, period: Period, kinds: Kind[] = []): string {
	const query = new URLSearchParams();

	for (const kind of kinds) {
		query.append('kind', kind);
	}

	if (period.from !== undefined) {
		query.set('from', period.from);
	}

	if (period.to !== undefined) {
		query.set('to', period.to);
	}

	const text = query.toString();

	return text === '' ? path : `${path}?${text}`;
}

export function createCarLogApi(host: PluginHost) {
	const { api } = host;
	const vehicle = (id: string) => `/vehicles/${encodeURIComponent(id)}`;

	return {
		connection: {
			read: (): Promise<Tagged<Connection> | null> => api.getTagged<Connection>(CONNECTION),
			connect: () => api.put<Connection>(CONNECTION),
			disconnect: () => api.del<null>(CONNECTION)
		},

		vehicles: () => api.get<Page<Vehicle>>('/vehicles'),

		overview: (id: string, period: Period) =>
			api.get<Overview>(withQuery(`${vehicle(id)}/overview`, period)),

		/** Reads the first page of the kinds in the period, or the page that a link names. */
		logEntries: (id: string, period: Period, kinds: Kind[], link?: string) =>
			link
				? api.follow<Page<LogEntryItem>>(link)
				: api.get<Page<LogEntryItem>>(withQuery(`${vehicle(id)}/log-entries`, period, kinds)),

		logEntryTotals: (id: string, period: Period, kind: Kind) =>
			api.get<LogEntryTotals>(withQuery(`${vehicle(id)}/log-entry-totals`, period, [kind])),

		logEntry: (id: string, entryId: string) =>
			api.get<LogEntry>(`${vehicle(id)}/log-entries/${encodeURIComponent(entryId)}`),

		reminders: (id: string) => api.get<Page<Reminder>>(`${vehicle(id)}/reminders`)
	};
}

export type CarLogApi = ReturnType<typeof createCarLogApi>;
