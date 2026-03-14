<script lang="ts">
	import { rpcClient } from '$lib/api/rpc-client';
	import { toast } from 'svelte-sonner';
	import { formatDate } from '$lib/utils';
	import type { Agent } from '$lib/proto/discordiance/v1/types_pb';
	import { Plus, Pencil, Trash2, X, Bot } from '@lucide/svelte';

	let configs = $state<Agent[]>([]);
	let loading = $state(true);
	let showModal = $state(false);
	let editing = $state<Agent | null>(null);
	let confirmDelete = $state<Agent | null>(null);
	let saving = $state(false);

	let formName = $state('');
	let formBaseUrl = $state('');
	let formApiKey = $state('');
	let formOrgId = $state('');
	let formModel = $state('');
	let formSystemPrompt = $state('');
	let formBatchSize = $state(10);
	let formBatchTimeout = $state(30);

	async function load() {
		loading = true;
		try {
			const res = await rpcClient.agent.listAgents({});
			configs = res.agents;
		} catch {
			// handled
		} finally {
			loading = false;
		}
	}

	$effect(() => { load(); });

	function openCreate() {
		editing = null;
		formName = '';
		formBaseUrl = '';
		formApiKey = '';
		formOrgId = '';
		formModel = '';
		formSystemPrompt = '';
		formBatchSize = 10;
		formBatchTimeout = 30;
		showModal = true;
	}

	function openEdit(c: Agent) {
		editing = c;
		formName = c.name;
		formBaseUrl = c.baseUrl;
		formApiKey = '';
		formOrgId = c.orgId;
		formModel = c.model;
		formSystemPrompt = c.systemPrompt;
		formBatchSize = c.batchSize;
		formBatchTimeout = c.batchTimeout;
		showModal = true;
	}

	async function save() {
		if (!formName || !formModel) return;
		saving = true;
		try {
			const data = {
				name: formName,
				baseUrl: formBaseUrl,
				apiKey: formApiKey,
				orgId: formOrgId,
				model: formModel,
				systemPrompt: formSystemPrompt,
				batchSize: formBatchSize,
				batchTimeout: formBatchTimeout
			};
			if (editing) {
				await rpcClient.agent.updateAgent({ id: editing.id, ...data });
				toast.success('Agent updated');
			} else {
				await rpcClient.agent.createAgent(data);
				toast.success('Agent created');
			}
			showModal = false;
			load();
		} catch {
			// handled
		} finally {
			saving = false;
		}
	}

	async function deleteConfig(c: Agent) {
		try {
			await rpcClient.agent.deleteAgent({ id: c.id });
			toast.success('Agent deleted');
			confirmDelete = null;
			load();
		} catch {
			// handled
		}
	}
</script>

<div class="page-header">
	<div>
		<h1 class="page-title">Agents</h1>
		<p class="page-subtitle">Manage LLM agent settings for insight analysis</p>
	</div>
	<button class="btn-primary" onclick={openCreate}>
		<Plus class="w-4 h-4" />
		Create Agent
	</button>
</div>

