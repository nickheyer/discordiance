<script lang="ts">
	import { rpcClient } from '$lib/api/rpc-client';
	import { toast } from 'svelte-sonner';
	import { formatRelative } from '$lib/utils';
	import type { Message, Product } from '$lib/proto/discordiance/v1/types_pb';
	import { MessageSquare, ChevronLeft, ChevronRight, Trash2, CheckCircle, Circle } from '@lucide/svelte';

	let messages = $state<Message[]>([]);
	let products = $state<Product[]>([]);
	let totalCount = $state(0);
	let stats = $state({ total: 0, processed: 0, unprocessed: 0 });
	let loading = $state(true);

	let filterProductId = $state('');
	let filterProcessed = $state('all');
	let currentPage = $state(1);
	let pageSize = 50;
	let confirmDeleteFilter = $state(false);

	async function load() {
		loading = true;
		try {
			const [msgRes, prodRes, statsRes] = await Promise.all([
				rpcClient.message.listMessages({
					productId: filterProductId ? BigInt(filterProductId) : BigInt(0),
					processedFilter: filterProcessed,
					pageSize,
					page: currentPage
				}),
				rpcClient.product.listProducts({}),
				rpcClient.message.getMessageStats({
					productId: filterProductId ? BigInt(filterProductId) : BigInt(0)
				})
			]);
			messages = msgRes.messages;
			totalCount = msgRes.totalCount;
			products = prodRes.products;
			stats = { total: statsRes.total, processed: statsRes.processed, unprocessed: statsRes.unprocessed };
		} catch {
			// handled
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		filterProductId;
		filterProcessed;
		currentPage;
		load();
	});

	async function deleteFiltered() {
		try {
			const res = await rpcClient.message.deleteMessages({
				productId: filterProductId ? BigInt(filterProductId) : BigInt(0),
				processedFilter: filterProcessed
			});
			toast.success(`Deleted ${res.deletedCount} messages`);
			confirmDeleteFilter = false;
			load();
		} catch {
			// handled
		}
	}

	let totalPages = $derived(Math.max(1, Math.ceil(totalCount / pageSize)));
</script>

<div class="page-header">
	<div>
		<h1 class="page-title">Messages</h1>
		<p class="page-subtitle">Incoming platform messages from all sources</p>
	</div>
</div>

<div class="grid grid-cols-1 sm:grid-cols-3 gap-3 mb-5">
	<div class="kpi-card py-3 px-4">
		<span class="kpi-label text-[10px]">Total</span>
		<span class="text-lg font-bold">{stats.total.toLocaleString()}</span>
	</div>
	<div class="kpi-card py-3 px-4">
		<span class="kpi-label text-[10px]">Processed</span>
		<span class="text-lg font-bold text-success">{stats.processed.toLocaleString()}</span>
	</div>
	<div class="kpi-card py-3 px-4">
		<span class="kpi-label text-[10px]">Unprocessed</span>
		<span class="text-lg font-bold text-warning">{stats.unprocessed.toLocaleString()}</span>
	</div>
</div>

<div class="filter-bar">
	<select class="form-select" bind:value={filterProductId} onchange={() => (currentPage = 1)}>
		<option value="">All Products</option>
		{#each products as p}
			<option value={String(p.id)}>{p.name}</option>
		{/each}
	</select>
	<select class="form-select" bind:value={filterProcessed} onchange={() => (currentPage = 1)}>
		<option value="all">All Messages</option>
		<option value="processed">Processed</option>
		<option value="unprocessed">Unprocessed</option>
	</select>
	<div class="ml-auto">
		<button class="btn-outline-destructive btn-sm" onclick={() => (confirmDeleteFilter = true)}>
			<Trash2 class="w-3 h-3" />
			Delete Filtered
		</button>
	</div>
</div>

{#if loading}
	<div class="card">
		<div class="p-8">
			{#each Array(5) as _}
				<div class="skeleton h-10 w-full mb-2"></div>
			{/each}
		</div>
	</div>
{:else if messages.length === 0}
	<div class="card">
		<div class="empty-state">
			<MessageSquare class="w-10 h-10 empty-state-icon" />
			<p class="empty-state-title">No messages</p>
			<p class="empty-state-text">Messages appear when pipelines collect data from platform sources</p>
		</div>
	</div>
{:else}
	<div class="card overflow-hidden">
		<div class="overflow-x-auto">
			<table class="data-table">
				<thead>
					<tr>
						<th>Author</th>
						<th>Content</th>
						<th>Platform</th>
						<th>Channel</th>
						<th>Status</th>
						<th>Time</th>
					</tr>
				</thead>
				<tbody>
					{#each messages as msg}
						<tr>
							<td class="font-medium whitespace-nowrap">{msg.authorName || msg.authorId}</td>
							<td class="max-w-[300px]">
								<span class="truncate-2 text-sm">{msg.content}</span>
							</td>
							<td><span class="badge badge-muted">{msg.platformType}</span></td>
							<td class="text-xs text-muted-foreground font-mono">{msg.channelId}</td>
							<td>
								{#if msg.processed}
									<span class="flex items-center gap-1 text-xs text-success">
										<CheckCircle class="w-3 h-3" /> Processed
									</span>
								{:else}
									<span class="flex items-center gap-1 text-xs text-muted-foreground">
										<Circle class="w-3 h-3" /> Pending
									</span>
								{/if}
							</td>
							<td class="text-xs text-muted-foreground whitespace-nowrap">{formatRelative(msg.timestamp || msg.createdAt)}</td>
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
					<button class="pagination-btn" disabled={currentPage <= 1} onclick={() => (currentPage = Math.max(1, currentPage - 1))}>
						<ChevronLeft class="w-4 h-4" />
					</button>
					<button class="pagination-btn" disabled={currentPage >= totalPages} onclick={() => (currentPage = Math.min(totalPages, currentPage + 1))}>
						<ChevronRight class="w-4 h-4" />
					</button>
				</div>
			</div>
		{/if}
	</div>
{/if}

{#if confirmDeleteFilter}
	<div class="modal-backdrop" onclick={() => (confirmDeleteFilter = false)} role="presentation">
		<div class="confirm-dialog" onclick={(e) => e.stopPropagation()} role="dialog">
			<div class="modal-header">
				<h3 class="modal-title">Delete Messages</h3>
			</div>
			<div class="modal-body">
				<p class="text-sm">This will permanently delete all messages matching the current filters. This action cannot be undone.</p>
				<div class="mt-2 p-2 rounded bg-muted text-xs space-y-1">
					<p><strong>Product:</strong> {filterProductId ? products.find((p) => String(p.id) === filterProductId)?.name : 'All'}</p>
					<p><strong>Status:</strong> {filterProcessed === 'all' ? 'All' : filterProcessed}</p>
				</div>
			</div>
			<div class="modal-footer">
				<button class="btn-secondary" onclick={() => (confirmDeleteFilter = false)}>Cancel</button>
				<button class="btn-destructive" onclick={deleteFiltered}>Delete Messages</button>
			</div>
		</div>
	</div>
{/if}
