<script lang="ts">
	import { page } from '$app/state';
</script>

<div>
	<h1 class="font-display text-[42px] leading-[1.05] tracking-tight">
		Placeholder · <span class="text-primary">{page.params.path || 'page'}</span>
	</h1>
	<p class="text-base-content/70 mt-3 max-w-xl text-[15px] leading-relaxed">
		This page is a placeholder for the selected nav item. Real content will land here once the
		corresponding plugin surfaces a route.
	</p>
</div>
