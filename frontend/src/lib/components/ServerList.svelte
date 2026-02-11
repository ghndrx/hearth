<script lang="ts">
	import { servers, currentServer } from '$lib/stores/servers';
	import { createEventDispatcher } from 'svelte';
	
	const dispatch = createEventDispatcher();
	
	function selectServer(server: any) {
		currentServer.set(server);
		dispatch('select', server);
	}
	
	function createServer() {
		dispatch('create');
	}
</script>

<div class="server-list">
	<!-- Home/DMs button -->
	<button 
		class="server-icon home"
		class:active={$currentServer === null}
		on:click={() => selectServer(null)}
		title="Direct Messages"
	>
		<svg viewBox="0 0 24 24" fill="currentColor" width="24" height="24">
			<path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-1 17.93c-3.95-.49-7-3.85-7-7.93 0-.62.08-1.21.21-1.79L9 15v1c0 1.1.9 2 2 2v1.93zm6.9-2.54c-.26-.81-1-1.39-1.9-1.39h-1v-3c0-.55-.45-1-1-1H8v-2h2c.55 0 1-.45 1-1V7h2c1.1 0 2-.9 2-2v-.41c2.93 1.19 5 4.06 5 7.41 0 2.08-.8 3.97-2.1 5.39z"/>
		</svg>
	</button>
	
	<div class="separator"></div>
	
	<!-- Server list -->
	{#each $servers as server (server.id)}
		<button
			class="server-icon"
			class:active={$currentServer?.id === server.id}
			on:click={() => selectServer(server)}
			title={server.name}
		>
			{#if server.icon}
				<img src={server.icon} alt={server.name} />
			{:else}
				<span class="server-initials">
					{server.name.split(' ').map(w => w[0]).join('').slice(0, 2)}
				</span>
			{/if}
		</button>
	{/each}
	
	<!-- Add server button -->
	<button class="server-icon add" on:click={createServer} title="Add a Server">
		<svg viewBox="0 0 24 24" fill="currentColor" width="24" height="24">
			<path d="M19 13h-6v6h-2v-6H5v-2h6V5h2v6h6v2z"/>
		</svg>
	</button>
</div>

<style>
	.server-list {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 8px;
		padding: 12px 0;
		width: 72px;
		background: var(--bg-tertiary);
		overflow-y: auto;
	}
	
	.server-icon {
		width: 48px;
		height: 48px;
		border-radius: 50%;
		background: var(--bg-primary);
		border: none;
		cursor: pointer;
		display: flex;
		align-items: center;
		justify-content: center;
		color: var(--text-secondary);
		transition: all 0.15s ease;
		overflow: hidden;
	}
	
	.server-icon:hover {
		border-radius: 16px;
		background: var(--brand-primary);
		color: white;
	}
	
	.server-icon.active {
		border-radius: 16px;
		background: var(--brand-primary);
		color: white;
	}
	
	.server-icon img {
		width: 100%;
		height: 100%;
		object-fit: cover;
	}
	
	.server-initials {
		font-size: 18px;
		font-weight: 600;
		text-transform: uppercase;
	}
	
	.server-icon.home {
		background: var(--bg-secondary);
	}
	
	.server-icon.add {
		background: transparent;
		border: 1px dashed var(--text-muted);
	}
	
	.server-icon.add:hover {
		border-color: var(--brand-primary);
		background: transparent;
		color: var(--brand-primary);
	}
	
	.separator {
		width: 32px;
		height: 2px;
		background: var(--bg-modifier-accent);
		border-radius: 1px;
	}
</style>
