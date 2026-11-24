import { isProblem, type FieldFailure, type Problem } from './problem';

const API_ERROR: unique symbol = Symbol.for('maroid.api-client.ApiError');

export class ApiError extends Error {
	/**
	 * A plugin bundles its own copy of this class, and the deck throws its copy, so
	 * `instanceof` reads a brand that every copy shares instead of the class.
	 */
	static [Symbol.hasInstance](value: unknown): boolean {
		return typeof value === 'object' && value !== null && API_ERROR in value;
	}

	readonly [API_ERROR] = true;

	readonly status: number;
	readonly statusText: string;
	readonly body: unknown;
	/** The problem that the hub answered, or null for a body of another shape. */
	readonly problem: Problem | null;
	/**
	 * The flow identifier of the request, bare. A person reports it and one
	 * search over the log reaches the line that holds the cause.
	 */
	readonly flowId: string;

	constructor(status: number, statusText: string, body: unknown, flowId = '') {
		super(`API request failed: ${status} ${statusText}`);
		this.name = 'ApiError';
		this.status = status;
		this.statusText = statusText;
		this.body = body;
		this.problem = isProblem(body) ? body : null;
		this.flowId = flowId;
	}

	/** The problem type, or the empty string when the body carries no problem. */
	get type(): string {
		return this.problem?.type ?? '';
	}

	/** Reports whether the hub answered with this problem type. */
	is(type: string): boolean {
		return this.problem?.type === type;
	}

	/** The fields that caused a rejection. */
	get fields(): FieldFailure[] {
		return this.problem?.errors ?? [];
	}
}
