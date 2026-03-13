<script lang="ts">
	import { rpcClient } from '$lib/api/rpc-client';
	import { toast } from 'svelte-sonner';
	import { formatDate } from '$lib/utils';
	import type {
		Pipeline,
		PipelineStatus,
		Product,
		Platform,
		Agent,
		Reporter
	} from '$lib/proto/discordiance/v1/types_pb';
	import {
		Plus,
		Play,
		Square,
		RefreshCw,
		RotateCcw,
		Pencil,
		Trash2,
		X,
		GitBranch
	} from '@lucide/svelte';

	let pipelines = $state<Pipeline[]>([]);
	let statuses = $state<PipelineStatus[]>([]);
	let products = $state<Product[]>([]);
	let platforms = $state<Platform[]>([]);
	let agents = $state<Agent[]>([]);
	let reporters = $state<Reporter[]>([]);
	let loading = $state(true);

	let showModal = $state(false);
	let editingPipeline = $state<Pipeline | null>(null);
	let confirmDelete = $state<Pipeline | null>(null);
	let saving = $state(false);

	let formName = $state('');
	let formProductId = $state('');
	let formPlatformId = $state('');
	let formAgentId = $state('');
	let formReporterIds = $state<string[]>([]);
	let formEnabled = $state(true);

	async function load() {
		loading = true;
		try {
			const [pipeRes, statusRes, prodRes, platRes, agentRes, repRes] = await Promise.all([
				rpcClient.pipeline.listPipelines({}),
				rpcClient.pipeline.getPipelineStatus({}),
				rpcClient.product.listProducts({}),
				rpcClient.platform.listPlatforms({}),
				rpcClient.agent.listAgents({}),
				rpcClient.reporter.listReporters({})
			]);
			pipelines = pipeRes.pipelines;
			statuses = statusRes.statuses;
			products = prodRes.products;
			platforms = platRes.platforms;
			agents = agentRes.agents;
			reporters = repRes.reporters;
		} catch {
			// handled by interceptor
		} finally {
			loading = false;
		}
	}

	$effect(() => { load(); });

	function getStatus(pipelineId: bigint): PipelineStatus | undefined {
		return statuses.find((s) => s.pipelineId === pipelineId);
	}

	function openCreate() {
		editingPipeline = null;
		formName = '';
		formProductId = '';
		formPlatformId = '';
		formAgentId = '';
		formReporterIds = [];
		formEnabled = true;
		showModal = true;
	}

	function openEdit(p: Pipeline) {
		editingPipeline = p;
		formName = p.name;
		formProductId = String(p.productId);
		formPlatformId = String(p.platformId);
		formAgentId = String(p.agentId);
		formReporterIds = p.reporterIds.map(String);
		formEnabled = p.enabled;
		showModal = true;
	}

	async function save() {
		if (!formName || !formProductId || !formPlatformId || !formAgentId) return;
		saving = true;
		try {
			const data = {
				name: formName,
				productId: BigInt(formProductId),
				platformId: BigInt(formPlatformId),
				agentId: BigInt(formAgentId),
				reporterIds: formReporterIds.map(BigInt),
				enabled: formEnabled
			};
			if (editingPipeline) {
				await rpcClient.pipeline.updatePipeline({ id: editingPipeline.id, ...data });
				toast.success('Pipeline updated');
			} else {
				await rpcClient.pipeline.createPipeline(data);
				toast.success('Pipeline created');
			}
			showModal = false;
			load();
		} catch {
			// handled
		} finally {
			saving = false;
		}
	}

	async function deletePipeline(p: Pipeline) {
		try {
			await rpcClient.pipeline.deletePipeline({ id: p.id });
			toast.success('Pipeline deleted');
			confirmDelete = null;
			load();
		} catch {
			// handled
		}
	}

	async function startPipeline(id: bigint) {
		try {
			await rpcClient.pipeline.startPipeline({ pipelineId: id });
			toast.success('Pipeline started');
			load();
		} catch { /* handled */ }
	}

	async function stopPipeline(id: bigint) {
		try {
			await rpcClient.pipeline.stopPipeline({ pipelineId: id });
			toast.success('Pipeline stopped');
			load();
		} catch { /* handled */ }
	}

	async function restartPipeline(id: bigint) {
		try {
			await rpcClient.pipeline.restartPipeline({ pipelineId: id });
			toast.success('Pipeline restarted');
			load();
		} catch { /* handled */ }
	}

	async function backfillPipeline(id: bigint) {
		try {
			await rpcClient.pipeline.backfillPipeline({ pipelineId: id });
			toast.success('Backfill triggered');
		} catch { /* handled */ }
	}

	function toggleReporter(id: string) {
		if (formReporterIds.includes(id)) {
			formReporterIds = formReporterIds.filter((r) => r !== id);
		} else {
			formReporterIds = [...formReporterIds, id];
		}
	}
