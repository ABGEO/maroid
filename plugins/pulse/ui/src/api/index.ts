import type { ApiClient, Page, PluginHost, RequestOptions } from '@maroid/plugin-sdk';

import type {
	Checkin,
	CheckinPayload,
	Day,
	Question,
	QuestionPayload,
	Settings,
	SettingsPayload,
	Value
} from './types';

export type * from './types';

/** A range of local days, both included, as YYYY-MM-DD. */
export interface DayRange {
	from: string;
	to: string;
}

function createSettingsApi(api: ApiClient) {
	return {
		get: () => api.getTagged<Settings>('/settings'),
		put: (payload: SettingsPayload, ifMatch?: string) =>
			api.put<Settings>('/settings', payload, { ifMatch })
	};
}

function createQuestionsApi(api: ApiClient) {
	return {
		list: (archived: boolean, link?: string) =>
			link
				? api.follow<Page<Question>>(link)
				: api.get<Page<Question>>('/questions', { params: { archived: String(archived) } }),
		get: (id: string) => api.getTagged<Question>(`/questions/${id}`),
		create: (payload: QuestionPayload, options?: RequestOptions) =>
			api.post<Question>('/questions', payload, options),
		update: (id: string, payload: QuestionPayload, ifMatch?: string) =>
			api.put<Question>(`/questions/${id}`, payload, { ifMatch }),
		remove: (id: string) => api.del<null>(`/questions/${id}`),
		values: (id: string, range: DayRange, link?: string) =>
			link
				? api.follow<Page<Value>>(link)
				: api.get<Page<Value>>(`/questions/${id}/values`, { params: { ...range } }),
		days: (id: string, range: DayRange) =>
			api.get<{ items: Day[] }>(`/questions/${id}/days`, { params: { ...range } })
	};
}

function createCheckinsApi(api: ApiClient) {
	return {
		list: (link?: string) =>
			link ? api.follow<Page<Checkin>>(link) : api.get<Page<Checkin>>('/checkins'),
		get: (id: string) => api.getTagged<Checkin>(`/checkins/${id}`),
		create: (payload: CheckinPayload, options?: RequestOptions) =>
			api.post<Checkin>('/checkins', payload, options),
		update: (id: string, payload: CheckinPayload, ifMatch?: string) =>
			api.put<Checkin>(`/checkins/${id}`, payload, { ifMatch }),
		remove: (id: string) => api.del<null>(`/checkins/${id}`)
	};
}

export function createPulseApi(host: PluginHost) {
	return {
		settings: createSettingsApi(host.api),
		questions: createQuestionsApi(host.api),
		checkins: createCheckinsApi(host.api)
	};
}

export type PulseApi = ReturnType<typeof createPulseApi>;
