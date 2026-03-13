<script lang="ts">
	import { onMount } from 'svelte';
	import { rpcClient } from '$lib/api/rpc-client';
	import type { Message } from '$lib/proto/discordiance/v1/types_pb';
	import Badge from '$lib/components/badge.svelte';
	import { ChevronLeft, ChevronRight } from '@lucide/svelte';

	let messages = $state<Message[]>([]);
	let totalCount = $state(0);
	let pg = $state(1);
	const pgSize = 50;

	onMount(() => load());

	async function load() {
		const resp = await rpcClient.message.listMessages({ pageSize: pgSize, page: pg });
		messages = resp.messages;
		totalCount = resp.totalCount;
	}

	const totalPages = $derived(Math.max(1, Math.ceil(totalCount / pgSize)));

	function formatTime(ts: { seconds: bigint } | undefined): string {
		if (!ts) return '';
		return new Date(Number(ts.seconds) * 1000).toLocaleString();
	}

	function truncate(s: string, n = 100): string {
		return s.length <= n ? s : s.slice(0, n) + '...';
	}
</script>

<div class="space-y-6">
	<div>
		<h1 class="text-xl font-semibold tracking-tight">Messages</h1>
		<p class="text-sm text-muted-foreground mt-0.5">Raw messages collected from platform sources</p>
	</div>

	{#if messages.length === 0}
		<div class="card card-body py-16 text-center">
			<p class="text-muted-foreground">No messages received yet. Messages appear here once a platform is connected and running.</p>
		</div>
	{:else}
		<div class="card overflow-hidden">
			<table class="data-table">
				<thead>
					<tr>
						<th>Platform</th>
						<th class="hidden sm:table-cell">Author</th>
						<th>Content</th>
						<th class="hidden md:table-cell">Time</th>
						<th>Processed</th>
					</tr>
				</thead>
				<tbody>
					{#each messages as msg}
						<tr>
							<td><Badge value={msg.platformType} /></td>
							<td class="hidden sm:table-cell">
								<span class="text-sm">{msg.authorName}</span>
								{#if msg.channelId}
									<span class="text-xs text-muted-foreground block font-mono">#{msg.channelId}</span>
								{/if}
							</td>
							<td class="max-w-md">
								<p class="text-sm truncate" title={msg.content}>{truncate(msg.content)}</p>
							</td>
							<td class="text-xs text-muted-foreground hidden md:table-cell whitespace-nowrap">{formatTime(msg.timestamp)}</td>
							<td>
								<Badge value={msg.processed ? 'processed' : 'pending'} type="status" />
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>

		{#if totalPages > 1}
			<div class="flex items-center justify-between text-sm">
				<span class="text-muted-foreground">{totalCount} messages</span>
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