</script>

<div class="page-header">
	<div>
		<h1 class="page-title">Pipelines</h1>
		<p class="page-subtitle">Manage message processing pipelines</p>
	</div>
	<button class="btn-primary" onclick={openCreate}>
		<Plus class="w-4 h-4" />
		Create Pipeline
	</button>
</div>

{#if loading}
	<div class="card">
		<div class="p-8">
			<div class="skeleton h-4 w-48 mb-4"></div>
			{#each Array(3) as _}
				<div class="skeleton h-10 w-full mb-2"></div>
			{/each}
		</div>
	</div>
{:else if pipelines.length === 0}
	<div class="card">
		<div class="empty-state">
			<GitBranch class="w-10 h-10 empty-state-icon" />
			<p class="empty-state-title">No pipelines</p>
			<p class="empty-state-text">Pipelines connect a product, platform source, AI agent, and reporters into a processing workflow</p>
			<button class="btn-primary btn-sm" onclick={openCreate}>Create your first pipeline</button>
		</div>
	</div>
{:else}
	<div class="card overflow-hidden">
		<div class="overflow-x-auto">
			<table class="data-table">
				<thead>
					<tr>
						<th>Name</th>
						<th>Product</th>
						<th>Platform</th>
						<th>Agent</th>
						<th>Reporters</th>
						<th>Status</th>
						<th>Health</th>
						<th class="text-right">Actions</th>
					</tr>
				</thead>
				<tbody>
					{#each pipelines as pipeline}
						{@const status = getStatus(pipeline.id)}
						<tr>
							<td>
								<a href="/pipelines/{pipeline.id}" class="font-medium text-foreground hover:text-primary transition-colors">
									{pipeline.name}
								</a>
							</td>
							<td class="text-muted-foreground">{pipeline.product?.name || '—'}</td>
							<td class="text-muted-foreground">{pipeline.platform?.name || '—'}</td>
							<td class="text-muted-foreground">{pipeline.agent?.name || '—'}</td>
							<td class="text-muted-foreground">
								{#if pipeline.reporters.length > 0}
									{pipeline.reporters.map((r) => r.name).join(', ')}
								{:else}
									—
								{/if}
							</td>
							<td>
								{#if status?.running}
									<span class="badge badge-success"><span class="badge-dot bg-success"></span> Running</span>
								{:else if status}
									<span class="badge badge-muted"><span class="badge-dot bg-muted-foreground"></span> Stopped</span>
								{:else}
									<span class="badge badge-muted">—</span>
								{/if}
							</td>
							<td>
								{#if status?.running && status.healthy}
									<span class="badge badge-success">Healthy</span>
								{:else if status?.running}
									<span class="badge badge-destructive">Unhealthy</span>
								{:else}
									<span class="text-xs text-muted-foreground">—</span>
								{/if}
							</td>
							<td>
								<div class="flex items-center justify-end gap-1">
									{#if status?.running}
										<button class="btn-icon-sm" title="Stop" onclick={() => stopPipeline(pipeline.id)}>
											<Square class="w-3 h-3" />
										</button>
										<button class="btn-icon-sm" title="Restart" onclick={() => restartPipeline(pipeline.id)}>
											<RefreshCw class="w-3 h-3" />
										</button>
									{:else}
										<button class="btn-icon-sm" title="Start" onclick={() => startPipeline(pipeline.id)}>
											<Play class="w-3 h-3" />
										</button>
									{/if}
									<button class="btn-icon-sm" title="Backfill" onclick={() => backfillPipeline(pipeline.id)}>
										<RotateCcw class="w-3 h-3" />
									</button>
									<button class="btn-icon-sm" title="Edit" onclick={() => openEdit(pipeline)}>
										<Pencil class="w-3 h-3" />
									</button>
									<button class="btn-icon-sm text-destructive" title="Delete" onclick={() => (confirmDelete = pipeline)}>
										<Trash2 class="w-3 h-3" />
									</button>
								</div>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	</div>
{/if}

{#if showModal}
	<div class="modal-backdrop" onclick={() => (showModal = false)} role="presentation">
		<div class="modal" onclick={(e) => e.stopPropagation()} role="dialog">
			<div class="modal-header">
				<h3 class="modal-title">{editingPipeline ? 'Edit Pipeline' : 'Create Pipeline'}</h3>
				<button class="btn-icon-sm" onclick={() => (showModal = false)}><X class="w-4 h-4" /></button>
			</div>
			<div class="modal-body">
				<div class="form-group">
					<label class="form-label" for="pName">Name</label>
					<input id="pName" class="form-input" bind:value={formName} placeholder="My Pipeline" />
				</div>
				<div class="form-group">
					<label class="form-label" for="pProduct">Product</label>
					<select id="pProduct" class="form-select" bind:value={formProductId}>
						<option value="">Select a product</option>
						{#each products as p}
							<option value={String(p.id)}>{p.name}</option>
						{/each}
					</select>
				</div>
				<div class="form-group">
					<label class="form-label" for="pPlatform">Platform Source</label>
					<select id="pPlatform" class="form-select" bind:value={formPlatformId}>
						<option value="">Select a platform</option>
						{#each platforms as p}
							<option value={String(p.id)}>{p.name} ({p.type})</option>
						{/each}
					</select>
				</div>
				<div class="form-group">
					<label class="form-label" for="pAgent">AI Agent</label>
					<select id="pAgent" class="form-select" bind:value={formAgentId}>
						<option value="">Select an agent</option>
						{#each agents as a}
							<option value={String(a.id)}>{a.name} ({a.model})</option>
						{/each}
					</select>
				</div>
				<div class="form-group">
					<label class="form-label">Reporters</label>
					{#if reporters.length === 0}
						<p class="text-xs text-muted-foreground">No reporters available. <a href="/settings/reporters" class="text-primary hover:underline">Create one</a></p>
					{:else}
						<div class="space-y-1.5">
							{#each reporters as r}
								<label class="flex items-center gap-2 px-2 py-1.5 rounded hover:bg-muted/50 cursor-pointer text-sm">
									<input type="checkbox" checked={formReporterIds.includes(String(r.id))} onchange={() => toggleReporter(String(r.id))} class="accent-primary" />
									<span>{r.name}</span>
									<span class="text-xs text-muted-foreground">({r.type})</span>
								</label>
							{/each}
						</div>
					{/if}
				</div>
				<div class="flex items-center gap-3">
					<label class="form-label mb-0">Enabled</label>
					<button type="button" class="form-toggle" class:active={formEnabled} onclick={() => (formEnabled = !formEnabled)}></button>
				</div>
			</div>
			<div class="modal-footer">
				<button class="btn-secondary" onclick={() => (showModal = false)}>Cancel</button>
				<button class="btn-primary" onclick={save} disabled={saving || !formName || !formProductId || !formPlatformId || !formAgentId}>
					{saving ? 'Saving...' : editingPipeline ? 'Update' : 'Create'}
				</button>
			</div>
		</div>
	</div>
{/if}

{#if confirmDelete}
	<div class="modal-backdrop" onclick={() => (confirmDelete = null)} role="presentation">
		<div class="confirm-dialog" onclick={(e) => e.stopPropagation()} role="dialog">
			<div class="modal-header">
				<h3 class="modal-title">Delete Pipeline</h3>
			</div>
			<div class="modal-body">
				<p class="text-sm">Are you sure you want to delete <strong>{confirmDelete.name}</strong>? This action cannot be undone.</p>
			</div>
			<div class="modal-footer">
				<button class="btn-secondary" onclick={() => (confirmDelete = null)}>Cancel</button>
				<button class="btn-destructive" onclick={() => confirmDelete && deletePipeline(confirmDelete)}>Delete</button>
			</div>
		</div>
	</div>
{/if}
