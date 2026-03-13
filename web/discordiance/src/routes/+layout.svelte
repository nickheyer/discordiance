<script lang="ts">
	import '../app.css';
	import { ModeWatcher } from 'mode-watcher';
	import { page } from '$app/state';
	import { toggleMode, mode } from 'mode-watcher';
	import { Toaster } from 'svelte-sonner';
	import {
		LayoutDashboard,
		Package,
		AlertTriangle,
		MessageSquare,
		Sun,
		Moon,
		Menu,
		X
	} from '@lucide/svelte';

	let { children } = $props();
	let mobileOpen = $state(false);

	function isActive(path: string): boolean {
		if (path === '/') return page.url.pathname === '/';
		return page.url.pathname.startsWith(path);
	}

	const nav = [
		{ href: '/', label: 'Dashboard', icon: LayoutDashboard },
		{ href: '/products', label: 'Products', icon: Package },
		{ href: '/issues', label: 'Issues', icon: AlertTriangle },
		{ href: '/messages', label: 'Messages', icon: MessageSquare }
	];
</script>

<svelte:head>
	<title>Discordiance</title>
</svelte:head>

<ModeWatcher />
<Toaster position="bottom-right" richColors />

<div class="min-h-screen flex flex-col">
	<!-- Top bar -->
	<header class="sticky top-0 z-50 border-b border-border bg-card/80 backdrop-blur-md">
		<div class="mx-auto flex h-12 max-w-6xl items-center px-4">
			<a href="/" class="font-semibold tracking-tight text-primary mr-8">discordiance</a>

			<nav class="hidden md:flex items-center gap-1 flex-1">
				{#each nav as link}
					<a
						href={link.href}
						class="flex items-center gap-1.5 px-2.5 py-1.5 rounded-md text-sm transition-colors
							{isActive(link.href)
							? 'bg-primary/10 text-primary font-medium'
							: 'text-muted-foreground hover:text-foreground hover:bg-muted'}"
					>
						<link.icon class="h-4 w-4" />
						{link.label}
					</a>
				{/each}
			</nav>

			<div class="flex-1 md:flex-none"></div>

			<button class="btn-icon" onclick={toggleMode}>
				{#if mode.current === 'light'}
					<Moon class="h-4 w-4 text-muted-foreground" />
				{:else}
					<Sun class="h-4 w-4 text-muted-foreground" />
				{/if}
			</button>

			<button class="btn-icon md:hidden ml-1" onclick={() => (mobileOpen = !mobileOpen)}>
				{#if mobileOpen}<X class="h-4 w-4" />{:else}<Menu class="h-4 w-4" />{/if}
			</button>
		</div>

		{#if mobileOpen}
			<nav class="md:hidden border-t border-border px-4 py-2 bg-card">
				{#each nav as link}
					<a
						href={link.href}
						class="flex items-center gap-2 px-3 py-2 rounded-md text-sm transition-colors
							{isActive(link.href)
							? 'bg-primary/10 text-primary font-medium'
							: 'text-muted-foreground hover:text-foreground hover:bg-muted'}"
						onclick={() => (mobileOpen = false)}
					>
						<link.icon class="h-4 w-4" />
						{link.label}
					</a>
				{/each}
			</nav>
		{/if}
	</header>

	<main class="flex-1">
		{#key page.url.pathname}
			<div class="mx-auto max-w-6xl px-4 py-6 page-enter">
				{@render children?.()}
			</div>
		{/key}
	</main>
</div>
