<script lang="ts">
	import { rpcClient } from '$lib/api/rpc-client';
	import type { PlatformConfig } from '$lib/proto/discordiance/v1/types_pb';
	import { toast } from 'svelte-sonner';
	import Badge from './badge.svelte';
	import { Plus, Pencil, Trash2, X, Save } from '@lucide/svelte';
	import { onMount } from 'svelte';

	let { productId, platformConfigs, onchange }: {
		productId: bigint;
		platformConfigs: PlatformConfig[];
		onchange: () => Promise<void>;
	} = $props();

	let availableTypes = $state<string[]>([]);
	let showCreate = $state(false);
	let editingId = $state<bigint | null>(null);

	let createType = $state('');
	let createEnabled = $state(true);
	let createSettings = $state<Record<string, string>>({});

	let editEnabled = $state(true);
	let editSettings = $state<Record<string, string>>({});

	const platformFields: Record<string, Array<{ key: string; label: string; required: boolean; type: string; hint: string }>> = {
		discord: [
			{ key: 'token', label: 'Bot Token', required: true, type: 'password', hint: 'Discord bot token from the Developer Portal' },
			{ key: 'channel_ids', label: 'Channel IDs', required: false, type: 'text', hint: 'Comma-separated Discord channel IDs to monitor. Leave empty for all.' }
		]
	};

	onMount(async () => {
		const resp = await rpcClient.platformConfig.listAvailablePlatforms({});
		availableTypes = resp.platforms;
	});

	function getFields(type: string) {
		return platformFields[type] || [];
	}

	function resetCreate() {
		createType = '';
		createEnabled = true;
		createSettings = {};
		showCreate = false;
	}

	function startEdit(pc: PlatformConfig) {
		editingId = pc.id;
		editEnabled = pc.enabled;
		editSettings = { ...pc.settings };
	}

	function cancelEdit() { editingId = null; }

	async function handleCreate() {
		if (!createType) return;
		await rpcClient.platformConfig.createPlatformConfig({
			productId, type: createType, enabled: createEnabled, settings: createSettings
		});
		toast.success(`${createType} platform added`);
		resetCreate();
		await onchange();
	}

	async function handleUpdate(id: bigint, type: string) {
		await rpcClient.platformConfig.updatePlatformConfig({
			id, type, enabled: editEnabled, settings: editSettings
		});
		toast.success('Platform config updated');
		editingId = null;
		await onchange();
	}

	async function handleDelete(id: bigint) {
		if (!confirm('Delete this platform configuration?')) return;
		await rpcClient.platformConfig.deletePlatformConfig({ id });
		toast.success('Platform config deleted');
		await onchange();
	}

	function onTypeChange() {
		createSettings = {};
		for (const f of getFields(createType)) createSettings[f.key] = '';
	}

</script>

