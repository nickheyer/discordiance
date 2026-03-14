<script lang="ts">
	import { rpcClient } from '$lib/api/rpc-client';
	import { formatRelative, severityColor, statusColor } from '$lib/utils';
	import type { Insight, Product } from '$lib/proto/discordiance/v1/types_pb';
	import { Lightbulb, ChevronLeft, ChevronRight, FileText } from '@lucide/svelte';

	let insights = $state<Insight[]>([]);
	let products = $state<Product[]>([]);
	let totalCount = $state(0);
	let loading = $state(true);

	let filterProductId = $state('');
	let filterStatus = $state('');
	let currentPage = $state(1);
	let pageSize = 25;

	async function load() {
		loading = true;
		try {
			const [insightRes, productRes] = await Promise.all([
				rpcClient.insight.listInsights({
					productId: filterProductId ? BigInt(filterProductId) : BigInt(0),
					status: filterStatus,
					pageSize,
					page: currentPage
				}),
				rpcClient.product.listProducts({})
			]);
			insights = insightRes.insights;
			totalCount = insightRes.totalCount;
			products = productRes.products;
		} catch {
			// handled
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		filterProductId;
		filterStatus;
		currentPage;
		load();
	});

	function resetFilters() {
		filterProductId = '';
		filterStatus = '';
		currentPage = 1;
	}

	let totalPages = $derived(Math.max(1, Math.ceil(totalCount / pageSize)));
</script>

<div class="page-header">
	<div>
		<h1 class="page-title">Insights</h1>
		<p class="page-subtitle">AI-detected insights from community messages</p>
	</div>
</div>

<div class="filter-bar">
	<select class="form-select" bind:value={filterProductId} onchange={() => (currentPage = 1)}>
		<option value="">All Products</option>
		{#each products as p}
			<option value={String(p.id)}>{p.name}</option>
		{/each}
	</select>
	<select class="form-select" bind:value={filterStatus} onchange={() => (currentPage = 1)}>
		<option value="">All Statuses</option>
		<option value="new">New</option>
		<option value="confirmed">Confirmed</option>
		<option value="resolved">Resolved</option>
		<option value="dismissed">Dismissed</option>
	</select>
	{#if filterProductId || filterStatus}
		<button class="btn-ghost btn-sm text-muted-foreground" onclick={resetFilters}>Clear filters</button>
	{/if}
</div>

{#if loading}
	<div class="card">
		<div class="p-8">
			{#each Array(5) as _}
				<div class="skeleton h-10 w-full mb-2"></div>
			{/each}
		</div>
	</div>
{:else if insights.length === 0}
	<div class="card">
		<div class="empty-state">
			<Lightbulb class="w-10 h-10 empty-state-icon" />
			<p class="empty-state-title">No insights found</p>
			<p class="empty-state-text">
				{#if filterProductId || filterStatus}
					Try adjusting your filters
				{:else}
					Insights will appear as pipelines analyze community messages
				{/if}
			</p>
			{#if filterProductId || filterStatus}
				<button class="btn-secondary btn-sm" onclick={resetFilters}>Clear filters</button>
			{/if}
		</div>
	</div>
{:else}
	<div class="card overflow-hidden">
		<div class="overflow-x-auto">
			<table class="data-table">
				<thead>
					<tr>
						<th>Title</th>
						<th>Product</th>
						<th>Severity</th>
						<th>Category</th>
						<th>Status</th>
						<th>Reports</th>
						<th>Detected</th>
					</tr>
				</thead>
				<tbody>
					{#each insights as insight}
						<tr class="row-link" onclick={() => { window.location.href = `/insights/${insight.id}` }}>
							<td>
								<span class="font-medium text-foreground">{insight.title}</span>
							</td>
							<td class="text-muted-foreground">{insight.productName || '—'}</td>
							<td><span class="badge {severityColor(insight.severity)}">{insight.severity}</span></td>
							<td class="text-muted-foreground">{insight.category}</td>
							<td><span class="badge {statusColor(insight.status)}">{insight.status}</span></td>
							<td>
								{#if insight.reports.length > 0}
									<span class="flex items-center gap-1 text-xs text-muted-foreground">
										<FileText class="w-3 h-3" />
										{insight.reports.length}
									</span>
								{:else}
									<span class="text-xs text-muted-foreground">—</span>
								{/if}
							</td>
							<td class="text-xs text-muted-foreground whitespace-nowrap">{formatRelative(insight.createdAt)}</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
		{#if totalPages > 1}
			<div class="pagination px-4">
				<span class="pagination-info">
					Page {currentPage} of {totalPages} ({totalCount} total)
				</span>
				<div class="pagination-buttons">
					<button
						class="pagination-btn"
						disabled={currentPage <= 1}
						onclick={() => (currentPage = Math.max(1, currentPage - 1))}
					>
						<ChevronLeft class="w-4 h-4" />
					</button>
					<button
						class="pagination-btn"
						disabled={currentPage >= totalPages}
						onclick={() => (currentPage = Math.min(totalPages, currentPage + 1))}
					>
						<ChevronRight class="w-4 h-4" />
					</button>
				</div>
			</div>
		{/if}
	</div>
{/if}
