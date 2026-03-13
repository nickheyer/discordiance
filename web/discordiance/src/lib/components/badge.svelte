<script lang="ts">
	let { value, type = 'default' }: { value: string; type?: 'severity' | 'status' | 'default' } = $props();

	function cls(val: string, t: string): string {
		const base = 'inline-flex text-[11px] px-1.5 py-0.5 rounded font-medium leading-none';
		if (t === 'severity') {
			const map: Record<string, string> = {
				critical: 'bg-red-500/15 text-red-700 dark:text-red-400',
				high: 'bg-orange-500/15 text-orange-700 dark:text-orange-400',
				medium: 'bg-yellow-500/15 text-yellow-700 dark:text-yellow-400',
				low: 'bg-blue-500/15 text-blue-700 dark:text-blue-400'
			};
			return `${base} ${map[val] || 'bg-muted text-muted-foreground'}`;
		}
		if (t === 'status') {
			const map: Record<string, string> = {
				enabled: 'bg-success/15 text-green-700 dark:text-green-400',
				healthy: 'bg-success/15 text-green-700 dark:text-green-400',
				open: 'bg-blue-500/15 text-blue-700 dark:text-blue-400',
				filed: 'bg-blue-500/15 text-blue-700 dark:text-blue-400',
				acknowledged: 'bg-violet-500/15 text-violet-700 dark:text-violet-400',
				resolved: 'bg-muted text-muted-foreground',
				closed: 'bg-muted text-muted-foreground',
				disabled: 'bg-muted text-muted-foreground',
				unhealthy: 'bg-red-500/15 text-red-700 dark:text-red-400'
			};
			return `${base} ${map[val] || 'bg-muted text-muted-foreground'}`;
		}
		return `${base} bg-primary/10 text-primary`;
	}
</script>

<span class={cls(value, type)}>{value}</span>
