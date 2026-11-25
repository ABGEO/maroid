import { ApiError } from '@maroid/plugin-sdk';

const PREFIX = '/problems/dev.maroid.drivvo/';

export const VEHICLE_NOT_FOUND = `${PREFIX}vehicle-not-found`;
export const LOG_ENTRY_NOT_FOUND = `${PREFIX}log-entry-not-found`;
export const SERVICE_UNAVAILABLE = `${PREFIX}service-unavailable`;

export function isProblem(error: unknown, type: string): boolean {
	return error instanceof ApiError && error.is(type);
}
