import type {
	ChoiceConfig,
	NumberConfig,
	Question,
	QuestionType,
	ScaleConfig,
	Value
} from '../api';

export const TYPE_LABELS: Record<QuestionType, string> = {
	scale: 'Scale',
	boolean: 'Yes or no',
	number: 'Number',
	choice: 'Choice',
	text: 'Text',
	location: 'Location'
};

const WEEKDAY_LABELS = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun'];

/** The short name of an ISO weekday, 1 for Monday. */
export function weekdayLabel(day: number): string {
	return WEEKDAY_LABELS[day - 1] ?? String(day);
}

/** The weekdays of a check-in as a person reads them. An empty list is every day. */
export function weekdaysText(days: number[]): string {
	if (days.length === 0 || days.length === 7) {
		return 'Every day';
	}

	return [...days]
		.sort((a, b) => a - b)
		.map(weekdayLabel)
		.join(', ');
}

export function scaleOf(question: Question): ScaleConfig | undefined {
	return question.type === 'scale' ? (question.config as ScaleConfig) : undefined;
}

export function numberOf(question: Question): NumberConfig | undefined {
	return question.type === 'number' ? (question.config as NumberConfig) : undefined;
}

export function choicesOf(question: Question): ChoiceConfig['choices'] {
	return question.type === 'choice' ? (question.config as ChoiceConfig).choices : [];
}

// The pages show a point with five decimals, about one metre.
const COORDINATE_DECIMALS = 5;

export function coordinates(value: Value): string {
	return `${value.latitude?.toFixed(COORDINATE_DECIMALS)}, ${value.longitude?.toFixed(COORDINATE_DECIMALS)}`;
}

export function mapLink(value: Value): string {
	const lat = value.latitude?.toFixed(COORDINATE_DECIMALS);
	const lon = value.longitude?.toFixed(COORDINATE_DECIMALS);

	return `https://www.openstreetmap.org/?mlat=${lat}&mlon=${lon}#map=16/${lat}/${lon}`;
}

/** A value as one line of text. A location reads as its coordinates. */
export function valueText(question: Question, value: Value): string {
	switch (question.type) {
		case 'scale':
			return `${value.number} of ${scaleOf(question)?.max}`;
		case 'number': {
			const unit = numberOf(question)?.unit;

			return unit ? `${value.number} ${unit}` : String(value.number);
		}
		case 'boolean':
			return value.boolean ? 'Yes' : 'No';
		case 'choice':
			return choicesOf(question).find((choice) => choice.id === value.choice_id)?.label ?? '';
		case 'text':
			return value.text ?? '';
		case 'location':
			return coordinates(value);
	}
}
