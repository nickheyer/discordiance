<script lang="ts">
	import { onMount } from 'svelte';
	import { rpcClient, silentCallOptions } from '$lib/api/rpc-client';
	import type { ProductStatus } from '$lib/proto/discordiance/v1/types_pb';
	import type { Issue } from '$lib/proto/discordiance/v1/types_pb';
	import Badge from '$lib/components/badge.svelte';
	import { Activity, Package, AlertTriangle, MessageSquare } from '@lucide/svelte';

	let productCount = $state(0);
	let issueCount = $state(0);
	let messageCount = $state(0);
	let statuses = $state<ProductStatus[]>([]);
	let recentIssues = $state<Issue[]>([]);
	let health = $state<'loading' | 'ok' | 'degraded' | 'error'>('loading');

	onMount(async () => {
		try {
			const [products, issues, messages, pipeline, hc] = await Promise.all([
				rpcClient.product.listProducts({}, silentCallOptions),
				rpcClient.issue.listIssues({ pageSize: 5 }, silentCallOptions),
				rpcClient.message.listMessages({ pageSize: 1 }, silentCallOptions),
				rpcClient.pipeline.getPipelineStatus({}, silentCallOptions),
				rpcClient.health.healthCheck({}, silentCallOptions)
			]);
			productCount = products.products.length;
			issueCount = issues.totalCount;
			messageCount = messages.totalCount;
			statuses = pipeline.statuses;
			recentIssues = issues.issues;
			health = hc.status as 'ok' | 'degraded';
		} catch {
			health = 'error';
		}
	});
</script>

<div class="space-y-8">
	<div>
		<h1 class="text-xl font-semibold tracking-tight">Dashboard</h1>
		<p class="text-sm text-muted-foreground mt-0.5">Community feedback pipeline overview</p>
	</div>

	<!-- Stats row -->
	<div class="grid gap-3 grid-cols-2 lg:grid-cols-4">
		{#each [
			{ label: 'Products', value: productCount, icon: Package, color: 'text-primary' },
			{ label: 'Issues', value: issueCount, icon: AlertTriangle, color: 'text-orange-500' },
			{ label: 'Messages', value: messageCount, icon: MessageSquare, color: 'text-blue-500' },
			{ label: 'Engine', value: health === 'loading' ? '...' : health, icon: Activity, color: health === 'ok' ? 'text-success' : 'text-destructive' }
		] as stat}
			<div class="card card-body flex items-center gap-3">
				<stat.icon class="h-5 w-5 {stat.color} shrink-0" />
				<div>
					<p class="text-xs text-muted-foreground">{stat.label}</p>
					<p class="text-lg font-semibold capitalize">{stat.value}</p>
				</div>
			</div>
		{/each}
	</div>

	<div class="grid gap-6 lg:grid-cols-2">
		<!-- Pipelines -->
		<div class="card">
			<div class="card-header flex items-center justify-between">
				<h2 class="text-sm font-semibold">Pipelines</h2>
				<span class="text-xs text-muted-foreground">{statuses.length} running</span>
			</div>
			{#if statuses.length === 0}
				<div class="card-body text-sm text-muted-foreground">No pipelines running.</div>
			{:else}
				<div class="divide-y divide-border/50">
					{#each statuses as s}
						<div class="flex items-center justify-between px-5 py-2.5">
							<div class="flex items-center gap-2.5">
								<span class="h-2 w-2 rounded-full {s.healthy ? 'bg-success' : 'bg-destructive'}"></span>
								<span class="text-sm font-medium">{s.productName}</span>
							</div>
							<Badge value={s.healthy ? 'healthy' : 'unhealthy'} type="status" />
						</div>
					{/each}
				</div>
			{/if}
		</div>

		<!-- Recent Issues -->
		<div class="card">
			<div class="card-header flex items-center justify-between">
				<h2 class="text-sm font-semibold">Recent Issues</h2>
				<a href="/issues" class="text-xs text-primary hover:underline">View all</a>
			</div>
			{#if recentIssues.length === 0}
				<div class="card-body text-sm text-muted-foreground">No issues yet.</div>
			{:else}
				<div class="divide-y divide-border/50">
					{#each recentIssues as issue}
						<a href="/issues/{issue.id}" class="flex items-center justify-between px-5 py-2.5 hover:bg-muted/30 transition-colors">
							<div class="min-w-0 mr-3">
								<p class="text-sm font-medium truncate">{issue.title}</p>
								<p class="text-xs text-muted-foreground">{issue.productName}</p>
							</div>
							<div class="flex items-center gap-1.5 shrink-0">
								<Badge value={issue.severity} type="severity" />
								<Badge value={issue.status} type="status" />
							</div>
						</a>
					{/each}
				</div>
			{/if}
		</div>
	</div>
</div>
