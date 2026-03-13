<script lang="ts">
	import '../app.css';
	import { page } from '$app/state';
	import { Toaster } from 'svelte-sonner';
	import { ModeWatcher, toggleMode, mode } from 'mode-watcher';
	import { rpcClient, silentCallOptions } from '$lib/api/rpc-client';
	import {
		LayoutDashboard,
		GitBranch,
		Package,
		Lightbulb,
		MessageSquare,
		Server,
		Bot,
		Send,
		PanelLeftClose,
		PanelLeft,
		Sun,
		Moon,
		Activity
	} from '@lucide/svelte';

	let { children } = $props();

	let collapsed = $state(false);
	let healthStatus = $state('');
	let appVersion = $state('');

	const navGroups = [
		{
			label: 'Overview',
			items: [{ label: 'Dashboard', href: '/', icon: LayoutDashboard }]
		},
		{
			label: 'Operations',
			items: [{ label: 'Pipelines', href: '/pipelines', icon: GitBranch }]
		},
		{
			label: 'Data',
			items: [
				{ label: 'Products', href: '/products', icon: Package },
				{ label: 'Insights', href: '/insights', icon: Lightbulb },
				{ label: 'Messages', href: '/messages', icon: MessageSquare }
			]
		},
		{
			label: 'Configuration',
			items: [
				{ label: 'Platforms', href: '/settings/platforms', icon: Server },
				{ label: 'Agents', href: '/settings/agents', icon: Bot },
				{ label: 'Reporters', href: '/settings/reporters', icon: Send }
			]
		}
	];

	function isActive(href: string): boolean {
		if (href === '/') return page.url.pathname === '/';
		return page.url.pathname.startsWith(href);
	}

	async function checkHealth() {
		try {
			const res = await rpcClient.health.healthCheck({}, silentCallOptions);
			healthStatus = res.status;
			appVersion = res.version;
		} catch {
			healthStatus = 'unreachable';
		}
	}

	$effect(() => {
		checkHealth();
		const interval = setInterval(checkHealth, 30000);
		return () => clearInterval(interval);
	});
</script>

<svelte:head>
	<title>Discordiance</title>
</svelte:head>

<ModeWatcher />
<Toaster position="bottom-right" richColors closeButton />

<div class="flex h-screen overflow-hidden">
	<aside class="sidebar" class:collapsed>
		<div class="sidebar-header">
			{#if !collapsed}
				<div class="flex items-center gap-2.5 min-w-0">
					<div class="w-7 h-7 rounded-md bg-primary flex items-center justify-center shrink-0">
						<Activity class="w-4 h-4 text-primary-foreground" />
					</div>
					<span class="text-sm font-bold text-foreground tracking-tight truncate">Discordiance</span>
				</div>
			{:else}
				<div class="w-7 h-7 rounded-md bg-primary flex items-center justify-center mx-auto">
					<Activity class="w-4 h-4 text-primary-foreground" />
				</div>
			{/if}
		</div>

		<nav class="sidebar-nav">
			{#each navGroups as group, gi}
				{#if !collapsed}
					<div class="sidebar-group-label" class:mt-4={gi > 0}>{group.label}</div>
				{:else if gi > 0}
					<div class="h-3"></div>
				{/if}
				{#each group.items as item}
					<a
						href={item.href}
						class="sidebar-item"
						class:active={isActive(item.href)}
						title={collapsed ? item.label : undefined}
					>
						<item.icon class="w-4 h-4 shrink-0" />
						{#if !collapsed}
							<span>{item.label}</span>
						{/if}
					</a>
				{/each}
			{/each}
		</nav>

		<div class="sidebar-footer space-y-2">
			{#if !collapsed}
				<div class="flex items-center gap-2 px-1 text-xs text-sidebar-foreground">
					<span
						class="status-dot"
						class:status-dot-success={healthStatus === 'ok'}
						class:status-dot-destructive={healthStatus === 'unreachable'}
						class:status-dot-warning={healthStatus !== '' && healthStatus !== 'ok' && healthStatus !== 'unreachable'}
						class:status-dot-muted={healthStatus === ''}
					></span>
					<span class="truncate">{healthStatus === 'ok' ? 'System Healthy' : healthStatus === 'unreachable' ? 'Unreachable' : healthStatus || 'Checking...'}</span>
					{#if appVersion}
						<span class="ml-auto opacity-50">{appVersion}</span>
					{/if}
				</div>
			{:else}
				<div class="flex justify-center">
					<span
						class="status-dot"
						class:status-dot-success={healthStatus === 'ok'}
						class:status-dot-destructive={healthStatus === 'unreachable'}
						class:status-dot-muted={healthStatus === ''}
					></span>
				</div>
			{/if}
			<div class="flex items-center" class:justify-between={!collapsed} class:justify-center={collapsed}>
				{#if !collapsed}
					<button class="btn-ghost btn-sm" onclick={toggleMode}>
						{#if mode.current === 'dark'}
							<Sun class="w-3.5 h-3.5" />
							<span>Light</span>
						{:else}
							<Moon class="w-3.5 h-3.5" />
							<span>Dark</span>
						{/if}
					</button>
					<button class="btn-icon-sm" onclick={() => (collapsed = true)} title="Collapse sidebar">
						<PanelLeftClose class="w-3.5 h-3.5" />
					</button>
				{:else}
					<button class="btn-icon-sm" onclick={toggleMode} title="Toggle theme">
						{#if mode.current === 'dark'}
							<Sun class="w-3.5 h-3.5" />
						{:else}
							<Moon class="w-3.5 h-3.5" />
						{/if}
					</button>
				{/if}
			</div>
			{#if collapsed}
				<div class="flex justify-center">
					<button class="btn-icon-sm" onclick={() => (collapsed = false)} title="Expand sidebar">
						<PanelLeft class="w-3.5 h-3.5" />
					</button>
				</div>
			{/if}
		</div>
	</aside>

	<main
		class="flex-1 overflow-y-auto transition-[margin-left] duration-200 ease-in-out"
		style="margin-left: {collapsed ? '56px' : '240px'}"
	>
		<div class="max-w-[1400px] mx-auto px-6 py-6 animate-in">
			{@render children()}
		</div>
	</main>
</div>
