<script lang="ts">
	import { page } from '$app/state';
	import { rpcClient } from '$lib/api/rpc-client';
	import { toast } from 'svelte-sonner';
	import { formatDate, formatRelative, severityColor, statusColor } from '$lib/utils';
	import type { Insight, Reporter } from '$lib/proto/discordiance/v1/types_pb';
	import {
		ChevronRight,
		ExternalLink,
		FileText,
		Send,
		X
	} from '@lucide/svelte';

	let insight = $state<Insight | null>(null);
	let reporters = $state<Reporter[]>([]);
	let loading = $state(true);
	let showFileModal = $state(false);
	let selectedReporterId = $state('');
	let filing = $state(false);

	const insightId = $derived(BigInt(page.params.id!));

	async function load() {
		loading = true;
		try {
			const [insightRes, reporterRes] = await Promise.all([
				rpcClient.insight.getInsight({ id: insightId }),
				rpcClient.reporter.listReporters({})
			]);
			insight = insightRes.insight ?? null;
			reporters = reporterRes.reporters;
		} catch {
			// handled
		} finally {
			loading = false;
		}
	}

	$effect(() => { load(); });

	async function updateStatus(newStatus: string) {
		try {
			const res = await rpcClient.insight.updateInsightStatus({ id: insightId, status: newStatus });
			insight = res.insight ?? insight;
			toast.success(`Status updated to ${newStatus}`);
		} catch {
			// handled
		}
	}

	async function fileToReporter() {
		if (!selectedReporterId) return;
		filing = true;
		try {
			await rpcClient.insight.fileInsightToReporter({
				insightId,
				reporterId: BigInt(selectedReporterId)
			});
			toast.success('Insight filed to reporter');
			showFileModal = false;
			selectedReporterId = '';
			load();
		} catch {
			// handled
		} finally {
			filing = false;
		}
	}
</script>

<div class="breadcrumb">
	<a href="/insights">Insights</a>
	<ChevronRight class="w-3 h-3 separator" />
	<span class="current">{insight?.title || '...'}</span>
</div>

