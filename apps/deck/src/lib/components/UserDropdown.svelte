<div class="dropdown dropdown-end">
	<div
		tabindex="0"
		role="button"
		class="hover:bg-base-200 flex cursor-pointer items-center gap-2 rounded-full py-1 pr-2 pl-1"
	>
		<div class="avatar">
			<div class="ring-base-300 w-8 rounded-full ring-1">
				<img alt="Esme Valverde" src="https://i.pravatar.cc/80?img=47" />
			</div>
		</div>
		<div class="hidden pr-1 text-left leading-tight sm:block">
			<div class="text-[12px] font-semibold">Esme Valverde</div>
			<div class="text-base-content/50 font-mono text-[10px]">self-hosted · home.hub</div>
		</div>
		<svg
			xmlns="http://www.w3.org/2000/svg"
			width="12"
			height="12"
			viewBox="0 0 24 24"
			fill="none"
			stroke="currentColor"
			stroke-width="2"
			opacity="0.5"
		>
			<path d="m6 9 6 6 6-6" />
		</svg>
	</div>

	<div
		tabindex="-1"
		role="menu"
		class="dropdown-content bg-base-100 border-base-300 rounded-box z-50 mt-2 w-60 border p-2 shadow-lg"
	>
		<div class="bg-base-200/50 flex items-center gap-3 rounded-md p-3">
			<div class="avatar">
				<div class="ring-base-300 w-10 rounded-full ring-1">
					<img src="https://i.pravatar.cc/80?img=47" alt="Esme Valverde" />
				</div>
			</div>
			<div class="min-w-0">
				<div class="truncate text-sm font-semibold">Esme Valverde</div>
				<div class="text-base-content/55 truncate font-mono text-[11px]">esme@valverde.studio</div>
			</div>
		</div>

		<ul class="menu menu-sm mt-2 w-full">
			<li>
				<a>
					<svg
						xmlns="http://www.w3.org/2000/svg"
						width="14"
						height="14"
						viewBox="0 0 24 24"
						fill="none"
						stroke="currentColor"
						stroke-width="2"
					>
						<circle cx="12" cy="8" r="4" />
						<path d="M6 21v-2a4 4 0 0 1 4-4h4a4 4 0 0 1 4 4v2" />
					</svg>
					Profile
				</a>
			</li>
			<li>
				<a>
					<svg
						xmlns="http://www.w3.org/2000/svg"
						width="14"
						height="14"
						viewBox="0 0 24 24"
						fill="none"
						stroke="currentColor"
						stroke-width="2"
					>
						<path d="M4 4h16v4H4zM4 12h16v8H4z" />
						<path d="M8 6h.01M8 14h.01" />
					</svg>
					Hub settings
				</a>
			</li>
		</ul>

		<div class="border-base-300 mt-1 border-t px-3 py-2">
			<div class="flex items-center justify-between">
				<span class="text-base-content/50 font-mono text-[10px] tracking-wider uppercase">
					Theme
				</span>
				<div class="join">
					<input
						type="radio"
						name="theme-buttons"
						class="btn btn-xs theme-controller join-item"
						aria-label="Auto"
						value="default"
					/>
					<input
						type="radio"
						name="theme-buttons"
						checked
						class="btn btn-xs theme-controller join-item"
						aria-label="Paper"
						value="maroid"
					/>
					<input
						type="radio"
						name="theme-buttons"
						class="btn btn-xs theme-controller join-item"
						aria-label="Dusk"
						value="maroid-dusk"
					/>
				</div>
			</div>
		</div>

		<ul class="menu menu-sm w-full">
			<li>
				<a class="text-error">
					<svg
						xmlns="http://www.w3.org/2000/svg"
						width="14"
						height="14"
						viewBox="0 0 24 24"
						fill="none"
						stroke="currentColor"
						stroke-width="2"
					>
						<path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4" />
						<path d="m16 17 5-5-5-5" />
						<path d="M21 12H9" />
					</svg>
					Log out
				</a>
			</li>
		</ul>
	</div>
</div>
