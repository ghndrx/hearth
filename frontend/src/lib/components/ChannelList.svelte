<script lang="ts">
	import { channels, currentChannel } from '$lib/stores/channels';
	import { currentServer, leaveServer } from '$lib/stores/servers';
	import { user } from '$lib/stores/auth';
	import { settings } from '$lib/stores/settings';
	import { createEventDispatcher } from 'svelte';
	import UserPanel from './UserPanel.svelte';
	
	const dispatch = createEventDispatcher();
	
	let showServerMenu = false;
	
	$: serverChannels = $channels.filter(c => c.server_id === $currentServer?.id);
	$: textChannels = serverChannels.filter(c => c.type === 0);
	$: voiceChannels = serverChannels.filter(c => c.type === 2);
	$: isOwner = $currentServer?.owner_id === $user?.id;
	
	function selectChannel(channel: any) {
		currentChannel.set(channel);
		dispatch('select', channel);
	}
	
	function toggleServerMenu() {
		showServerMenu = !showServerMenu;
	}
	
	function closeServerMenu() {
		showServerMenu = false;
	}
	
	function openServerSettings() {
		closeServerMenu();
		settings.openServerSettings();
	}
	
	async function handleLeaveServer() {
		if (!$currentServer) return;
		if (!confirm(`Are you sure you want to leave ${$currentServer.name}?`)) return;
		closeServerMenu();
		try {
			await leaveServer($currentServer.id);
		} catch (error) {
			console.error('Failed to leave server:', error);
		}
	}
</script>

<svelte:window on:click={closeServerMenu} />

