/** The media type of every error response of the hub. */
export const PROBLEM_MEDIA_TYPE = 'application/problem+json';

/** One field that caused a rejection. */
export interface FieldFailure {
  /** The reason, as a sentence a person reads. */
  detail: string;
  /** A JSON Pointer in the fragment form of RFC 6901, so it starts with `#`. */
  pointer: string;
}

/** The body of an error response. It obeys RFC 9457. */
export interface Problem {
  /** The problem type, a URN of the form urn:maroid:problem:<owner>:<slug>. */
  type: string;
  title: string;
  status: number;
  detail?: string;
  instance?: string;
  errors?: FieldFailure[];
}

/** The problem types that the hub owns. */
export const PROBLEM_TYPE = {
  requestInvalid: 'urn:maroid:problem:http:request-invalid',
  bodyInvalid: 'urn:maroid:problem:http:body-invalid',
  accessDenied: 'urn:maroid:problem:http:access-denied',
  networkNotAllowed: 'urn:maroid:problem:hub:network-not-allowed',
  notFound: 'urn:maroid:problem:http:not-found',
  settingsAbsent: 'urn:maroid:problem:hub:settings-absent',
  methodNotAllowed: 'urn:maroid:problem:http:method-not-allowed',
  identityLast: 'urn:maroid:problem:hub:identity-last',
  validationFailed: 'urn:maroid:problem:http:validation-failed',
  settingsInvalid: 'urn:maroid:problem:hub:settings-invalid',
  internal: 'urn:maroid:problem:http:internal'
} as const;

/**
 * Reports whether the value carries the three members that a problem requires.
 * A body that another server wrote reaches this too, so it checks the shape.
 */
export function isProblem(value: unknown): value is Problem {
  if (typeof value !== 'object' || value === null) {
    return false;
  }

  const candidate = value as Partial<Problem>;

  return (
    typeof candidate.type === 'string' &&
    typeof candidate.title === 'string' &&
    typeof candidate.status === 'number'
  );
}
