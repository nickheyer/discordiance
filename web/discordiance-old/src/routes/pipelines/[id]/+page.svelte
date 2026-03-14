<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { rpcClient } from '$lib/api/rpc-client';
	import { toast } from 'svelte-sonner';
	import { formatDate } from '$lib/utils';
	import type { Pipeline, PipelineStatus } from '$lib/proto/discordiance/v1/types_pb';
	import {
		ChevronRight,
		Play,
		Square,
		RefreshCw,
		RotateCcw,
		Trash2,
		Server,
		Bot,
		Send,
		Package
	} from '@lucide/svelte';

	let pipeline = $state<Pipeline | null>(null);
	let status = $state<PipelineStatus | null>(null);
	let loading = $state(true);
	let confirmDelete = $state(false);

	const pipelineId = $derived(BigInt(page.params.id!));

	async function load() {
		loading = true;
		try {
			const [pRes, statusRes] = await Promise.all([
				rpcClient.pipeline.getPipeline({ id: pipelineId }),
				rpcClient.pipeline.getPipelineStatus({})
			]);
			pipeline = pRes.pipeline ?? null;
			status = statusRes.statuses.find((s) => s.pipelineId === pipelineId) ?? null;
		} catch {
			// handled
		} finally {
			loading = false;
		}
	}

	$effect(() => { load(); });

	async function startPipeline() {
		await rpcClient.pipeline.startPipeline({ pipelineId });
		toast.success('Pipeline started');
		load();
	}

	async function stopPipeline() {
		await rpcClient.pipeline.stopPipeline({ pipelineId });
		toast.success('Pipeline stopped');
		load();
	}

	async function restartPipeline() {
		await rpcClient.pipeline.restartPipeline({ pipelineId });
		toast.success('Pipeline restarted');
		load();
	}

	async function backfillPipeline() {
		await rpcClient.pipeline.backfillPipeline({ pipelineId });
		toast.success('Backfill triggered');
	}

	async function deletePipeline() {
		await rpcClient.pipeline.deletePipeline({ id: pipelineId });
		toast.success('Pipeline deleted');
		goto('/pipelines');
	}
</script>

<div class="breadcrumb">
	<a href="/pipelines">Pipelines</a>
	<ChevronRight class="w-3 h-3 separator" />
	<span class="current">{pipeline?.name || '...'}</span>
</div>

