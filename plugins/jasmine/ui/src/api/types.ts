export interface EnvironmentReference {
	id: string;
	display_id: string;
	name: string;
}

export interface Environment extends EnvironmentReference {
	active_plant_count: number;
	created_at: string;
	updated_at: string;
}

export interface EnvironmentPayload {
	name: string;
}

export type PlantStatus = 'active' | 'inactive';

export interface PlantReference {
	id: string;
	display_id: string;
	common_name: string;
	environment: EnvironmentReference;
}

export interface Plant extends PlantReference {
	scientific_name?: string;
	acquired_on?: string;
	care_notes?: string;
	status: PlantStatus;
	profile_photo?: ProfilePhoto;
	created_at: string;
	updated_at: string;
}

/** A signed address of one variant of a photo, and the time it stops working. */
export interface ReadAddress {
	url: string;
	expires_at: string;
}

/** The profile picture of a plant. The thumbnail is absent while the file is not ready. */
export interface ProfilePhoto {
	id: string;
	thumbnail?: ReadAddress;
}

/** One photo of a moment. The addresses are absent while the file is not ready. */
export interface Photo {
	id: string;
	width: number;
	height: number;
	thumbnail?: ReadAddress;
	display?: ReadAddress;
}

/** One day in the life of a plant, with a note and one or more photos. */
export interface Moment {
	id: string;
	plant_id: string;
	taken_on: string;
	note?: string;
	photos: Photo[];
	created_at: string;
	updated_at: string;
}

export interface MomentInput {
	taken_on: string;
	note?: string;
	file_ids: string[];
}

export interface MomentUpdate {
	taken_on: string;
	note?: string;
}

export interface PlantPayload {
	common_name: string;
	scientific_name?: string | null;
	environment_id: string;
	acquired_on?: string | null;
	care_notes?: string | null;
}

export interface PlantUpdatePayload extends PlantPayload {
	status: PlantStatus;
}

export interface PlantFilter {
	environment_id?: string;
	status?: PlantStatus;
}

export type MaintenanceType =
	| 'watering'
	| 'fertilizing'
	| 'pruning'
	| 'repotting'
	| 'pest_treatment'
	| 'cleaning'
	| 'misting'
	| 'other';

export interface MaintenanceEntry {
	id: string;
	plant: PlantReference;
	type: MaintenanceType;
	performed_at: string;
	note?: string;
	created_at: string;
	updated_at: string;
}

export interface MaintenanceEntryBatchPayload {
	plant_ids: string[];
	type: MaintenanceType;
	performed_at?: string;
	note?: string | null;
}

export interface MaintenanceEntryPayload {
	type: MaintenanceType;
	performed_at: string;
	note?: string | null;
}

export interface MaintenanceEntryFilter {
	plant_id?: string;
	type?: MaintenanceType;
	sort?: 'performed_at' | '-performed_at';
}

export type CareState = 'on_schedule' | 'due' | 'overdue';

export interface CareInterval {
	id: string;
	plant: PlantReference;
	type: MaintenanceType;
	days: number;
	set_at: string;
	last_performed_at?: string;
	counted_from: string;
	days_since: number;
	due_in: number;
	state?: CareState;
	updated_at: string;
}

export interface CareIntervalPayload {
	days: number;
}

export interface CareIntervalFilter {
	state?: CareState[];
	environment_id?: string;
	due_within?: number;
}
