<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { rpcClient } from '$lib/api/rpc-client';
	import type { Issue } from '$lib/proto/discordiance/v1/types_pb';
	import { toast } from 'svelte-sonner';
	import Badge from '$lib/components/badge.svelte';
	import { ChevronLeft, ExternalLink } from '@lucide/svelte';

	const issueId = $derived(BigInt(page.params.id));
	let issue = $state<Issue | null>(null);
	let newStatus = $state('');

	onMount(loadIssue);

	async function loadIssue() {
		const resp = await rpcClient.issue.getIssue({ id: issueId });
		issue = resp.issue!;
		newStatus = issue.status;
	}

	async function updateStatus() {
		if (newStatus === issue?.status) return;
		await rpcClient.issue.updateIssueStatus({ id: issueId, status: newStatus });
		toast.success('Status updated');
		await loadIssue();
	}

	function formatDate(ts: { seconds: bigint } | undefined): string {
		if (!ts) return '';
		return new Date(Number(ts.seconds) * 1000).toLocaleString();
	}
</script>

{#if !issue}
	<div class="flex items-center justify-center py-20">
		<div class="h-5 w-5 border-2 border-primary/30 border-t-primary rounded-full animate-spin"></div>
	</div>
{:else}
	<div class="space-y-6">
		<a href="/issues" class="inline-flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground">
			<ChevronLeft class="h-3 w-3" /> Issues
		</a>

		<div>
			<h1 class="text-xl font-semibold tracking-tight">{issue.title}</h1>
			<div class="flex items-center gap-2 mt-2">
				<Badge value={issue.severity} type="severity" />
				<Badge value={issue.category} />
				<Badge value={issue.status} type="status" />
				<span class="text-xs text-muted-foreground ml-2">{issue.productName}</span>
			</div>
		</div>

		<div class="grid gap-6 lg:grid-cols-3">
			<!-- Main content -->
			<div class="lg:col-span-2 space-y-6">
				<div class="card card-body">
					<h2 class="text-sm font-semibold mb-2">Description</h2>
					<p class="text-sm whitespace-pre-wrap leading-relaxed">{issue.description}</p>
				</div>

				{#if issue.reports.length > 0}
					<div class="card">
						<div class="card-header">
							<h2 class="text-sm font-semibold">Reports ({issue.reports.length})</h2>
						</div>
						<table class="data-table">
							<thead>
								<tr>
									<th>Reporter</th>
									<th>External ID</th>
									<th>Status</th>
									<th></th>
								</tr>
							</thead>
							<tbody>
								{#each issue.reports as report}
									<tr>
										<td><Badge value={report.reporterType} /></td>
										<td class="font-mono text-xs">{report.externalId}</td>
										<td><Badge value={report.status} type="status" /></td>
										<td>
											{#if report.externalUrl}
												<a
													href={report.externalUrl}
													target="_blank"
													rel="noopener noreferrer"
													class="inline-flex items-center gap-1 text-xs text-primary hover:underline"
												>
													<ExternalLink class="h-3 w-3" /> Open
												</a>
											{/if}
										</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
				{/if}
			</div>

			<!-- Sidebar -->
			<div class="space-y-4">
				<div class="card card-body space-y-3">
					<h2 class="text-sm font-semibold">Update Status</h2>
					<select class="input" bind:value={newStatus}>
						<option value="open">Open</option>
						<option value="acknowledged">Acknowledged</option>
						<option value="resolved">Resolved</option>
						<option value="closed">Closed</option>
					</select>
					<button class="btn-primary btn-sm w-full" onclick={updateStatus} disabled={newStatus === issue.status}>
						Update Status
					</button>
				</div>

				<div class="card card-body space-y-2 text-sm">
					<h2 class="text-sm font-semibold mb-1">Details</h2>
					<div class="flex justify-between">
						<span class="text-muted-foreground">Created</span>
						<span class="text-xs">{formatDate(issue.createdAt)}</span>
					</div>
					<div class="flex justify-between">
						<span class="text-muted-foreground">Updated</span>
						<span class="text-xs">{formatDate(issue.updatedAt)}</span>
					</div>
					<div class="flex justify-between">
						<span class="text-muted-foreground">Fingerprint</span>
						<span class="text-xs font-mono truncate ml-2 max-w-[120px]" title={issue.fingerprint}>{issue.fingerprint}</span>
					</div>
					{#if issue.sourceMsgIds}
						<div>
							<span class="text-muted-foreground">Source IDs</span>
							<p class="text-xs font-mono mt-1 break-all">{issue.sourceMsgIds}</p>
						</div>
					{/if}
				</div>
			</div>
		</div>
	</div>
{/if}
