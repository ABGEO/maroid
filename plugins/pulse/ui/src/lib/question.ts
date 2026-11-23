import type { Question, QuestionPayload } from '../api';

/** The body that writes the question back as it is, with the archive flag given. */
export function payloadOf(question: Question, archived = question.archived): QuestionPayload {
	const payload: QuestionPayload = { type: question.type, prompt: question.prompt, archived };

	if (question.config !== undefined) {
		payload.config = question.config;
	}

	return payload;
}
