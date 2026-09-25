/** The header that carries the flow identifier of a request. See `ERR-006`. */
export const FLOW_ID_HEADER = 'X-Flow-ID';

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
  /** The problem type, a relative reference of the form /problems/<owner>/<slug>. */
  type: string;
  title: string;
  status: number;
  detail?: string;
  instance?: string;
  errors?: FieldFailure[];
}

/** The problem types that the hub owns. */
export const PROBLEM_TYPE = {
  requestInvalid: '/problems/http/request-invalid',
  bodyInvalid: '/problems/http/body-invalid',
  accessDenied: '/problems/http/access-denied',
  networkNotAllowed: '/problems/hub/network-not-allowed',
  notFound: '/problems/http/not-found',
  settingsAbsent: '/problems/hub/settings-absent',
  methodNotAllowed: '/problems/http/method-not-allowed',
  identityLast: '/problems/hub/identity-last',
  validationFailed: '/problems/http/validation-failed',
  settingsInvalid: '/problems/hub/settings-invalid',
  internal: '/problems/http/internal'
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
