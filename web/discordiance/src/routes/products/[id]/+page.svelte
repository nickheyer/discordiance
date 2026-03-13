<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { rpcClient } from '$lib/api/rpc-client';
	import { toast } from 'svelte-sonner';
	import { formatDate, formatBytes, severityColor, statusColor } from '$lib/utils';
	import type { Product, Insight } from '$lib/proto/discordiance/v1/types_pb';
	import type { ProductFile } from '$lib/proto/discordiance/v1/file_pb';
	import {
		ChevronRight,
		Trash2,
		Upload,
		File,
		ExternalLink,
		KeyRound,
		Lightbulb,
		GitBranch
	} from '@lucide/svelte';

	let product = $state<Product | null>(null);
	let files = $state<ProductFile[]>([]);
	let totalFileSize = $state<bigint>(BigInt(0));
	let insights = $state<Insight[]>([]);
	let loading = $state(true);
	let uploading = $state(false);
	let confirmDelete = $state(false);

	const productId = $derived(BigInt(page.params.id!));

	async function load() {
		loading = true;
		try {
			const [prodRes, fileRes, insightRes] = await Promise.all([
				rpcClient.product.getProduct({ id: productId }),
				rpcClient.productFile.listProductFiles({ productId }),
				rpcClient.insight.listInsights({ productId, pageSize: 10, page: 1 })
			]);
			product = prodRes.product ?? null;
			files = fileRes.files;
			totalFileSize = fileRes.totalSize;
			insights = insightRes.insights;
		} catch {
			// handled
		} finally {
			loading = false;
		}
	}

	$effect(() => { load(); });

	async function handleFileUpload(event: Event) {
		const input = event.target as HTMLInputElement;
		const file = input.files?.[0];
		if (!file) return;
		uploading = true;
		try {
			const buffer = await file.arrayBuffer();
			await rpcClient.productFile.uploadProductFile({
				productId,
				filename: file.name,
				mimeType: file.type || 'application/octet-stream',
				content: new Uint8Array(buffer)
			});
			toast.success(`Uploaded ${file.name}`);
			load();
		} catch {
			// handled
		} finally {
			uploading = false;
			input.value = '';
		}
	}

	async function deleteFile(fileId: bigint, filename: string) {
		try {
			await rpcClient.productFile.deleteProductFile({ id: fileId });
			toast.success(`Deleted ${filename}`);
			load();
		} catch {
			// handled
		}
	}

	async function deleteProduct() {
		await rpcClient.product.deleteProduct({ id: productId });
		toast.success('Product deleted');
		goto('/products');
	}
</script>

<div class="breadcrumb">
	<a href="/products">Products</a>
	<ChevronRight class="w-3 h-3 separator" />
	<span class="current">{product?.name || '...'}</span>
</div>