{#if loading}
	<div class="space-y-4">
		<div class="skeleton h-8 w-96"></div>
		<div class="skeleton h-40 w-full"></div>
	</div>
{:else if insight}
	<div class="flex items-start justify-between mb-6">
		<div class="flex-1 min-w-0">
			<h1 class="page-title">{insight.title}</h1>
			<div class="flex items-center gap-3 mt-2 flex-wrap">
				<span class="badge {severityColor(insight.severity)}">{insight.severity}</span>
				<span class="badge {statusColor(insight.status)}">{insight.status}</span>
				<span class="text-xs text-muted-foreground">{insight.category}</span>
				{#if insight.productName}
					<span class="text-xs text-muted-foreground">· <a href="/products/{insight.productId}" class="hover:text-foreground">{insight.productName}</a></span>
				{/if}
			</div>
		</div>
		<div class="flex items-center gap-2 shrink-0">
			<button class="btn-primary btn-sm" onclick={() => (showFileModal = true)}>
				<Send class="w-3 h-3" /> File to Reporter
			</button>
		</div>
	</div>

	<div class="grid grid-cols-1 lg:grid-cols-3 gap-5 mb-5">
		<div class="lg:col-span-2 space-y-5">
			<div class="card">
				<div class="card-header">
					<h3 class="card-title">Description</h3>
				</div>
				<div class="card-body">
					<p class="text-sm leading-relaxed whitespace-pre-wrap">{insight.description || 'No description'}</p>
				</div>
			</div>

			{#if insight.reports.length > 0}
				<div class="card">
					<div class="card-header">
						<h3 class="card-title flex items-center gap-2"><FileText class="w-4 h-4 text-muted-foreground" /> Filed Reports</h3>
					</div>
					<div class="overflow-x-auto">
						<table class="data-table">
							<thead>
								<tr>
									<th>Reporter</th>
									<th>External ID</th>
									<th>Status</th>
									<th>Created</th>
									<th></th>
								</tr>
							</thead>
							<tbody>
								{#each insight.reports as report}
									<tr>
										<td><span class="badge badge-info">{report.reporterType}</span></td>
										<td class="font-mono text-xs">{report.externalId || '—'}</td>
										<td><span class="badge {statusColor(report.status)}">{report.status}</span></td>
										<td class="text-xs text-muted-foreground">{formatDate(report.createdAt)}</td>
										<td>
											{#if report.externalUrl}
												<a href={report.externalUrl} target="_blank" rel="noopener" class="btn-ghost btn-xs text-primary">
													<ExternalLink class="w-3 h-3" /> Open
												</a>
											{/if}
										</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
				</div>
			{/if}
		</div>

		<div class="space-y-5">
			<div class="card">
				<div class="card-header">
					<h3 class="card-title">Status</h3>
				</div>
				<div class="card-body space-y-2">
					{#each ['new', 'confirmed', 'resolved', 'dismissed'] as s}
						<button
							class="w-full text-left px-3 py-1.5 rounded text-sm transition-colors {insight.status === s ? 'bg-primary/10 text-primary font-medium' : 'hover:bg-muted/50 text-muted-foreground'}"
							onclick={() => updateStatus(s)}
						>
							{s.charAt(0).toUpperCase() + s.slice(1)}
						</button>
					{/each}
				</div>
			</div>

			<div class="card">
				<div class="card-header">
					<h3 class="card-title">Metadata</h3>
				</div>
				<div class="card-body space-y-2">
					<div class="detail-row"><span class="detail-label">Fingerprint</span><span class="detail-value text-xs font-mono truncate max-w-[140px]">{insight.fingerprint || '—'}</span></div>
					<div class="detail-row"><span class="detail-label">Source IDs</span><span class="detail-value text-xs font-mono truncate max-w-[140px]">{insight.sourceMsgIds || '—'}</span></div>
					<div class="detail-row"><span class="detail-label">Created</span><span class="detail-value text-xs">{formatDate(insight.createdAt)}</span></div>
					<div class="detail-row"><span class="detail-label">Updated</span><span class="detail-value text-xs">{formatDate(insight.updatedAt)}</span></div>
					<div class="detail-row"><span class="detail-label">ID</span><span class="detail-value text-xs font-mono">{String(insight.id)}</span></div>
				</div>
			</div>
		</div>
	</div>

	{#if showFileModal}
		<div class="modal-backdrop" onclick={() => (showFileModal = false)} role="presentation">
			<div class="modal" onclick={(e) => e.stopPropagation()} role="dialog" style="max-width: 420px;">
				<div class="modal-header">
					<h3 class="modal-title">File to Reporter</h3>
					<button class="btn-icon-sm" onclick={() => (showFileModal = false)}><X class="w-4 h-4" /></button>
				</div>
				<div class="modal-body">
					<p class="text-sm text-muted-foreground mb-3">Select a reporter to file this insight as an external ticket.</p>
					<div class="form-group">
						<label class="form-label" for="reporter">Reporter</label>
						<select id="reporter" class="form-select" bind:value={selectedReporterId}>
							<option value="">Select a reporter</option>
							{#each reporters as r}
								<option value={String(r.id)}>{r.name} ({r.type})</option>
							{/each}
						</select>
					</div>
				</div>
				<div class="modal-footer">
					<button class="btn-secondary" onclick={() => (showFileModal = false)}>Cancel</button>
					<button class="btn-primary" onclick={fileToReporter} disabled={filing || !selectedReporterId}>
						{filing ? 'Filing...' : 'File Insight'}
					</button>
				</div>
			</div>
		</div>
	{/if}
{:else}
	<div class="card">
		<div class="empty-state">
			<p class="empty-state-title">Insight not found</p>
			<a href="/insights" class="btn-primary btn-sm">Back to Insights</a>
		</div>
	</div>
{/if}
