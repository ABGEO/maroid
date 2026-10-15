import type { CareState, MaintenanceType } from '../api';

/** The maintenance types, in the order that every list shows them. */
export const MAINTENANCE_TYPES: { value: MaintenanceType; label: string }[] = [
	{ value: 'watering', label: 'Watering' },
	{ value: 'fertilizing', label: 'Fertilizing' },
	{ value: 'pruning', label: 'Pruning' },
	{ value: 'repotting', label: 'Repotting' },
	{ value: 'pest_treatment', label: 'Pest treatment' },
	{ value: 'cleaning', label: 'Cleaning' },
	{ value: 'misting', label: 'Misting' },
	{ value: 'other', label: 'Other' }
];

export function typeLabel(type: MaintenanceType): string {
	return MAINTENANCE_TYPES.find((entry) => entry.value === type)?.label ?? type;
}

export const STATE_BADGE: Record<CareState, { label: string; tone: string }> = {
	on_schedule: { label: 'On schedule', tone: 'badge-success' },
	due: { label: 'Due', tone: 'badge-warning' },
	overdue: { label: 'Overdue', tone: 'badge-error' }
};
