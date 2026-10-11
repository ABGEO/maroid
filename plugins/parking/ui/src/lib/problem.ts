import { ApiError } from "@maroid/plugin-sdk";

/**
 * The reason that a person reads for a failed call: the first field that the hub
 * refused, the detail of the problem, or the fallback for any other failure.
 */
export function failureMessage(error: unknown, fallback: string): string {
  if (!(error instanceof ApiError) || error.problem === null) {
    return fallback;
  }

  const [field] = error.fields;
  if (field) {
    return `The code ${field.detail}.`;
  }

  return error.problem.detail ?? error.problem.title;
}
