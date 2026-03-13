<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { rpcClient } from '$lib/api/rpc-client';
	import type { Product } from '$lib/proto/discordiance/v1/types_pb';
	import { toast } from 'svelte-sonner';
	import Badge from '$lib/components/badge.svelte';
	import { Plus, Trash2 } from '@lucide/svelte';

	let products = $state<Product[]>([]);
	let showCreate = $state(false);
	let newName = $state('');
	let creating = $state(false);

	onMount(loadProducts);

	async function loadProducts() {
		const resp = await rpcClient.product.listProducts({});
		products = resp.products;
	}

	async function createProduct() {
		if (!newName.trim()) return;
		creating = true;
		try {
			const resp = await rpcClient.product.createProduct({ name: newName.trim(), enabled: true });
			toast.success('Product created');
			goto(`/products/${resp.product?.id}`);
		} finally {
			creating = false;
		}
	}

	async function deleteProduct(e: Event, id: bigint, name: string) {
		e.preventDefault();
		e.stopPropagation();
		if (!confirm(`Delete "${name}" and all its configurations?`)) return;
		await rpcClient.product.deleteProduct({ id });
		toast.success('Product deleted');
		await loadProducts();
	}
</script>

<div class="space-y-6">
	<div class="flex items-center justify-between">
		<div>
			<h1 class="text-xl font-semibold tracking-tight">Products</h1>
			<p class="text-sm text-muted-foreground mt-0.5">Each product monitors a community and detects issues</p>
		</div>
		<button class="btn-primary btn-sm" onclick={() => (showCreate = !showCreate)}>
			<Plus class="h-3.5 w-3.5" />
			New Product
		</button>
	</div>

	{#if showCreate}
		<form
			class="card card-body flex items-end gap-3"
			onsubmit={(e) => { e.preventDefault(); createProduct(); }}
		>
			<div class="flex-1">
				<label for="new-name" class="label">Product Name</label>
				<input id="new-name" class="input" bind:value={newName} placeholder="e.g. My Discord Server" autofocus />
			</div>
			<button type="submit" class="btn-primary" disabled={creating || !newName.trim()}>
				{creating ? 'Creating...' : 'Create'}
			</button>
			<button type="button" class="btn-secondary" onclick={() => { showCreate = false; newName = ''; }}>
				Cancel
			</button>
		</form>
	{/if}

	{#if products.length === 0 && !showCreate}
		<div class="card card-body py-16 text-center">
			<p class="text-muted-foreground mb-3">No products yet</p>
			<button class="btn-primary btn-sm mx-auto" onclick={() => (showCreate = true)}>
				<Plus class="h-3.5 w-3.5" /> Create your first product
			</button>
		</div>
	{:else if products.length > 0}
		<div class="card overflow-hidden">
			<table class="data-table">
				<thead>
					<tr>
						<th>Name</th>
						<th>Status</th>
						<th>Platforms</th>
						<th>Agent</th>
						<th>Reporters</th>
						<th class="w-12"></th>
					</tr>
				</thead>
				<tbody>
					{#each products as product}
						<tr class="cursor-pointer" onclick={() => goto(`/products/${product.id}`)}>
							<td class="font-medium">{product.name}</td>
							<td><Badge value={product.enabled ? 'enabled' : 'disabled'} type="status" /></td>
							<td class="text-muted-foreground">{product.platformConfigs.length}</td>
							<td>
								{#if product.agentConfig}
									<Badge value={product.agentConfig.model || 'configured'} />
								{:else}
									<span class="text-xs text-muted-foreground">none</span>
								{/if}
							</td>
							<td class="text-muted-foreground">{product.reporterConfigs.length}</td>
							<td>
								<button
									class="btn-icon text-muted-foreground hover:text-destructive"
									onclick={(e) => deleteProduct(e, product.id, product.name)}
								>
									<Trash2 class="h-3.5 w-3.5" />
								</button>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}
</div>
