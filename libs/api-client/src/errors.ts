import { isProblem, type FieldFailure, type Problem } from './problem';

export class ApiError extends Error {
  readonly status: number;
  readonly statusText: string;
  readonly body: unknown;
  /** The problem that the hub answered, or null for a body of another shape. */
  readonly problem: Problem | null;

  constructor(status: number, statusText: string, body: unknown) {
    super(`API request failed: ${status} ${statusText}`);
    this.name = 'ApiError';
    this.status = status;
    this.statusText = statusText;
    this.body = body;
    this.problem = isProblem(body) ? body : null;
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
