import type { PluginHost } from '@maroid/plugin-sdk';

import { createPulseApi } from '../api';

/**
 * The zone that every time helper of a page takes. A profile with no zone gives
 * UTC, which the schedule uses then, not the zone of the browser.
 */
export function scheduleZone(host: PluginHost): string {
	return host.timeZone ?? 'UTC';
}

/** Whether pulse is on for the person. It stays undefined until the read answers. */
export function createPulseState(host: () => PluginHost) {
	let enabled = $state<boolean | undefined>(undefined);

	return {
		get enabled() {
			return enabled;
		},
		async load(): Promise<void> {
			const read = await createPulseApi(host()).settings.get();
			enabled = read?.value.enabled ?? false;
		}
	};
}
