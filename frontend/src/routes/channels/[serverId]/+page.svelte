<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import { currentServer, servers } from '$lib/stores/servers';
	import { loadServerChannels, channels } from '$lib/stores/channels';
	
	$: serverId = $page.params.serverId;
	
	// Set currentServer from URL
	$: if (serverId && $servers.length > 0) {
		const server = $servers.find(s => s.id === serverId);
		if (server && $currentServer?.id !== serverId) {
			currentServer.set(server);
		}
	}
	
	onMount(async () => {
		if (serverId && serverId !== '@me') {
			await loadServerChannels(serverId);
		}
	});
	
	// Redirect to first channel when channels load
	$: if ($channels.length > 0 && serverId) {
		const serverChannels = $channels.filter(c => c.server_id === serverId);
		const firstChannel = serverChannels.find(c => c.type === 0) || serverChannels[0];
		if (firstChannel) {
			goto(`/channels/${serverId}/${firstChannel.id}`, { replaceState: true });
		}
	}
</script>

<svelte:head>
	<title>{$currentServer?.name || 'Server'} | Hearth</title>
</svelte:head>

<div class="empty-state">
	<div class="content">
		<div class="icon">💬</div>
		<h2>Select a channel</h2>
		<p>Choose a channel from the sidebar to start chatting</p>
	</div>
</div>

<style>
	.empty-state {
		flex: 1;
		display: flex;
		align-items: center;
		justify-content: center;
		background: var(--bg-primary);
	}
	
	.content {
		text-align: center;
		max-width: 400px;
		padding: 32px;
	}
	
	.icon {
		font-size: 64px;
		margin-bottom: 16px;
	}
	
	h2 {
		font-size: 24px;
		font-weight: 600;
		color: var(--text-primary);
		margin-bottom: 8px;
	}
	
	p {
		color: var(--text-muted);
		font-size: 16px;
	}
</style>
