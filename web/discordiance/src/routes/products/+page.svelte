<script lang="ts">
	import { rpcClient } from '$lib/api/rpc-client';
	import { toast } from 'svelte-sonner';
	import { formatDate } from '$lib/utils';
	import type { Product } from '$lib/proto/discordiance/v1/types_pb';
	import { Plus, Pencil, Trash2, X, Package, ExternalLink, KeyRound } from '@lucide/svelte';

	let products = $state<Product[]>([]);
	let loading = $state(true);
	let showModal = $state(false);
	let editingProduct = $state<Product | null>(null);
	let confirmDelete = $state<Product | null>(null);
	let saving = $state(false);

	let formName = $state('');
	let formDescription = $state('');
	let formRepoUrl = $state('');
	let formGithubToken = $state('');
	let formEnabled = $state(true);

	async function load() {
		loading = true;
		try {
			const res = await rpcClient.product.listProducts({});
			products = res.products;
		} catch { /* handled */ } finally {
			loading = false;
		}
	}

	$effect(() => { load(); });

	function openCreate() {
		editingProduct = null;
		formName = '';
		formDescription = '';
		formRepoUrl = '';
		formGithubToken = '';
		formEnabled = true;
		showModal = true;
	}

	function openEdit(p: Product) {
		editingProduct = p;
		formName = p.name;
		formDescription = p.description;
		formRepoUrl = p.repoUrl;
		formGithubToken = '';
		formEnabled = p.enabled;
		showModal = true;
	}

	async function save() {
		if (!formName) return;
		saving = true;
		try {
			const data = {
				name: formName,
				description: formDescription,
				repoUrl: formRepoUrl,
				githubToken: formGithubToken,
				enabled: formEnabled
			};
			if (editingProduct) {
				await rpcClient.product.updateProduct({ id: editingProduct.id, ...data });
				toast.success('Product updated');
			} else {
				await rpcClient.product.createProduct(data);
				toast.success('Product created');
			}
			showModal = false;
			load();
		} catch { /* handled */ } finally {
			saving = false;
		}
	}

	async function deleteProduct(p: Product) {
		try {
			await rpcClient.product.deleteProduct({ id: p.id });
			toast.success('Product deleted');
			confirmDelete = null;
			load();
		} catch { /* handled */ }
	}
</script>

<div class="page-header">
	<div>
		<h1 class="page-title">Products</h1>
		<p class="page-subtitle">Manage monitored products and their configurations</p>
	</div>
	<button class="btn-primary" onclick={openCreate}>
		<Plus class="w-4 h-4" />
		Create Product
	</button>
</div>

