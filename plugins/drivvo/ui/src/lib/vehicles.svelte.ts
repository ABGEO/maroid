import type { PluginHost } from '@maroid/plugin-sdk';

import { createCarLogApi, type Vehicle } from '../api';

export interface Fleet {
	readonly vehicles: Vehicle[] | null;
	/** The vehicle that the address names, or undefined when the copy does not hold it. */
	readonly vehicle: Vehicle | undefined;
	readonly status: 'loading' | 'ready' | 'error';
	load(): Promise<void>;
}

/** The vehicles of the copy, and the one of them that the address names. */
export function createFleet(host: () => PluginHost, vehicleId: () => string): Fleet {
	let vehicles = $state<Vehicle[] | null>(null);
	let status = $state<'loading' | 'ready' | 'error'>('loading');
	const vehicle = $derived(vehicles?.find((candidate) => candidate.id === vehicleId()));

	async function load(): Promise<void> {
		status = 'loading';

		try {
			const read = await createCarLogApi(host()).vehicles();
			if (read !== null) {
				vehicles = read.items;
				status = 'ready';
			}
		} catch (error) {
			console.error('Failed to load the vehicles', error);
			status = 'error';
		}
	}

	return {
		get vehicles() {
			return vehicles;
		},
		get vehicle() {
			return vehicle;
		},
		get status() {
			return status;
		},
		load
	};
}
