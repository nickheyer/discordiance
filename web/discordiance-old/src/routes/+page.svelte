<script lang="ts">
	import { rpcClient, silentCallOptions } from '$lib/api/rpc-client';
	import { severityColor, statusColor } from '$lib/utils';
	import type { PipelineStatus, Insight, Product } from '$lib/proto/discordiance/v1/types_pb';
	import {
		GitBranch,
		Package,
		Lightbulb,
		MessageSquare,
		ArrowRight
	} from '@lucide/svelte';

	let pipelineStatuses = $state<PipelineStatus[]>([]);
	let products = $state<Product[]>([]);
	let recentInsights = $state<Insight[]>([]);
	let messageStats = $state({ total: 0, processed: 0, unprocessed: 0 });
	let loading = $state(true);

	async function loadDashboard() {
		loading = true;
		try {
			const [statusRes, productRes, insightRes, msgStatsRes] = await Promise.all([
				rpcClient.pipeline.getPipelineStatus({}, silentCallOptions),
				rpcClient.product.listProducts({}, silentCallOptions),
				rpcClient.insight.listInsights({ pageSize: 5, page: 1 }, silentCallOptions),
				rpcClient.message.getMessageStats({}, silentCallOptions)
			]);
			pipelineStatuses = statusRes.statuses;
			products = productRes.products;
			recentInsights = insightRes.insights;
			messageStats = {
				total: msgStatsRes.total,
				processed: msgStatsRes.processed,
				unprocessed: msgStatsRes.unprocessed
			};
		} catch {
			// Errors shown by interceptor
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		loadDashboard();
	});

	let activePipelines = $derived(pipelineStatuses.filter((s) => s.running).length);
	let healthyPipelines = $derived(pipelineStatuses.filter((s) => s.healthy).length);
</script>

<div class="page-header">
	<div>
		<h1 class="page-title">Dashboard</h1>
		<p class="page-subtitle">System overview and operational status</p>
	</div>
</div>

{#if loading}
	<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
		{#each Array(4) as _}
			<div class="kpi-card">
				<div class="skeleton h-3 w-20 mb-2"></div>
				<div class="skeleton h-7 w-12"></div>
			</div>
		{/each}
	</div>
{:else}
	<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
		<div class="kpi-card">
			<div class="flex items-center justify-between">
				<span class="kpi-label">Active Pipelines</span>
				<GitBranch class="w-4 h-4 text-muted-foreground" />
			</div>
			<div class="kpi-value">{activePipelines}</div>
			<div class="kpi-sub">{pipelineStatuses.length} total, {healthyPipelines} healthy</div>
		</div>

		<div class="kpi-card">
			<div class="flex items-center justify-between">
				<span class="kpi-label">Products</span>
				<Package class="w-4 h-4 text-muted-foreground" />
			</div>
			<div class="kpi-value">{products.length}</div>
			<div class="kpi-sub">{products.filter((p) => p.enabled).length} enabled</div>
		</div>

		<div class="kpi-card">
			<div class="flex items-center justify-between">
				<span class="kpi-label">Open Insights</span>
				<Lightbulb class="w-4 h-4 text-muted-foreground" />
			</div>
			<div class="kpi-value">{recentInsights.length}</div>
			<div class="kpi-sub">Latest from all products</div>
		</div>

		<div class="kpi-card">
			<div class="flex items-center justify-between">
				<span class="kpi-label">Messages</span>
				<MessageSquare class="w-4 h-4 text-muted-foreground" />
			</div>
			<div class="kpi-value">{messageStats.total.toLocaleString()}</div>
			<div class="kpi-sub">{messageStats.processed} processed, {messageStats.unprocessed} pending</div>
		</div>
	</div>

	<div class="grid grid-cols-1 lg:grid-cols-2 gap-5">
		<div class="card">
			<div class="card-header">
				<h2 class="card-title">Pipeline Status</h2>
				<a href="/pipelines" class="btn-ghost btn-sm text-muted-foreground">
					View all
					<ArrowRight class="w-3 h-3" />
				</a>
			</div>
			{#if pipelineStatuses.length === 0}
				<div class="empty-state py-10">
					<GitBranch class="w-8 h-8 empty-state-icon" />
					<p class="empty-state-title">No pipelines configured</p>
					<p class="empty-state-text">Create a pipeline to start processing community messages</p>
					<a href="/pipelines" class="btn-primary btn-sm">Create Pipeline</a>
				</div>
			{:else}
				<div class="overflow-x-auto">
					<table class="data-table">
						<thead>
							<tr>
								<th>Pipeline</th>
								<th>Product</th>
								<th>Status</th>
								<th>Health</th>
							</tr>
						</thead>
						<tbody>
							{#each pipelineStatuses as status}
								<tr class="row-link" onclick={() => { window.location.href = `/pipelines/${status.pipelineId}` }}>
									<td class="font-medium">{status.pipelineName}</td>
									<td class="text-muted-foreground">{status.productName}</td>
									<td>
										{#if status.running}
											<span class="badge badge-success">
												<span class="badge-dot bg-success"></span>
												Running
											</span>
										{:else}
											<span class="badge badge-muted">
												<span class="badge-dot bg-muted-foreground"></span>
												Stopped
											</span>
										{/if}
									</td>
									<td>
										{#if status.running && status.healthy}
											<span class="badge badge-success">Healthy</span>
										{:else if status.running && !status.healthy}
											<span class="badge badge-destructive">Unhealthy</span>
										{:else}
											<span class="text-xs text-muted-foreground">—</span>
										{/if}
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			{/if}
		</div>

		<div class="card">
			<div class="card-header">
				<h2 class="card-title">Recent Insights</h2>
				<a href="/insights" class="btn-ghost btn-sm text-muted-foreground">
					View all
					<ArrowRight class="w-3 h-3" />
				</a>
			</div>
			{#if recentInsights.length === 0}
				<div class="empty-state py-10">
					<Lightbulb class="w-8 h-8 empty-state-icon" />
					<p class="empty-state-title">No insights yet</p>
					<p class="empty-state-text">Insights appear as pipelines process community messages</p>
				</div>
			{:else}
				<div class="divide-y divide-border/50">
					{#each recentInsights as insight}
						<a href="/insights/{insight.id}" class="flex items-start gap-3 px-5 py-3 hover:bg-muted/30 transition-colors">
							<div class="flex-1 min-w-0">
								<p class="text-sm font-medium text-foreground truncate">{insight.title}</p>
								<div class="flex items-center gap-2 mt-0.5 text-xs text-muted-foreground">
									<span class="badge {severityColor(insight.severity)}">{insight.severity}</span>
									<span>{insight.category}</span>
									{#if insight.productName}
										<span>· {insight.productName}</span>
									{/if}
								</div>
							</div>
							<span class="badge {statusColor(insight.status)} shrink-0 mt-0.5">{insight.status}</span>
						</a>
					{/each}
				</div>
			{/if}
		</div>
	</div>
{/if}
