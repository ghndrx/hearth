<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { isAuthenticated } from '$lib/stores/auth';
	import { loadServers } from '$lib/stores/servers';
	import { loadDMChannels } from '$lib/stores/channels';
	import ServerList from '$lib/components/ServerList.svelte';
	import ChannelList from '$lib/components/ChannelList.svelte';
	
	onMount(() => {
		if (!$isAuthenticated) {
			goto('/login');
			return;
		}
		
		loadServers();
		loadDMChannels();
	});
	
	$: if ($isAuthenticated === false) {
		goto('/login');
	}
</script>

<div class="app-layout">
	<ServerList />
	<ChannelList />
	<main class="main-content">
		<slot />
	</main>
</div>

<style>
	.app-layout {
		display: flex;
		height: 100vh;
		overflow: hidden;
	}
	
	.main-content {
		flex: 1;
		display: flex;
		flex-direction: column;
		background: var(--bg-primary);
		min-width: 0;
	}
</style>
