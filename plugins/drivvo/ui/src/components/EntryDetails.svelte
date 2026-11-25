<script lang="ts" module>
	import type { LogEntry } from '../api';

	/** Reports whether the entry holds a member beyond its moment, odometer value, and amount. */
	export function hasDetails(entry: LogEntry): boolean {
		const names = [
			entry.title,
			entry.income_type,
			entry.station,
			entry.place,
			entry.payment_method,
			entry.reason,
			entry.note
		];
		const values = [
			entry.missed_refuel,
			entry.battery_start_pct,
			entry.battery_end_pct,
			entry.charge_minutes
		];

		return (
			names.some(Boolean) ||
			values.some((value) => value !== undefined) ||
			(entry.lines?.length ?? 0) > 0 ||
			(entry.instalments?.length ?? 0) > 0
		);
	}
</script>

<script lang="ts">
	import type { Units } from '../api';
	import { amount, day, money, volume } from '../lib/format';

	let { entry, units }: { entry: LogEntry; units: Units } = $props();

	const facts = $derived(
		[
			['Title', entry.title],
			['Income type', entry.income_type],
			['Station', entry.station],
			['Place', entry.place],
			['Payment method', entry.payment_method],
			['Reason', entry.reason],
			[
				'Missed refuel',
				entry.missed_refuel === undefined ? undefined : entry.missed_refuel ? 'Yes' : 'No'
			],
			[
				'Battery charge',
				entry.battery_start_pct === undefined && entry.battery_end_pct === undefined
					? undefined
					: `${percent(entry.battery_start_pct)} to ${percent(entry.battery_end_pct)}`
			],
			[
				'Charge time',
				entry.charge_minutes === undefined ? undefined : `${entry.charge_minutes} min`
			]
		].filter((fact): fact is [string, string] => Boolean(fact[1]))
	);
	const fuel = $derived(entry.kind === 'refuel');

	function percent(value: number | undefined): string {
		return value === undefined ? '' : `${value}%`;
	}
</script>

<div class="flex flex-col gap-4">
	{#if facts.length > 0 || entry.note}
		<dl class="grid gap-x-6 gap-y-3 sm:grid-cols-2 lg:grid-cols-3">
			{#each facts as [label, value] (label)}
				<div>
					<dt class="meta">{label}</dt>
					<dd>{value}</dd>
				</div>
			{/each}
			{#if entry.note}
				<div class="sm:col-span-2 lg:col-span-3">
					<dt class="meta">Note</dt>
					<dd class="whitespace-pre-line">{entry.note}</dd>
				</div>
			{/if}
		</dl>
	{/if}

	{#if entry.lines && entry.lines.length > 0}
		<div class="overflow-x-auto">
			<table class="table-sm table">
				<caption class="meta text-left">{fuel ? 'Fuels' : 'Types'}</caption>
				<thead>
					<tr>
						<th>{fuel ? 'Fuel' : 'Type'}</th>
						{#if fuel}
							<th class="text-right">Volume</th>
							<th class="text-right">Unit price</th>
							<th>Full tank</th>
						{/if}
						<th class="text-right">Amount</th>
					</tr>
				</thead>
				<tbody>
					{#each entry.lines as line (line.position)}
						<tr>
							<td>
								{line.item ?? ''}
								{#if line.free}
									<span class="badge badge-soft badge-sm">Free</span>
								{/if}
							</td>
							{#if fuel}
								<td class="text-right whitespace-nowrap tabular-nums">
									{line.volume === undefined ? '' : volume(line.volume, units)}
								</td>
								<td class="text-right whitespace-nowrap tabular-nums">
									{line.unit_price === undefined ? '' : amount(line.unit_price)}
								</td>
								<td>
									{#if line.full_tank}
										<span class="badge badge-soft badge-sm">Full</span>
									{/if}
								</td>
							{/if}
							<td class="text-right whitespace-nowrap tabular-nums">{money(line.amount)}</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}

	{#if entry.instalments && entry.instalments.length > 0}
		<div class="overflow-x-auto">
			<table class="table-sm table">
				<caption class="meta text-left">Instalments</caption>
				<thead>
					<tr>
						<th>Number</th>
						<th>Due</th>
						<th class="text-right">Amount</th>
					</tr>
				</thead>
				<tbody>
					{#each entry.instalments as instalment (instalment.number)}
						<tr>
							<td class="tabular-nums">{instalment.number}</td>
							<td class="whitespace-nowrap">{day(instalment.due_on)}</td>
							<td class="text-right whitespace-nowrap tabular-nums">{money(instalment.amount)}</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}
</div>