<div class="channel-list">
	<div class="channel-list-content">
	{#if $currentServer}
		<div class="server-header-wrapper">
			<button 
				class="server-header" 
				class:menu-open={showServerMenu}
				on:click|stopPropagation={toggleServerMenu}
			>
				<h2>{$currentServer.name}</h2>
				<svg viewBox="0 0 24 24" width="18" height="18" fill="currentColor" class="dropdown-icon">
					{#if showServerMenu}
						<path d="M18.3 5.71a.996.996 0 0 0-1.41 0L12 10.59 7.11 5.7A.996.996 0 1 0 5.7 7.11L10.59 12 5.7 16.89a.996.996 0 1 0 1.41 1.41L12 13.41l4.89 4.89a.996.996 0 1 0 1.41-1.41L13.41 12l4.89-4.89c.38-.38.38-1.02 0-1.4z"/>
					{:else}
						<path d="M7 10l5 5 5-5z"/>
					{/if}
				</svg>
			</button>
			
			{#if showServerMenu}
				<!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
				<div class="server-dropdown" on:click|stopPropagation>
					<button class="dropdown-item" on:click={() => { closeServerMenu(); /* TODO: invite modal */ }}>
						<svg viewBox="0 0 24 24" width="18" height="18" fill="currentColor">
							<path d="M15 12c2.21 0 4-1.79 4-4s-1.79-4-4-4-4 1.79-4 4 1.79 4 4 4zm-9-2V7H4v3H1v2h3v3h2v-3h3v-2H6zm9 4c-2.67 0-8 1.34-8 4v2h16v-2c0-2.66-5.33-4-8-4z"/>
						</svg>
						<span>Invite People</span>
					</button>
					
					<button class="dropdown-item" on:click={openServerSettings}>
						<svg viewBox="0 0 24 24" width="18" height="18" fill="currentColor">
							<path d="M19.14 12.94c.04-.31.06-.63.06-.94 0-.31-.02-.63-.06-.94l2.03-1.58a.49.49 0 0 0 .12-.61l-1.92-3.32a.49.49 0 0 0-.59-.22l-2.39.96c-.5-.38-1.03-.7-1.62-.94l-.36-2.54a.484.484 0 0 0-.48-.41h-3.84c-.24 0-.43.17-.47.41l-.36 2.54c-.59.24-1.13.57-1.62.94l-2.39-.96a.49.49 0 0 0-.59.22L2.74 8.87c-.12.21-.08.47.12.61l2.03 1.58c-.04.31-.06.63-.06.94s.02.63.06.94l-2.03 1.58a.49.49 0 0 0-.12.61l1.92 3.32c.12.22.37.29.59.22l2.39-.96c.5.38 1.03.7 1.62.94l.36 2.54c.05.24.24.41.48.41h3.84c.24 0 .44-.17.47-.41l.36-2.54c.59-.24 1.13-.56 1.62-.94l2.39.96c.22.08.47 0 .59-.22l1.92-3.32c.12-.22.07-.47-.12-.61l-2.01-1.58zM12 15.6c-1.98 0-3.6-1.62-3.6-3.6s1.62-3.6 3.6-3.6 3.6 1.62 3.6 3.6-1.62 3.6-3.6 3.6z"/>
						</svg>
						<span>Server Settings</span>
					</button>
					
					<div class="dropdown-divider"></div>
					
					{#if !isOwner}
						<button class="dropdown-item danger" on:click={handleLeaveServer}>
							<svg viewBox="0 0 24 24" width="18" height="18" fill="currentColor">
								<path d="M10.09 15.59L11.5 17l5-5-5-5-1.41 1.41L12.67 11H3v2h9.67l-2.58 2.59zM19 3H5a2 2 0 0 0-2 2v4h2V5h14v14H5v-4H3v4a2 2 0 0 0 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2z"/>
							</svg>
							<span>Leave Server</span>
						</button>
					{/if}
				</div>
			{/if}
		</div>
		
		<!-- Text Channels -->
		{#if textChannels.length > 0}
			<div class="channel-category">
				<button class="category-header">
					<svg viewBox="0 0 24 24" width="12" height="12" fill="currentColor">
						<path d="M7 10l5 5 5-5z"/>
					</svg>
					<span>TEXT CHANNELS</span>
				</button>
				<button class="add-channel" title="Create Channel">
					<svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor">
						<path d="M19 13h-6v6h-2v-6H5v-2h6V5h2v6h6v2z"/>
					</svg>
				</button>
			</div>
			
			{#each textChannels as channel (channel.id)}
				<button
					class="channel"
					class:active={$currentChannel?.id === channel.id}
					on:click={() => selectChannel(channel)}
				>
					<span class="channel-icon">#</span>
					<span class="channel-name">{channel.name}</span>
					{#if channel.e2ee_enabled}
						<span class="e2ee-badge" title="End-to-End Encrypted">🔒</span>
					{/if}
				</button>
			{/each}
		{/if}
		
		<!-- Voice Channels -->
		{#if voiceChannels.length > 0}
			<div class="channel-category">
				<button class="category-header">
					<svg viewBox="0 0 24 24" width="12" height="12" fill="currentColor">
						<path d="M7 10l5 5 5-5z"/>
					</svg>
					<span>VOICE CHANNELS</span>
				</button>
				<button class="add-channel" title="Create Channel">
					<svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor">
						<path d="M19 13h-6v6h-2v-6H5v-2h6V5h2v6h6v2z"/>
					</svg>
				</button>
			</div>
			
			{#each voiceChannels as channel (channel.id)}
				<button
					class="channel voice"
					class:active={$currentChannel?.id === channel.id}
					on:click={() => selectChannel(channel)}
				>
					<svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor" class="channel-icon">
						<path d="M12 14c1.66 0 3-1.34 3-3V5c0-1.66-1.34-3-3-3S9 3.34 9 5v6c0 1.66 1.34 3 3 3zm5.91-3c-.49 0-.9.36-.98.85C16.52 14.2 14.47 16 12 16s-4.52-1.8-4.93-4.15c-.08-.49-.49-.85-.98-.85-.61 0-1.09.54-1 1.14.49 3 2.89 5.35 5.91 5.78V20c0 .55.45 1 1 1s1-.45 1-1v-2.08c3.02-.43 5.42-2.78 5.91-5.78.1-.6-.39-1.14-1-1.14z"/>
					</svg>
					<span class="channel-name">{channel.name}</span>
				</button>
			{/each}
		{/if}
	{:else}
		<!-- DM List -->
		<div class="dm-header">
			<button class="dm-search">Find or start a conversation</button>
		</div>
		
		<div class="dm-section">
			<span>DIRECT MESSAGES</span>
		</div>
		
		{#each $channels.filter(c => c.type === 1 || c.type === 3) as dm (dm.id)}
			<button
				class="dm-item"
				class:active={$currentChannel?.id === dm.id}
				on:click={() => selectChannel(dm)}
			>
				<div class="dm-avatar">
					{#if dm.recipients?.[0]?.avatar}
						<img src={dm.recipients[0].avatar} alt="" />
					{:else}
						<div class="avatar-placeholder"></div>
					{/if}
				</div>
				<span class="dm-name">
					{dm.name || dm.recipients?.map(r => r.display_name || r.username).join(', ') || 'Unknown'}
				</span>
				<span class="e2ee-indicator">🔒</span>
			</button>
		{/each}
	{/if}
	</div>
	
	<UserPanel />
</div>

<style>
	.channel-list {
		display: flex;
		flex-direction: column;
		width: 240px;
		background: var(--bg-secondary);
	}
	
	.channel-list-content {
		flex: 1;
		overflow-y: auto;
	}
	
	.server-header-wrapper {
		position: relative;
	}
	
	.server-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 12px 16px;
		border-bottom: 1px solid var(--bg-modifier-accent);
		cursor: pointer;
		width: 100%;
		background: none;
		border-left: none;
		border-right: none;
		border-top: none;
		transition: background 0.15s ease;
	}
	
	.server-header:hover {
		background: var(--bg-modifier-hover);
	}
	
	.server-header.menu-open {
		background: var(--bg-modifier-selected);
	}
	
	.server-header h2 {
		font-size: 15px;
		font-weight: 600;
		color: var(--text-primary);
		margin: 0;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	
	.dropdown-icon {
		color: var(--text-primary);
		flex-shrink: 0;
	}
	
	.server-dropdown {
		position: absolute;
		top: calc(100% + 4px);
		left: 8px;
		right: 8px;
		background: var(--bg-floating);
		border-radius: 4px;
		padding: 6px 8px;
		box-shadow: 0 8px 16px rgba(0, 0, 0, 0.24);
		z-index: 100;
	}
	
	.dropdown-item {
		display: flex;
		align-items: center;
		gap: 10px;
		width: 100%;
		padding: 8px;
		background: none;
		border: none;
		border-radius: 2px;
		color: var(--text-secondary);
		font-size: 14px;
		cursor: pointer;
		text-align: left;
	}
	
	.dropdown-item:hover {
		background: var(--brand-primary);
		color: white;
	}
	
	.dropdown-item:hover svg {
		fill: white;
	}
	
	.dropdown-item.danger {
		color: var(--status-danger);
	}
	
	.dropdown-item.danger:hover {
		background: var(--status-danger);
		color: white;
	}
	
	.dropdown-divider {
		height: 1px;
		margin: 4px;
		background: var(--bg-modifier-accent);
	}
	
	.channel-category {
		display: flex;
		align-items: center;
		padding: 16px 8px 4px 16px;
	}
	
	.category-header {
		display: flex;
		align-items: center;
		gap: 4px;
		background: none;
		border: none;
		color: var(--text-muted);
		font-size: 12px;
		font-weight: 600;
		letter-spacing: 0.02em;
		cursor: pointer;
		flex: 1;
		text-align: left;
	}
	
	.add-channel {
		background: none;
		border: none;
		color: var(--text-muted);
		cursor: pointer;
		padding: 0;
		opacity: 0;
	}
	
	.channel-category:hover .add-channel {
		opacity: 1;
	}
	
	.channel {
		display: flex;
		align-items: center;
		gap: 6px;
		padding: 6px 8px;
		margin: 1px 8px;
		border-radius: 4px;
		background: none;
		border: none;
		color: var(--text-muted);
		font-size: 16px;
		cursor: pointer;
		width: calc(100% - 16px);
		text-align: left;
	}
	
	.channel:hover {
		background: var(--bg-modifier-hover);
		color: var(--text-primary);
	}
	
	.channel.active {
		background: var(--bg-modifier-selected);
		color: var(--text-primary);
	}
	
	.channel-icon {
		color: var(--text-muted);
		font-size: 20px;
		font-weight: 500;
		width: 20px;
		text-align: center;
	}
	
	.channel-name {
		flex: 1;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	
	.e2ee-badge {
		font-size: 12px;
	}
	
	/* DM styles */
	.dm-header {
		padding: 10px;
	}
	
	.dm-search {
		width: 100%;
		padding: 6px;
		background: var(--bg-tertiary);
		border: none;
		border-radius: 4px;
		color: var(--text-muted);
		font-size: 14px;
		cursor: pointer;
		text-align: left;
	}
	
	.dm-section {
		padding: 16px 8px 4px 16px;
		font-size: 12px;
		font-weight: 600;
		color: var(--text-muted);
		letter-spacing: 0.02em;
	}
	
	.dm-item {
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 6px 8px;
		margin: 1px 8px;
		border-radius: 4px;
		background: none;
		border: none;
		cursor: pointer;
		width: calc(100% - 16px);
	}
	
	.dm-item:hover {
		background: var(--bg-modifier-hover);
	}
	
	.dm-item.active {
		background: var(--bg-modifier-selected);
	}
	
	.dm-avatar {
		width: 32px;
		height: 32px;
		border-radius: 50%;
		overflow: hidden;
		background: var(--bg-tertiary);
	}
	
	.dm-avatar img {
		width: 100%;
		height: 100%;
		object-fit: cover;
	}
	
	.dm-name {
		flex: 1;
		color: var(--text-muted);
		font-size: 16px;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	
	.dm-item.active .dm-name,
	.dm-item:hover .dm-name {
		color: var(--text-primary);
	}
	
	.e2ee-indicator {
		font-size: 12px;
		opacity: 0.6;
	}
</style>
