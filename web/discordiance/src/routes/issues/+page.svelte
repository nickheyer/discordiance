<script lang="ts">
	import { onMount } from 'svelte';
	import { rpcClient } from '$lib/api/rpc-client';
	import type { Issue } from '$lib/proto/discordiance/v1/types_pb';
	import Badge from '$lib/components/badge.svelte';
	import { ChevronLeft, ChevronRight } from '@lucide/svelte';

	let issues = $state<Issue[]>([]);
	let totalCount = $state(0);
	let pg = $state(1);
	const pgSize = 25;

	onMount(() => load());

	async function load() {
		const resp = await rpcClient.issue.listIssues({ pageSize: pgSize, page: pg });
		issues = resp.issues;
		totalCount = resp.totalCount;
	}

	const totalPages = $derived(Math.max(1, Math.ceil(totalCount / pgSize)));

	function formatDate(ts: { seconds: bigint } | undefined): string {
		if (!ts) return '';
		return new Date(Number(ts.seconds) * 1000).toLocaleDateString();
	}
</script>

<div class="space-y-6">
	<div>
		<h1 class="text-xl font-semibold tracking-tight">Issues</h1>
		<p class="text-sm text-muted-foreground mt-0.5">Issues detected by the LLM agent from community messages</p>
	</div>

	{#if issues.length === 0}
		<div class="card card-body py-16 text-center">
			<p class="text-muted-foreground">No issues detected yet. Issues appear here once the pipeline processes messages.</p>
		</div>
	{:else}
		<div class="card overflow-hidden">
			<table class="data-table">
				<thead>
					<tr>
						<th>Title</th>
						<th class="hidden sm:table-cell">Product</th>
						<th>Severity</th>
						<th>Category</th>
						<th>Status</th>
						<th class="hidden md:table-cell">Date</th>
					</tr>
				</thead>
				<tbody>
					{#each issues as issue}
						<tr class="cursor-pointer" onclick={() => window.location.href = `/issues/${issue.id}`}>
							<td>
								<a href="/issues/{issue.id}" class="font-medium hover:text-primary">{issue.title}</a>
							</td>
							<td class="text-muted-foreground hidden sm:table-cell">{issue.productName}</td>
							<td><Badge value={issue.severity} type="severity" /></td>
							<td><Badge value={issue.category} /></td>
							<td><Badge value={issue.status} type="status" /></td>
							<td class="text-xs text-muted-foreground hidden md:table-cell">{formatDate(issue.createdAt)}</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>

		{#if totalPages > 1}
			<div class="flex items-center justify-between text-sm">
				<span class="text-muted-foreground">{totalCount} issues</span>
				<div class="flex items-center gap-1">
					<button class="btn-icon" disabled={pg <= 1} onclick={() => { pg--; load(); }}>
						<ChevronLeft class="h-4 w-4" />
					</button>
					<span class="px-2 text-muted-foreground">{pg} / {totalPages}</span>
					<button class="btn-icon" disabled={pg >= totalPages} onclick={() => { pg++; load(); }}>
						<ChevronRight class="h-4 w-4" />
					</button>
				</div>
			</div>
		{/if}
	{/if}
</div>
