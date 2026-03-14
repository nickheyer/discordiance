<script lang="ts">
	import { rpcClient } from '$lib/api/rpc-client';
	import { toast } from 'svelte-sonner';
	import { formatDate } from '$lib/utils';
	import type { Reporter } from '$lib/proto/discordiance/v1/types_pb';
	import { Plus, Pencil, Trash2, X, Send } from '@lucide/svelte';

	let configs = $state<Reporter[]>([]);
	let availableReporters = $state<string[]>([]);
	let loading = $state(true);
	let showModal = $state(false);
	let editing = $state<Reporter | null>(null);
	let confirmDelete = $state<Reporter | null>(null);
	let saving = $state(false);

	let formName = $state('');
	let formType = $state('');
	let formEnabled = $state(true);

	// GitHub settings
	let githubToken = $state('');
	let githubRepo = $state('');
	let githubLabels = $state('');
	let githubAutoFile = $state(true);

	async function load() {
		loading = true;
		try {
			const [configRes, reporterRes] = await Promise.all([
				rpcClient.reporter.listReporters({}),
				rpcClient.reporter.listAvailableReporters({})
			]);
			configs = configRes.reporters;
			availableReporters = reporterRes.reporters;
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
		formType = availableReporters[0] || '';
		formEnabled = true;
		githubToken = '';
		githubRepo = '';
		githubLabels = '';
		githubAutoFile = true;
		showModal = true;
	}

	function openEdit(c: Reporter) {
		editing = c;
		formName = c.name;
		formType = c.type;
		formEnabled = c.enabled;
		if (c.settings.case === 'githubSettings') {
			githubToken = c.settings.value.token;
			githubRepo = c.settings.value.repo;
			githubLabels = c.settings.value.labels;
			githubAutoFile = c.settings.value.autoFile;
		} else {
			githubToken = '';
			githubRepo = '';
			githubLabels = '';
			githubAutoFile = true;
		}
		showModal = true;
	}

	function buildSettings(): Reporter['settings'] {
		switch (formType) {
			case 'github':
				return {
					case: 'githubSettings',
					value: { token: githubToken, repo: githubRepo, labels: githubLabels, autoFile: githubAutoFile }
				} as Reporter['settings'];
			default:
				return { case: undefined, value: undefined };
		}
	}

	async function save() {
		if (!formName || !formType) return;
		saving = true;
		try {
			const settings = buildSettings();
			if (editing) {
				await rpcClient.reporter.updateReporter({
					id: editing.id,
					name: formName,
					type: formType,
					enabled: formEnabled,
					settings
				});
				toast.success('Reporter updated');
			} else {
				await rpcClient.reporter.createReporter({
					name: formName,
					type: formType,
					enabled: formEnabled,
					settings
				});
				toast.success('Reporter created');
			}
			showModal = false;
			load();
		} catch {
			// handled
		} finally {
			saving = false;
		}
	}

	async function deleteConfig(c: Reporter) {
		try {
			await rpcClient.reporter.deleteReporter({ id: c.id });
			toast.success('Reporter deleted');
			confirmDelete = null;
			load();
		} catch {
			// handled
		}
	}

	function settingsSummary(c: Reporter): string {
		if (c.settings.case === 'githubSettings') {
			const parts: string[] = [];
			if (c.settings.value.repo) parts.push(c.settings.value.repo);
			if (c.settings.value.token) parts.push('token set');
			if (c.settings.value.autoFile) parts.push('auto-file');
			return parts.join(', ') || 'no settings';
		}
		return 'unknown type';
	}
</script>

<div class="page-header">
	<div>
		<h1 class="page-title">Reporters</h1>
		<p class="page-subtitle">Manage destinations for filing insights as external tickets</p>
	</div>
	<button class="btn-primary" onclick={openCreate}>
		<Plus class="w-4 h-4" />
		Create Reporter
	</button>
</div>

{#if loading}
	<div class="card"><div class="p-8">{#each Array(3) as _}<div class="skeleton h-10 w-full mb-2"></div>{/each}</div></div>
{:else if configs.length === 0}
	<div class="card">
		<div class="empty-state">
			<Send class="w-10 h-10 empty-state-icon" />
			<p class="empty-state-title">No reporters</p>
			<p class="empty-state-text">Reporters define where insights are filed (GitHub Issues, Jira, etc.)</p>
			<button class="btn-primary btn-sm" onclick={openCreate}>Create reporter</button>
		</div>
	</div>
{:else}
	<div class="card overflow-hidden">
		<div class="overflow-x-auto">
			<table class="data-table">
				<thead>
					<tr>
						<th>Name</th>
						<th>Type</th>
						<th>Status</th>
						<th>Settings</th>
						<th>Created</th>
						<th class="text-right">Actions</th>
					</tr>
				</thead>
				<tbody>
					{#each configs as config}
						<tr>
							<td class="font-medium">{config.name}</td>
							<td><span class="badge badge-info">{config.type}</span></td>
							<td>
								{#if config.enabled}
									<span class="badge badge-success">Enabled</span>
								{:else}
									<span class="badge badge-muted">Disabled</span>
								{/if}
							</td>
							<td class="text-xs text-muted-foreground">{settingsSummary(config)}</td>
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
				<h3 class="modal-title">{editing ? 'Edit Reporter' : 'Create Reporter'}</h3>
				<button class="btn-icon-sm" onclick={() => (showModal = false)}><X class="w-4 h-4" /></button>
			</div>
			<div class="modal-body">
				<div class="form-group">
					<label class="form-label" for="rcName">Name</label>
					<input id="rcName" class="form-input" bind:value={formName} placeholder="GitHub Issues Reporter" />
				</div>
				<div class="form-group">
					<label class="form-label" for="rcType">Type</label>
					<select id="rcType" class="form-select" bind:value={formType}>
						{#each availableReporters as r}
							<option value={r}>{r}</option>
						{/each}
						{#if availableReporters.length === 0}
							<option value="">No reporters available</option>
						{/if}
					</select>
				</div>
				<div class="flex items-center gap-3">
					<label class="form-label mb-0">Enabled</label>
					<button type="button" class="form-toggle" class:active={formEnabled} onclick={() => (formEnabled = !formEnabled)}></button>
				</div>

				{#if formType === 'github'}
					<div class="form-group">
						<label class="form-label" for="ghToken">GitHub Token <span class="text-destructive">*</span></label>
						<input id="ghToken" class="form-input" type="password" bind:value={githubToken} placeholder="ghp_..." />
						<p class="text-xs text-muted-foreground mt-1">Personal access token with repo scope</p>
					</div>
					<div class="form-group">
						<label class="form-label" for="ghRepo">Repository <span class="text-destructive">*</span></label>
						<input id="ghRepo" class="form-input" bind:value={githubRepo} placeholder="owner/repo" />
						<p class="text-xs text-muted-foreground mt-1">Target repository in owner/repo format</p>
					</div>
					<div class="form-group">
						<label class="form-label" for="ghLabels">Labels</label>
						<input id="ghLabels" class="form-input" bind:value={githubLabels} placeholder="bug,discordiance" />
						<p class="text-xs text-muted-foreground mt-1">Comma-separated labels to apply to created issues</p>
					</div>
					<div class="flex items-center gap-3">
						<label class="form-label mb-0">Auto-file Issues</label>
						<button type="button" class="form-toggle" class:active={githubAutoFile} onclick={() => (githubAutoFile = !githubAutoFile)}></button>
						<p class="text-xs text-muted-foreground">Automatically file issues for new insights</p>
					</div>
				{/if}
			</div>
			<div class="modal-footer">
				<button class="btn-secondary" onclick={() => (showModal = false)}>Cancel</button>
				<button class="btn-primary" onclick={save} disabled={saving || !formName || !formType}>
					{saving ? 'Saving...' : editing ? 'Update' : 'Create'}
				</button>
			</div>
		</div>
	</div>
{/if}

{#if confirmDelete}
	<div class="modal-backdrop" onclick={() => (confirmDelete = null)} role="presentation">
		<div class="confirm-dialog" onclick={(e) => e.stopPropagation()} role="dialog">
			<div class="modal-header"><h3 class="modal-title">Delete Reporter</h3></div>
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
