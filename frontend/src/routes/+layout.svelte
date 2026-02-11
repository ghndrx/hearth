<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { auth, isAuthenticated, websocket, servers, messages } from '$stores';
	import '../app.css';

	const publicRoutes = ['/login', '/register'];

	onMount(async () => {
		await auth.init();

		// Set up reactive navigation based on auth state
		const unsubscribe = auth.subscribe(($auth) => {
			if (!$auth.loading) {
				const isPublic = publicRoutes.some((route) => $page.url.pathname.startsWith(route));

				if (!$auth.user && !isPublic) {
					goto('/login');
				} else if ($auth.user && isPublic) {
					goto('/');
				}

				// Connect WebSocket when authenticated
				if ($auth.user) {
					websocket.connect();
					servers.loadServers();
				} else {
					websocket.disconnect();
					servers.reset();
					messages.reset();
				}
			}
		});

		return () => {
			unsubscribe();
			websocket.disconnect();
		};
	});
</script>

{#if $auth.loading}
	<div class="h-screen flex items-center justify-center bg-dark-950">
		<div class="flex flex-col items-center gap-4">
			<div class="w-12 h-12 border-4 border-hearth-500 border-t-transparent rounded-full animate-spin"></div>
			<p class="text-gray-400">Loading Hearth...</p>
		</div>
	</div>
{:else}
	<slot />
{/if}
