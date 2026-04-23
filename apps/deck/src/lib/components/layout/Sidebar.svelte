<script lang="ts">
	import { resolve } from '$app/paths';
	import { page } from '$app/state';

	type NavItem = { label: string; href: string };
	type PluginGroupDef = {
		name: string;
		items: NavItem[];
	};

	function isActive(href: string) {
		return page.url.pathname === href;
	}

	function groupIsOpen(items: NavItem[]) {
		return items.some((i) => isActive(i.href));
	}

	function letterFromName(name: string) {
		return name.charAt(0).toUpperCase();
	}

	function nameToHue(name: string) {
		const code = letterFromName(name).charCodeAt(0);
		const hue = (code - 65) * 137.508;

		return hue % 360;
	}

	const pluginGroups: PluginGroupDef[] = [
		{
			name: 'Utilities',
			items: [
				{ label: 'Electricity', href: '/placeholder/utilities/electricity' },
				{ label: 'Gas', href: '/placeholder/utilities/gas' },
				{ label: 'Water', href: '/placeholder/utilities/water' },
				{ label: 'Internet', href: '/placeholder/utilities/internet' }
			]
		},
		{
			name: 'Parking',
			items: [
				{ label: 'Active session', href: '/placeholder/parking/active' },
				{ label: 'Vehicles', href: '/placeholder/parking/vehicles' },
				{ label: 'History', href: '/placeholder/parking/history' },
				{ label: 'Balance', href: '/placeholder/parking/balance' }
			]
		},
		{
			name: 'Jasmine',
			items: [
				{ label: 'Plants', href: '/placeholder/jasmine/plants' },
				{ label: 'Environments', href: '/placeholder/jasmine/environments' },
				{ label: 'Sensors', href: '/placeholder/jasmine/sensors' },
				{ label: 'MQTT log', href: '/placeholder/jasmine/mqtt' }
			]
		}
	];
</script>

<aside
	class="border-base-300 bg-base-200 lg:bg-base-200/40 is-drawer-close:w-14 is-drawer-open:w-68 flex min-h-full shrink-0 flex-col border-r pt-14 pb-10 transition-[width] duration-200 ease-out"
>
	<div class="is-drawer-close:overflow-visible flex-1 overflow-y-auto">
		<ul class="menu w-full">
			<li>
				<a href={resolve('/')} class:menu-active={isActive(resolve('/'))}>
					<svg
						xmlns="http://www.w3.org/2000/svg"
						width="16"
						height="16"
						viewBox="0 0 24 24"
						fill="none"
						stroke="currentColor"
						stroke-width="1.75"
						class="shrink-0"
					>
						<rect x="3" y="3" width="7" height="9" rx="1" />
						<rect x="14" y="3" width="7" height="5" rx="1" />
						<rect x="14" y="12" width="7" height="9" rx="1" />
						<rect x="3" y="16" width="7" height="5" rx="1" />
					</svg>
					<span class="is-drawer-close:hidden">Overview</span>
				</a>
			</li>
			<li>
				<a
					href={resolve('/placeholder/services')}
					class:menu-active={isActive(resolve('/placeholder/services'))}
				>
					<svg
						xmlns="http://www.w3.org/2000/svg"
						width="16"
						height="16"
						viewBox="0 0 24 24"
						fill="none"
						stroke="currentColor"
						stroke-width="1.75"
						class="shrink-0"
					>
						<path d="M4 4h6v6H4zM14 4h6v6h-6zM4 14h6v6H4zM14 14h6v6h-6z" />
					</svg>
					<span class="is-drawer-close:hidden">Services</span>
				</a>
			</li>
			<li>
				<a
					href={resolve('/placeholder/alerts')}
					class:menu-active={isActive(resolve('/placeholder/alerts'))}
				>
					<svg
						xmlns="http://www.w3.org/2000/svg"
						width="16"
						height="16"
						viewBox="0 0 24 24"
						fill="none"
						stroke="currentColor"
						stroke-width="1.75"
						class="shrink-0"
					>
						<path d="M6 2h12l-1 7H7z" />
						<path d="M12 9v8" />
						<circle cx="12" cy="20" r="2" />
					</svg>
					<span class="is-drawer-close:hidden">Alerts</span>
				</a>
			</li>
			<li
				class="menu-title is-drawer-close:hidden text-base-content/45 font-mono text-[10px] tracking-widest uppercase"
			>
				Plugins
			</li>
			{#each pluginGroups as group (group.name)}
				<li>
					<details open={groupIsOpen(group.items)}>
						<summary>
							<span
								class="grid h-4 w-4 shrink-0 place-items-center rounded font-mono text-[10px] font-semibold"
								style="background:oklch(92% 0.04 {nameToHue(
									group.name
								)});color:oklch(40% 0.1 {nameToHue(group.name)})"
							>
								{letterFromName(group.name)}
							</span>
							<span class="is-drawer-close:hidden">{group.name}</span>
						</summary>
						<ul>
							{#each group.items as item (item.href)}
								<li>
									<a href={resolve(item.href)} class:menu-active={isActive(resolve(item.href))}>
										{item.label}
									</a>
								</li>
							{/each}
						</ul>
					</details>
				</li>
			{/each}
			<div
				class="is-drawer-close:hidden border-base-300 mx-2 mt-2 rounded-md border border-dashed px-2 py-2"
			>
				<div class="text-base-content/50 font-mono text-[10px] leading-relaxed">
					Your installed plugins will show up here.
				</div>
			</div>
		</ul>
	</div>
</aside>
