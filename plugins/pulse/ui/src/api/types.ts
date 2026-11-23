export type QuestionType = 'scale' | 'boolean' | 'number' | 'choice' | 'text' | 'location';

export interface Settings {
	enabled: boolean;
	paused_until: string | null;
	updated_at: string;
}

export interface SettingsPayload {
	enabled: boolean;
	paused_until: string | null;
}

export interface ScaleConfig {
	min: number;
	max: number;
	low_label?: string;
	high_label?: string;
}

export interface NumberConfig {
	min: number;
	max: number;
	unit?: string;
	quick_values?: number[];
}

export interface Choice {
	/** Absent on a choice that a write adds. */
	id?: string;
	label: string;
}

export interface ChoiceConfig {
	choices: Choice[];
}

export type QuestionConfig = ScaleConfig | NumberConfig | ChoiceConfig;

export interface Value {
	id: string;
	occurred_at: string;
	answered_at: string;
	source: string;
	number?: number;
	boolean?: boolean;
	choice_id?: string;
	text?: string;
	latitude?: number;
	longitude?: number;
}

export interface Question {
	id: string;
	type: QuestionType;
	prompt: string;
	archived: boolean;
	/** Present on a scale, a number, and a choice. */
	config?: QuestionConfig;
	value_count: number;
	latest?: Value;
	created_at: string;
	updated_at: string;
}

export interface QuestionPayload {
	type: QuestionType;
	prompt: string;
	archived: boolean;
	config?: QuestionConfig;
}

export interface Checkin {
	id: string;
	name: string;
	/** HH:MM, local to the zone of the schedule. */
	local_times: string[];
	/** ISO weekdays, 1 for Monday. An empty list is every day. */
	weekdays: number[];
	enabled: boolean;
	question_ids: string[];
	created_at: string;
	updated_at: string;
}

export interface CheckinPayload {
	name: string;
	local_times: string[];
	weekdays: number[];
	enabled: boolean;
	question_ids: string[];
}

export interface Day {
	/** YYYY-MM-DD, local to the zone of the schedule. */
	date: string;
	count: number;
	mean?: number;
	min?: number;
	max?: number;
	yes_count?: number;
}

export type RoundState = 'offered' | 'started' | 'done' | 'declined' | 'expired';

export type RoundQuestionState = 'pending' | 'waiting' | 'answered' | 'skipped' | 'expired';

export interface RoundQuestion {
	/** Names a question of the questions list, active or archived. */
	question_id: string;
	state: RoundQuestionState;
	value?: Value;
}

export interface Round {
	id: string;
	/** Absent when the check-in was deleted. */
	checkin_id?: string;
	/** The name of the check-in when the round fired. */
	checkin_name: string;
	occurrence_at: string;
	state: RoundState;
	questions: RoundQuestion[];
}

export interface RoundFilter {
	checkin_id?: string;
	state?: RoundState;
}
