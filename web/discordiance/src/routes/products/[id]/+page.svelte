<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { rpcClient } from '$lib/api/rpc-client';
	import type { Product, ProductStatus } from '$lib/proto/discordiance/v1/types_pb';
	import { toast } from 'svelte-sonner';
	import PlatformConfigSection from '$lib/components/platform-config-section.svelte';
	import AgentConfigSection from '$lib/components/agent-config-section.svelte';
	import ReporterConfigSection from '$lib/components/reporter-config-section.svelte';
	import Badge from '$lib/components/badge.svelte';
	import {
		Save,
		Trash2,
		RefreshCw,
		ChevronLeft,
		Download,
		Play,
		Square
	} from '@lucide/svelte';

	const productId = $derived(BigInt(page.params.id || ''));
	let product = $state<Product | null>(null);
	let pipelineStatus = $state<ProductStatus | null>(null);
	let name = $state('');
	let enabled = $state(true);
	let saving = $state(false);
	let starting = $state(false);
	let stopping = $state(false);
	let restarting = $state(false);
	let backfilling = $state(false);

	const isRunning = $derived(pipelineStatus?.running ?? false);

	onMount(() => {
		loadProduct();
		loadStatus();
	});

	async function loadProduct() {
		const resp = await rpcClient.product.getProduct({ id: productId });
		product = resp.product!;
		name = product.name;
		enabled = product.enabled;
	}

	async function loadStatus() {
		const resp = await rpcClient.pipeline.getPipelineStatus({});
		pipelineStatus = resp.statuses.find(s => s.productId === productId) ?? null;
	}

	async function saveProduct() {
		saving = true;
		try {
			await rpcClient.product.updateProduct({ id: productId, name, enabled });
			toast.success('Product updated');
			await loadProduct();
		} finally {
			saving = false;
		}
	}

	async function deleteProduct() {
		if (!confirm(`Delete "${product?.name}" and all its configurations? This cannot be undone.`)) return;
		await rpcClient.product.deleteProduct({ id: productId });
		toast.success('Product deleted');
		goto('/products');
	}

	async function startPipeline() {
		starting = true;
		try {
			await rpcClient.pipeline.startPipeline({ productId });
			toast.success('Pipeline started');
			await loadStatus();
		} catch {
			// error already shown by interceptor
		} finally {
			starting = false;
		}
	}

	async function stopPipeline() {
		stopping = true;
		try {
			await rpcClient.pipeline.stopPipeline({ productId });
			toast.success('Pipeline stopped');
			await loadStatus();
		} catch {
			// error already shown by interceptor
		} finally {
			stopping = false;
		}
	}

	async function restartPipeline() {
		restarting = true;
		try {
			await rpcClient.pipeline.restartPipeline({ productId });
			toast.success('Pipeline restarted');
			await loadStatus();
		} catch {
			// error already shown by interceptor
		} finally {
			restarting = false;
		}
	}

	async function backfillMessages() {
		backfilling = true;
		try {
			await rpcClient.pipeline.backfillPipeline({ productId });
			toast.success('Backfill started');
		} catch {
			// error already shown by interceptor
		} finally {
			backfilling = false;
		}
	}
</script>

{#if !product}
	<div class="flex items-center justify-center py-20">
		<div class="h-5 w-5 border-2 border-primary/30 border-t-primary rounded-full animate-spin"></div>
	</div>
{:else}
	<div class="space-y-8">
		<!-- Breadcrumb + actions -->
		<div>
			<a href="/products" class="inline-flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground mb-3">
				<ChevronLeft class="h-3 w-3" /> Products
			</a>
			<div class="flex items-center justify-between">
				<div class="flex items-center gap-3">
					<h1 class="text-xl font-semibold tracking-tight">{product.name}</h1>
					<Badge value={product.enabled ? 'enabled' : 'disabled'} type="status" />
					{#if isRunning}
						<Badge value="running" type="status" />
					{:else}
						<Badge value="stopped" type="status" />
					{/if}
				</div>
				<div class="flex items-center gap-2">
					{#if isRunning}
						<button class="btn-secondary btn-sm" onclick={backfillMessages} disabled={backfilling}>
							<Download class="h-3.5 w-3.5" />
							{backfilling ? 'Starting...' : 'Backfill'}
						</button>
						<button class="btn-secondary btn-sm" onclick={restartPipeline} disabled={restarting}>
							<RefreshCw class="h-3.5 w-3.5 {restarting ? 'animate-spin' : ''}" />
							{restarting ? 'Restarting...' : 'Restart'}
						</button>
						<button class="btn-secondary btn-sm" onclick={stopPipeline} disabled={stopping}>
							<Square class="h-3.5 w-3.5" />
							{stopping ? 'Stopping...' : 'Stop'}
						</button>
					{:else}
						<button class="btn-primary btn-sm" onclick={startPipeline} disabled={starting}>
							<Play class="h-3.5 w-3.5" />
							{starting ? 'Starting...' : 'Start Pipeline'}
						</button>
					{/if}
					<button
						class="btn-ghost btn-sm text-destructive hover:bg-destructive/10"
						onclick={deleteProduct}
					>
						<Trash2 class="h-3.5 w-3.5" />
					</button>
				</div>
			</div>
		</div>

		<!-- Product settings -->
		<section class="card">
			<div class="card-header">
				<h2 class="text-sm font-semibold">General</h2>
			</div>
			<div class="card-body space-y-4">
				<div class="grid gap-4 sm:grid-cols-2">
					<div>
						<label for="prod-name" class="label">Name</label>
						<input id="prod-name" class="input" bind:value={name} />
					</div>
					<div>
						<label class="label">Status</label>
						<label class="flex items-center gap-2 mt-2 cursor-pointer">
							<input type="checkbox" bind:checked={enabled} class="rounded accent-primary" />
							<span class="text-sm">{enabled ? 'Enabled — pipeline can run' : 'Disabled — pipeline cannot start'}</span>
						</label>
					</div>
				</div>
				<button class="btn-primary btn-sm" onclick={saveProduct} disabled={saving}>
					<Save class="h-3.5 w-3.5" />
					{saving ? 'Saving...' : 'Save Changes'}
				</button>
			</div>
		</section>

		<!-- Platform Configs -->
		<PlatformConfigSection {productId} platformConfigs={product.platformConfigs} onchange={loadProduct} />

		<!-- Agent Config -->
		<AgentConfigSection {productId} agentConfig={product.agentConfig} onchange={loadProduct} />

		<!-- Reporter Configs -->
		<ReporterConfigSection {productId} reporterConfigs={product.reporterConfigs} onchange={loadProduct} />
	</div>
{/if}
