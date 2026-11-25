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

/** The query of a period and of the kinds. An absent value leaves the parameter out. */
function filterParams(period: Period, kinds: Kind[] = []) {
	return {
		kind: kinds.length > 0 ? kinds.join(',') : undefined,
		from: period.from,
		to: period.to
	};
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
			api.get<Overview>(`${vehicle(id)}/overview`, { params: filterParams(period) }),

		/** Reads the first page of the kinds in the period, or the page that a link names. */
		logEntries: (id: string, period: Period, kinds: Kind[], link?: string) =>
			link
				? api.follow<Page<LogEntryItem>>(link)
				: api.get<Page<LogEntryItem>>(`${vehicle(id)}/log-entries`, {
						params: filterParams(period, kinds)
					}),

		logEntryTotals: (id: string, period: Period, kind: Kind) =>
			api.get<LogEntryTotals>(`${vehicle(id)}/log-entry-totals`, {
				params: filterParams(period, [kind])
			}),

		logEntry: (id: string, entryId: string) =>
			api.get<LogEntry>(`${vehicle(id)}/log-entries/${encodeURIComponent(entryId)}`),

		reminders: (id: string) => api.get<Page<Reminder>>(`${vehicle(id)}/reminders`)
	};
}

export type CarLogApi = ReturnType<typeof createCarLogApi>;