{#if loading}
	<div class="space-y-4">
		<div class="skeleton h-8 w-64"></div>
		<div class="skeleton h-40 w-full"></div>
	</div>
{:else if product}
	<div class="flex items-start justify-between mb-6">
		<div>
			<h1 class="page-title">{product.name}</h1>
			<div class="flex items-center gap-3 mt-2">
				{#if product.enabled}
					<span class="badge badge-success">Enabled</span>
				{:else}
					<span class="badge badge-muted">Disabled</span>
				{/if}
				{#if product.hasGithubToken}
					<span class="badge badge-info"><KeyRound class="w-3 h-3" /> GitHub Connected</span>
				{/if}
			</div>
		</div>
		<button class="btn-outline-destructive btn-sm" onclick={() => (confirmDelete = true)}>
			<Trash2 class="w-3 h-3" /> Delete
		</button>
	</div>

	<div class="grid grid-cols-1 lg:grid-cols-3 gap-5 mb-6">
		<div class="lg:col-span-2 card">
			<div class="card-header">
				<h3 class="card-title">Details</h3>
			</div>
			<div class="card-body space-y-2">
				<div class="detail-row"><span class="detail-label">Description</span><span class="detail-value">{product.description || '—'}</span></div>
				<div class="detail-row">
					<span class="detail-label">Repository</span>
					{#if product.repoUrl}
						<a href={product.repoUrl} target="_blank" rel="noopener" class="detail-value text-primary flex items-center gap-1 hover:underline">
							{product.repoUrl} <ExternalLink class="w-3 h-3" />
						</a>
					{:else}
						<span class="detail-value">—</span>
					{/if}
				</div>
				<div class="detail-row"><span class="detail-label">Created</span><span class="detail-value">{formatDate(product.createdAt)}</span></div>
				<div class="detail-row"><span class="detail-label">Updated</span><span class="detail-value">{formatDate(product.updatedAt)}</span></div>
			</div>
		</div>

		<div class="card">
			<div class="card-header">
				<h3 class="card-title">Context Files</h3>
				<label class="btn-ghost btn-sm cursor-pointer">
					<Upload class="w-3 h-3" />
					{uploading ? 'Uploading...' : 'Upload'}
					<input type="file" class="hidden" onchange={handleFileUpload} disabled={uploading} />
				</label>
			</div>
			<div class="card-body">
				{#if files.length === 0}
					<p class="text-sm text-muted-foreground text-center py-4">No files uploaded</p>
				{:else}
					<div class="space-y-1">
						{#each files as file}
							<div class="flex items-center justify-between py-1.5 px-2 rounded hover:bg-muted/50 group">
								<div class="flex items-center gap-2 min-w-0">
									<File class="w-3.5 h-3.5 text-muted-foreground shrink-0" />
									<span class="text-sm truncate">{file.filename}</span>
									<span class="text-xs text-muted-foreground">{formatBytes(file.size)}</span>
								</div>
								<button class="btn-icon-sm text-destructive opacity-0 group-hover:opacity-100 transition-opacity" onclick={() => deleteFile(file.id, file.filename)}>
									<Trash2 class="w-3 h-3" />
								</button>
							</div>
						{/each}
					</div>
					<div class="mt-3 pt-2 border-t border-border text-xs text-muted-foreground">
						{files.length} files, {formatBytes(totalFileSize)} total
					</div>
				{/if}
			</div>
		</div>
	</div>

	<div class="card">
		<div class="card-header">
			<h3 class="card-title flex items-center gap-2"><Lightbulb class="w-4 h-4 text-muted-foreground" /> Recent Insights</h3>
		</div>
		{#if insights.length === 0}
			<div class="empty-state py-10">
				<p class="empty-state-title">No insights yet</p>
				<p class="empty-state-text">Insights for this product will appear here once pipelines process messages</p>
			</div>
		{:else}
			<div class="overflow-x-auto">
				<table class="data-table">
					<thead>
						<tr>
							<th>Title</th>
							<th>Severity</th>
							<th>Category</th>
							<th>Status</th>
						</tr>
					</thead>
					<tbody>
						{#each insights as insight}
							<tr class="row-link" onclick={() => { window.location.href = `/insights/${insight.id}` }}>
								<td class="font-medium">{insight.title}</td>
								<td><span class="badge {severityColor(insight.severity)}">{insight.severity}</span></td>
								<td class="text-muted-foreground">{insight.category}</td>
								<td><span class="badge {statusColor(insight.status)}">{insight.status}</span></td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	</div>

	{#if confirmDelete}
		<div class="modal-backdrop" onclick={() => (confirmDelete = false)} role="presentation">
			<div class="confirm-dialog" onclick={(e) => e.stopPropagation()} role="dialog">
				<div class="modal-header"><h3 class="modal-title">Delete Product</h3></div>
				<div class="modal-body">
					<p class="text-sm">Are you sure you want to delete <strong>{product.name}</strong>? All associated data will be removed.</p>
				</div>
				<div class="modal-footer">
					<button class="btn-secondary" onclick={() => (confirmDelete = false)}>Cancel</button>
					<button class="btn-destructive" onclick={deleteProduct}>Delete</button>
				</div>
			</div>
		</div>
	{/if}
{:else}
	<div class="card">
		<div class="empty-state">
			<p class="empty-state-title">Product not found</p>
			<a href="/products" class="btn-primary btn-sm">Back to Products</a>
		</div>
	</div>
{/if}