{#if loading}
	<div class="card">
		<div class="p-8">
			{#each Array(3) as _}
				<div class="skeleton h-10 w-full mb-2"></div>
			{/each}
		</div>
	</div>
{:else if products.length === 0}
	<div class="card">
		<div class="empty-state">
			<Package class="w-10 h-10 empty-state-icon" />
			<p class="empty-state-title">No products</p>
			<p class="empty-state-text">Products represent the software you're monitoring for community feedback</p>
			<button class="btn-primary btn-sm" onclick={openCreate}>Create your first product</button>
		</div>
	</div>
{:else}
	<div class="card overflow-hidden">
		<div class="overflow-x-auto">
			<table class="data-table">
				<thead>
					<tr>
						<th>Name</th>
						<th>Description</th>
						<th>Repository</th>
						<th>GitHub Token</th>
						<th>Status</th>
						<th>Created</th>
						<th class="text-right">Actions</th>
					</tr>
				</thead>
				<tbody>
					{#each products as product}
						<tr>
							<td>
								<a href="/products/{product.id}" class="font-medium text-foreground hover:text-primary transition-colors">
									{product.name}
								</a>
							</td>
							<td class="text-muted-foreground max-w-[200px] truncate">{product.description || '—'}</td>
							<td>
								{#if product.repoUrl}
									<span class="text-xs text-muted-foreground flex items-center gap-1">
										<ExternalLink class="w-3 h-3" />
										<span class="truncate max-w-[150px]">{product.repoUrl}</span>
									</span>
								{:else}
									<span class="text-xs text-muted-foreground">—</span>
								{/if}
							</td>
							<td>
								{#if product.hasGithubToken}
									<span class="badge badge-success"><KeyRound class="w-3 h-3" /> Configured</span>
								{:else}
									<span class="text-xs text-muted-foreground">Not set</span>
								{/if}
							</td>
							<td>
								{#if product.enabled}
									<span class="badge badge-success">Enabled</span>
								{:else}
									<span class="badge badge-muted">Disabled</span>
								{/if}
							</td>
							<td class="text-xs text-muted-foreground whitespace-nowrap">{formatDate(product.createdAt)}</td>
							<td>
								<div class="flex items-center justify-end gap-1">
									<button class="btn-icon-sm" title="Edit" onclick={() => openEdit(product)}>
										<Pencil class="w-3 h-3" />
									</button>
									<button class="btn-icon-sm text-destructive" title="Delete" onclick={() => (confirmDelete = product)}>
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
				<h3 class="modal-title">{editingProduct ? 'Edit Product' : 'Create Product'}</h3>
				<button class="btn-icon-sm" onclick={() => (showModal = false)}><X class="w-4 h-4" /></button>
			</div>
			<div class="modal-body">
				<div class="form-group">
					<label class="form-label" for="prodName">Name</label>
					<input id="prodName" class="form-input" bind:value={formName} placeholder="My Product" />
				</div>
				<div class="form-group">
					<label class="form-label" for="prodDesc">Description</label>
					<textarea id="prodDesc" class="form-textarea" bind:value={formDescription} placeholder="What does this product do?"></textarea>
				</div>
				<div class="form-group">
					<label class="form-label" for="prodRepo">Repository URL</label>
					<input id="prodRepo" class="form-input" bind:value={formRepoUrl} placeholder="https://github.com/org/repo" />
				</div>
				<div class="form-group">
					<label class="form-label" for="prodToken">GitHub Token</label>
					<input id="prodToken" class="form-input" type="password" bind:value={formGithubToken} placeholder={editingProduct?.hasGithubToken ? '••••••••' : 'ghp_...'} />
					<p class="form-hint">{editingProduct?.hasGithubToken ? 'Leave blank to keep existing token' : 'Personal access token for repository access'}</p>
				</div>
				<div class="flex items-center gap-3">
					<label class="form-label mb-0">Enabled</label>
					<button type="button" class="form-toggle" class:active={formEnabled} onclick={() => (formEnabled = !formEnabled)}></button>
				</div>
			</div>
			<div class="modal-footer">
				<button class="btn-secondary" onclick={() => (showModal = false)}>Cancel</button>
				<button class="btn-primary" onclick={save} disabled={saving || !formName}>
					{saving ? 'Saving...' : editingProduct ? 'Update' : 'Create'}
				</button>
			</div>
		</div>
	</div>
{/if}

{#if confirmDelete}
	<div class="modal-backdrop" onclick={() => (confirmDelete = null)} role="presentation">
		<div class="confirm-dialog" onclick={(e) => e.stopPropagation()} role="dialog">
			<div class="modal-header">
				<h3 class="modal-title">Delete Product</h3>
			</div>
			<div class="modal-body">
				<p class="text-sm">Are you sure you want to delete <strong>{confirmDelete.name}</strong>? This will also delete all associated configs and data.</p>
			</div>
			<div class="modal-footer">
				<button class="btn-secondary" onclick={() => (confirmDelete = null)}>Cancel</button>
				<button class="btn-destructive" onclick={() => confirmDelete && deleteProduct(confirmDelete)}>Delete</button>
			</div>
		</div>
	</div>
{/if}