{#if loading}
	<div class="card"><div class="p-8">{#each Array(3) as _}<div class="skeleton h-10 w-full mb-2"></div>{/each}</div></div>
{:else if configs.length === 0}
	<div class="card">
		<div class="empty-state">
			<Bot class="w-10 h-10 empty-state-icon" />
			<p class="empty-state-title">No agents</p>
			<p class="empty-state-text">Agents define the LLM settings used to analyze messages and extract insights</p>
			<button class="btn-primary btn-sm" onclick={openCreate}>Create agent</button>
		</div>
	</div>
{:else}
	<div class="card overflow-hidden">
		<div class="overflow-x-auto">
			<table class="data-table">
				<thead>
					<tr>
						<th>Name</th>
						<th>Model</th>
						<th>Base URL</th>
						<th>Batch Size</th>
						<th>Timeout</th>
						<th>Created</th>
						<th class="text-right">Actions</th>
					</tr>
				</thead>
				<tbody>
					{#each configs as config}
						<tr>
							<td class="font-medium">{config.name}</td>
							<td><span class="badge badge-primary">{config.model}</span></td>
							<td class="text-xs text-muted-foreground font-mono max-w-[180px] truncate">{config.baseUrl || 'Default'}</td>
							<td class="text-muted-foreground">{config.batchSize}</td>
							<td class="text-muted-foreground">{config.batchTimeout}s</td>
							<td class="text-xs text-muted-foreground whitespace-nowrap">{formatDate(config.createdAt)}</td>
							<td>
								<div class="flex items-center justify-end gap-1">
									<button class="btn-icon-sm" title="Edit" onclick={() => openEdit(config)}><Pencil class="w-3 h-3" /></button>
									<button class="btn-icon-sm text-destructive" title="Delete" onclick={() => (confirmDelete = config)}><Trash2 class="w-3 h-3" /></button>
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
				<h3 class="modal-title">{editing ? 'Edit Agent' : 'Create Agent'}</h3>
				<button class="btn-icon-sm" onclick={() => (showModal = false)}><X class="w-4 h-4" /></button>
			</div>
			<div class="modal-body">
				<div class="form-group">
					<label class="form-label" for="agName">Name</label>
					<input id="agName" class="form-input" bind:value={formName} placeholder="GPT-4 Analyzer" />
				</div>
				<div class="form-group">
					<label class="form-label" for="agModel">Model</label>
					<input id="agModel" class="form-input" bind:value={formModel} placeholder="gpt-4o" />
				</div>
				<div class="form-group">
					<label class="form-label" for="agBaseUrl">Base URL</label>
					<input id="agBaseUrl" class="form-input" bind:value={formBaseUrl} placeholder="https://api.openai.com/v1" />
					<p class="form-hint">Leave blank for provider default</p>
				</div>
				<div class="form-group">
					<label class="form-label" for="agApiKey">API Key</label>
					<input id="agApiKey" class="form-input" type="password" bind:value={formApiKey} placeholder={editing ? '••••••••' : 'sk-...'} />
					{#if editing}
						<p class="form-hint">Leave blank to keep existing key</p>
					{/if}
				</div>
				<div class="form-group">
					<label class="form-label" for="agOrgId">Organization ID</label>
					<input id="agOrgId" class="form-input" bind:value={formOrgId} placeholder="org-..." />
					<p class="form-hint">Optional organization identifier</p>
				</div>
				<div class="form-group">
					<label class="form-label" for="agPrompt">System Prompt</label>
					<textarea id="agPrompt" class="form-textarea" rows="4" bind:value={formSystemPrompt} placeholder="You are an AI agent that analyzes community messages..."></textarea>
				</div>
				<div class="grid grid-cols-2 gap-3">
					<div class="form-group">
						<label class="form-label" for="agBatch">Batch Size</label>
						<input id="agBatch" class="form-input" type="number" min="1" bind:value={formBatchSize} />
					</div>
					<div class="form-group">
						<label class="form-label" for="agTimeout">Batch Timeout (s)</label>
						<input id="agTimeout" class="form-input" type="number" min="1" bind:value={formBatchTimeout} />
					</div>
				</div>
			</div>
			<div class="modal-footer">
				<button class="btn-secondary" onclick={() => (showModal = false)}>Cancel</button>
				<button class="btn-primary" onclick={save} disabled={saving || !formName || !formModel}>
					{saving ? 'Saving...' : editing ? 'Update' : 'Create'}
				</button>
			</div>
		</div>
	</div>
{/if}

{#if confirmDelete}
	<div class="modal-backdrop" onclick={() => (confirmDelete = null)} role="presentation">
		<div class="confirm-dialog" onclick={(e) => e.stopPropagation()} role="dialog">
			<div class="modal-header"><h3 class="modal-title">Delete Agent</h3></div>
			<div class="modal-body">
				<p class="text-sm">Are you sure you want to delete <strong>{confirmDelete.name}</strong>?</p>
			</div>
			<div class="modal-footer">
				<button class="btn-secondary" onclick={() => (confirmDelete = null)}>Cancel</button>
				<button class="btn-destructive" onclick={() => confirmDelete && deleteConfig(confirmDelete)}>Delete</button>
			</div>
		</div>
	</div>
{/if}