<section class="card">
	<div class="card-header flex items-center justify-between">
		<div>
			<h2 class="text-sm font-semibold">Platform Configs</h2>
			<p class="text-xs text-muted-foreground mt-0.5">Sources that collect messages from your community</p>
		</div>
		{#if !showCreate}
			<button class="btn-ghost btn-sm text-primary" onclick={() => (showCreate = true)}>
				<Plus class="h-3.5 w-3.5" /> Add
			</button>
		{/if}
	</div>

	{#if showCreate}
		<div class="border-b border-border bg-muted/20 p-5 space-y-4">
			<div class="flex items-center justify-between">
				<h3 class="text-sm font-medium">New Platform</h3>
				<button class="btn-icon" onclick={resetCreate}><X class="h-4 w-4" /></button>
			</div>

			<div class="grid gap-4 sm:grid-cols-2">
				<div>
					<label for="pc-type" class="label">Platform Type</label>
					<select id="pc-type" class="input" bind:value={createType} onchange={onTypeChange}>
						<option value="">Choose a platform...</option>
						{#each availableTypes as t}
							<option value={t}>{t}</option>
						{/each}
					</select>
				</div>
				<div>
					<label class="label">Status</label>
					<label class="flex items-center gap-2 mt-2 cursor-pointer">
						<input type="checkbox" bind:checked={createEnabled} class="rounded accent-primary" />
						<span class="text-sm">Enabled</span>
					</label>
				</div>
			</div>

			{#if createType}
				{@const fields = getFields(createType)}
				{#if fields.length > 0}
					<div class="space-y-3">
						{#each fields as field}
							<div>
								<label for="pc-{field.key}" class="label">
									{field.label}
									{#if field.required}<span class="text-destructive">*</span>{/if}
								</label>
								<input id="pc-{field.key}" class="input" type={field.type}
									bind:value={createSettings[field.key]} placeholder={field.hint} />
								<p class="hint">{field.hint}</p>
							</div>
						{/each}
					</div>
				{:else}
					<p class="text-sm text-muted-foreground">No known fields for "{createType}".</p>
				{/if}
				<div class="flex gap-2 pt-2">
					<button class="btn-primary btn-sm" onclick={handleCreate}>
						<Plus class="h-3.5 w-3.5" /> Create Platform
					</button>
					<button class="btn-secondary btn-sm" onclick={resetCreate}>Cancel</button>
				</div>
			{/if}
		</div>
	{/if}

	{#if platformConfigs.length === 0 && !showCreate}
		<div class="card-body text-center py-10">
			<p class="text-sm text-muted-foreground mb-2">No platforms configured</p>
			<p class="text-xs text-muted-foreground">Add a platform to start collecting community messages</p>
		</div>
	{:else if platformConfigs.length > 0}
		<div class="divide-y divide-border/50">
			{#each platformConfigs as pc}
				{#if editingId === pc.id}
					{@const fields = getFields(pc.type)}
					<div class="p-5 bg-muted/20 space-y-4">
						<div class="flex items-center justify-between">
							<h3 class="text-sm font-medium">Editing: {pc.type}</h3>
							<button class="btn-icon" onclick={cancelEdit}><X class="h-4 w-4" /></button>
						</div>

						<label class="flex items-center gap-2 cursor-pointer">
							<input type="checkbox" bind:checked={editEnabled} class="rounded accent-primary" />
							<span class="text-sm">Enabled</span>
						</label>

						{#if fields.length > 0}
							<div class="space-y-3">
								{#each fields as field}
									<div>
										<label for="edit-{field.key}" class="label">
											{field.label}
											{#if field.required}<span class="text-destructive">*</span>{/if}
										</label>
										<input id="edit-{field.key}" class="input" type={field.type}
											bind:value={editSettings[field.key]} placeholder={field.hint} />
										<p class="hint">{field.hint}</p>
									</div>
								{/each}
							</div>
						{:else}
							<div class="space-y-2">
								{#each Object.entries(editSettings) as [key]}
									<div class="flex items-center gap-2">
										<span class="text-sm font-mono text-muted-foreground w-28 shrink-0">{key}</span>
										<input class="input" type={key.includes('token') || key.includes('key') || key.includes('secret') ? 'password' : 'text'} bind:value={editSettings[key]} />
									</div>
								{/each}
							</div>
						{/if}

						<div class="flex gap-2">
							<button class="btn-primary btn-sm" onclick={() => handleUpdate(pc.id, pc.type)}>
								<Save class="h-3.5 w-3.5" /> Save
							</button>
							<button class="btn-secondary btn-sm" onclick={cancelEdit}>Cancel</button>
						</div>
					</div>
				{:else}
					<div class="flex items-center justify-between px-5 py-3">
						<div class="flex items-center gap-3">
							<Badge value={pc.type} />
							<Badge value={pc.enabled ? 'enabled' : 'disabled'} type="status" />
							<span class="text-xs text-muted-foreground">{Object.keys(pc.settings).length} setting(s)</span>
						</div>
						<div class="flex items-center gap-1">
							<button class="btn-icon text-muted-foreground hover:text-foreground" onclick={() => startEdit(pc)}>
								<Pencil class="h-3.5 w-3.5" />
							</button>
							<button class="btn-icon text-muted-foreground hover:text-destructive" onclick={() => handleDelete(pc.id)}>
								<Trash2 class="h-3.5 w-3.5" />
							</button>
						</div>
					</div>
				{/if}
			{/each}
		</div>
	{/if}
</section>