{#if loading}
	<div class="space-y-4">
		<div class="skeleton h-8 w-64"></div>
		<div class="detail-grid">
			{#each Array(4) as _}
				<div class="card"><div class="p-5"><div class="skeleton h-20 w-full"></div></div></div>
			{/each}
		</div>
	</div>
{:else if pipeline}
	<div class="flex items-start justify-between mb-6">
		<div>
			<h1 class="page-title">{pipeline.name}</h1>
			<div class="flex items-center gap-3 mt-2">
				{#if pipeline.enabled}
					<span class="badge badge-success">Enabled</span>
				{:else}
					<span class="badge badge-muted">Disabled</span>
				{/if}
				{#if status?.running}
					<span class="badge badge-success"><span class="badge-dot bg-success"></span> Running</span>
					{#if status.healthy}
						<span class="badge badge-success">Healthy</span>
					{:else}
						<span class="badge badge-destructive">Unhealthy</span>
					{/if}
				{:else}
					<span class="badge badge-muted"><span class="badge-dot bg-muted-foreground"></span> Stopped</span>
				{/if}
			</div>
		</div>
		<div class="flex items-center gap-2">
			{#if status?.running}
				<button class="btn-secondary btn-sm" onclick={stopPipeline}>
					<Square class="w-3 h-3" /> Stop
				</button>
				<button class="btn-secondary btn-sm" onclick={restartPipeline}>
					<RefreshCw class="w-3 h-3" /> Restart
				</button>
			{:else}
				<button class="btn-primary btn-sm" onclick={startPipeline}>
					<Play class="w-3 h-3" /> Start
				</button>
			{/if}
			<button class="btn-secondary btn-sm" onclick={backfillPipeline}>
				<RotateCcw class="w-3 h-3" /> Backfill
			</button>
			<button class="btn-outline-destructive btn-sm" onclick={() => (confirmDelete = true)}>
				<Trash2 class="w-3 h-3" /> Delete
			</button>
		</div>
	</div>

	<div class="detail-grid">
		<div class="card">
			<div class="card-header">
				<h3 class="card-title flex items-center gap-2"><Package class="w-4 h-4 text-muted-foreground" /> Product</h3>
				{#if pipeline.product}
					<a href="/products/{pipeline.product.id}" class="btn-ghost btn-xs text-primary">View</a>
				{/if}
			</div>
			<div class="card-body space-y-2">
				{#if pipeline.product}
					<div class="detail-row"><span class="detail-label">Name</span><span class="detail-value">{pipeline.product.name}</span></div>
					<div class="detail-row"><span class="detail-label">Description</span><span class="detail-value text-sm">{pipeline.product.description || '—'}</span></div>
					<div class="detail-row"><span class="detail-label">Status</span>
						{#if pipeline.product.enabled}
							<span class="badge badge-success">Enabled</span>
						{:else}
							<span class="badge badge-muted">Disabled</span>
						{/if}
					</div>
				{:else}
					<p class="text-sm text-muted-foreground">No product assigned</p>
				{/if}
			</div>
		</div>

		<div class="card">
			<div class="card-header">
				<h3 class="card-title flex items-center gap-2"><Server class="w-4 h-4 text-muted-foreground" /> Platform Source</h3>
			</div>
			<div class="card-body space-y-2">
				{#if pipeline.platform}
					<div class="detail-row"><span class="detail-label">Name</span><span class="detail-value">{pipeline.platform.name}</span></div>
					<div class="detail-row"><span class="detail-label">Type</span><span class="badge badge-info">{pipeline.platform.type}</span></div>
					<div class="detail-row"><span class="detail-label">Status</span>
						{#if pipeline.platform.enabled}
							<span class="badge badge-success">Enabled</span>
						{:else}
							<span class="badge badge-muted">Disabled</span>
						{/if}
					</div>
				{:else}
					<p class="text-sm text-muted-foreground">No platform assigned</p>
				{/if}
			</div>
		</div>

		<div class="card">
			<div class="card-header">
				<h3 class="card-title flex items-center gap-2"><Bot class="w-4 h-4 text-muted-foreground" /> AI Agent</h3>
			</div>
			<div class="card-body space-y-2">
				{#if pipeline.agent}
					<div class="detail-row"><span class="detail-label">Name</span><span class="detail-value">{pipeline.agent.name}</span></div>
					<div class="detail-row"><span class="detail-label">Model</span><span class="badge badge-primary">{pipeline.agent.model}</span></div>
					<div class="detail-row"><span class="detail-label">Batch Size</span><span class="detail-value">{pipeline.agent.batchSize}</span></div>
					<div class="detail-row"><span class="detail-label">Batch Timeout</span><span class="detail-value">{pipeline.agent.batchTimeout}s</span></div>
				{:else}
					<p class="text-sm text-muted-foreground">No agent assigned</p>
				{/if}
			</div>
		</div>

		<div class="card">
			<div class="card-header">
				<h3 class="card-title flex items-center gap-2"><Send class="w-4 h-4 text-muted-foreground" /> Reporters</h3>
			</div>
			<div class="card-body">
				{#if pipeline.reporters.length > 0}
					<div class="space-y-2">
						{#each pipeline.reporters as reporter}
							<div class="detail-row">
								<span class="detail-value">{reporter.name}</span>
								<span class="badge badge-info">{reporter.type}</span>
							</div>
						{/each}
					</div>
				{:else}
					<p class="text-sm text-muted-foreground">No reporters assigned</p>
				{/if}
			</div>
		</div>
	</div>

	<div class="mt-5 card">
		<div class="card-body">
			<div class="flex items-center gap-6 text-xs text-muted-foreground">
				<span>Created {formatDate(pipeline.createdAt)}</span>
				<span>Updated {formatDate(pipeline.updatedAt)}</span>
				<span>ID: {String(pipeline.id)}</span>
			</div>
		</div>
	</div>

	{#if confirmDelete}
		<div class="modal-backdrop" onclick={() => (confirmDelete = false)} role="presentation">
			<div class="confirm-dialog" onclick={(e) => e.stopPropagation()} role="dialog">
				<div class="modal-header"><h3 class="modal-title">Delete Pipeline</h3></div>
				<div class="modal-body">
					<p class="text-sm">Are you sure you want to delete <strong>{pipeline.name}</strong>?</p>
				</div>
				<div class="modal-footer">
					<button class="btn-secondary" onclick={() => (confirmDelete = false)}>Cancel</button>
					<button class="btn-destructive" onclick={deletePipeline}>Delete</button>
				</div>
			</div>
		</div>
	{/if}
{:else}
	<div class="card">
		<div class="empty-state">
			<p class="empty-state-title">Pipeline not found</p>
			<a href="/pipelines" class="btn-primary btn-sm">Back to Pipelines</a>
		</div>
	</div>
{/if}
