<script lang="ts">
	import { rpcClient } from '$lib/api/rpc-client';
	import type { AgentConfig } from '$lib/proto/discordiance/v1/types_pb';
	import { toast } from 'svelte-sonner';
	import { Save } from '@lucide/svelte';

	let { productId, agentConfig, onchange }: {
		productId: bigint;
		agentConfig: AgentConfig | undefined;
		onchange: () => Promise<void>;
	} = $props();

	const DEFAULT_SYSTEM_PROMPT = `You are Discordiance, an AI-powered issue detection agent. You monitor community platform messages in real time to identify actionable product issues, bugs, feature requests, and complaints on behalf of project maintainers.

# Your Role
You receive batches of timestamped messages from a community platform (e.g. Discord, forums). Your job is to sift through the noise — general chatter, greetings, off-topic discussion, jokes, support that has already been resolved — and surface only genuinely actionable items that a product team should know about.

# Input Format
Messages are provided one per line as:
[YYYY-MM-DD HH:MM] AuthorName: Message content

# Analysis Guidelines

## What IS an issue:
- Bug reports: something is broken, crashes, errors, unexpected behavior
- Feature requests: users asking for new capabilities or improvements
- Complaints: recurring frustration, poor UX, missing documentation, performance problems
- Regressions: something that used to work but no longer does
- Security concerns: vulnerabilities, data exposure, auth problems

## What is NOT an issue:
- General conversation, greetings, jokes, off-topic chat
- One-off user errors or confusion that gets resolved in-thread
- Questions that are answered satisfactorily by other community members
- Vague dissatisfaction with no identifiable actionable item
- Messages that are solely praise or positive feedback

## Consolidation
If multiple messages describe the same underlying problem, consolidate them into a single issue. Reference the pattern (e.g. "Multiple users reported...") rather than creating duplicates.

# Output Schema
Respond with a JSON object containing an "issues" array. Each issue must have:

- "title": A concise, descriptive title written as you would write a bug tracker title (imperative or noun-phrase style, under 80 characters)
- "description": A detailed summary including what the problem is, any reproduction context from the messages, how many users mentioned it, and relevant quotes where helpful. Write this as if filing it directly into a bug tracker.
- "severity": One of:
  - "critical" — Data loss, security vulnerability, complete feature breakage affecting many users
  - "high" — Major functionality broken, significant degradation, blocking workflows
  - "medium" — Notable UX issues, non-blocking bugs, common pain points
  - "low" — Minor inconveniences, cosmetic issues, nice-to-have improvements
- "category": One of:
  - "bug" — Broken or incorrect behavior
  - "feature_request" — New capability or enhancement
  - "complaint" — Usability, performance, or experience grievance
  - "question" — Unanswered question that indicates a documentation or discoverability gap
  - "other" — Does not fit the above

If no actionable issues are found, return: {"issues": []}

Be conservative. It is better to miss a marginal issue than to flood the tracker with noise. Only surface items where there is a clear, actionable signal.`;

	let baseUrl = $state('');
	let apiKey = $state('');
	let orgId = $state('');
	let model = $state('');
	let systemPrompt = $state('');
	let batchSize = $state(10);
	let batchTimeout = $state(30);
	let saving = $state(false);
	let initialized = $state(false);

	$effect(() => {
		if (agentConfig && !initialized) {
			baseUrl = agentConfig.baseUrl;
			apiKey = agentConfig.apiKey;
			orgId = agentConfig.orgId;
			model = agentConfig.model;
			systemPrompt = agentConfig.systemPrompt;
			batchSize = agentConfig.batchSize || 10;
			batchTimeout = agentConfig.batchTimeout || 30;
			initialized = true;
		}
		if (!agentConfig && !initialized) {
			systemPrompt = DEFAULT_SYSTEM_PROMPT;
			initialized = true;
		}
	});

	async function save() {
		saving = true;
		try {
			await rpcClient.agentConfig.upsertAgentConfig({
				productId,
				baseUrl,
				apiKey,
				orgId,
				model,
				systemPrompt,
				batchSize,
				batchTimeout
			});
			toast.success('Agent config saved');
			await onchange();
		} finally {
			saving = false;
		}
	}
</script>

<section class="card">
	<div class="card-header">
		<h2 class="text-sm font-semibold">Agent Config</h2>
		<p class="text-xs text-muted-foreground mt-0.5">LLM that analyzes messages and detects issues</p>
	</div>
	<div class="card-body space-y-4">
		<div class="grid gap-4 sm:grid-cols-2">
			<div>
				<label for="agent-url" class="label">Base URL <span class="text-destructive">*</span></label>
				<input id="agent-url" class="input" bind:value={baseUrl} placeholder="https://api.openai.com/v1" />
				<p class="hint">OpenAI-compatible API endpoint</p>
			</div>
			<div>
				<label for="agent-key" class="label">API Key <span class="text-destructive">*</span></label>
				<input id="agent-key" class="input" type="password" bind:value={apiKey} placeholder="sk-..." />
				<p class="hint">Authentication key for the LLM API</p>
			</div>
			<div>
				<label for="agent-org" class="label">Org ID</label>
				<input id="agent-org" class="input" bind:value={orgId} placeholder="org-..." />
				<p class="hint">OpenAI organization ID (optional)</p>
			</div>
			<div>
				<label for="agent-model" class="label">Model <span class="text-destructive">*</span></label>
				<input id="agent-model" class="input" bind:value={model} placeholder="gpt-4o-mini" />
				<p class="hint">Model identifier (e.g. gpt-4o-mini, claude-3-haiku)</p>
			</div>
			<div class="grid grid-cols-2 gap-3">
				<div>
					<label for="batch-size" class="label">Batch Size</label>
					<input id="batch-size" class="input" type="number" min="1" bind:value={batchSize} />
					<p class="hint">Messages per batch</p>
				</div>
				<div>
					<label for="batch-timeout" class="label">Timeout (s)</label>
					<input id="batch-timeout" class="input" type="number" min="1" bind:value={batchTimeout} />
					<p class="hint">Max wait before processing</p>
				</div>
			</div>
		</div>
		<div>
			<label for="sys-prompt" class="label">System Prompt</label>
			<textarea
				id="sys-prompt"
				class="input font-mono text-xs leading-relaxed"
				rows={10}
				bind:value={systemPrompt}
			></textarea>
			<p class="hint">Instructions for the LLM on how to analyze messages and detect issues.</p>
		</div>
		<button class="btn-primary btn-sm" onclick={save} disabled={saving}>
			<Save class="h-3.5 w-3.5" />
			{saving ? 'Saving...' : agentConfig ? 'Update Agent Config' : 'Create Agent Config'}
		</button>
	</div>
</section>
