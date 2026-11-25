<script lang="ts">
	import type { Page, PluginHost, RouteParams } from '@maroid/plugin-sdk';

	import { createCarLogApi, type Reminder } from '../api';
	import VehicleShell from '../components/VehicleShell.svelte';
	import { day, daysLeft, distance, distanceLeft, kindLabel } from '../lib/format';
	import { createReader } from '../lib/reader.svelte';
	import { createFleet } from '../lib/vehicles.svelte';

	let { host, params }: { host: PluginHost; params: RouteParams } = $props();

	const api = $derived(createCarLogApi(host));
	const fleet = createFleet(
		() => host,
		() => params.vehicle
	);
	const reminders = createReader<Page<Reminder>>();

	function load(): void {
		const vehicle = fleet.vehicle;
		if (vehicle !== undefined) {
			void reminders.load(() => api.reminders(vehicle.id));
		}
	}

	$effect(load);
</script>

<VehicleShell
	{host}
	{fleet}
	page="reminders"
	title="Reminders"
	lead="The reminders of the vehicle, the due ones first."
>
	{#snippet children(vehicle)}
		{@const units = vehicle.units}
		{#if reminders.status === 'error'}
			<div role="alert" class="alert alert-error alert-soft max-w-xl">
				<span>Failed to load the reminders.</span>
				<button type="button" class="btn btn-sm" onclick={load}>Retry</button>
			</div>
		{:else if reminders.value === null}
			<div class="flex flex-col gap-2" aria-hidden="true">
				{#each [0, 1, 2] as i (i)}
					<span class="skeleton h-10 w-full"></span>
				{/each}
			</div>
		{:else if reminders.value.items.length === 0}
			<div class="empty-state">No reminder for this vehicle.</div>
		{:else}
			<div class="panel overflow-x-auto">
				<table class="table-zebra table">
					<thead>
						<tr>
							<th>Type</th>
							<th>Due day</th>
							<th class="text-right">Due odometer</th>
							<th>Left</th>
						</tr>
					</thead>
					<tbody>
						{#each reminders.value.items as reminder (reminder.id)}
							<tr>
								<td>
									<div class="flex flex-wrap items-center gap-2">
										<span>{reminder.type}</span>
										{#if reminder.due}
											<span class="badge badge-warning badge-soft badge-sm">Due</span>
										{/if}
										{#if reminder.repeats}
											<span class="badge badge-ghost badge-sm">Repeats</span>
										{/if}
									</div>
									<div class="meta">
										{kindLabel(reminder.kind)}{reminder.note ? `. ${reminder.note}` : ''}
									</div>
								</td>
								<td class="whitespace-nowrap">
									{reminder.due_on === undefined ? '' : day(reminder.due_on)}
								</td>
								<td class="text-right whitespace-nowrap tabular-nums">
									{reminder.due_odometer === undefined
										? ''
										: distance(reminder.due_odometer, units)}
								</td>
								<td class="whitespace-nowrap tabular-nums">
									{[
										reminder.days_left === undefined ? '' : daysLeft(reminder.days_left),
										reminder.distance_left === undefined
											? ''
											: distanceLeft(reminder.distance_left, units)
									]
										.filter(Boolean)
										.join(', ')}
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	{/snippet}
</VehicleShell>
