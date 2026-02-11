<script lang="ts">
	import { channels, currentChannel } from '$lib/stores/channels';
	import { currentServer } from '$lib/stores/servers';
	import { createEventDispatcher } from 'svelte';
	
	const dispatch = createEventDispatcher();
	
	$: serverChannels = $channels.filter(c => c.server_id === $currentServer?.id);
	$: textChannels = serverChannels.filter(c => c.type === 0);
	$: voiceChannels = serverChannels.filter(c => c.type === 2);
	
	function selectChannel(channel: any) {
		currentChannel.set(channel);
		dispatch('select', channel);
	}
</script>

<div class="channel-list">
	{#if $currentServer}
		<div class="server-header">
			<h2>{$currentServer.name}</h2>
			<button class="dropdown-btn">
				<svg viewBox="0 0 24 24" width="18" height="18" fill="currentColor">
					<path d="M7 10l5 5 5-5z"/>
				</svg>
			</button>
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

<style>
	.channel-list {
		display: flex;
		flex-direction: column;
		width: 240px;
		background: var(--bg-secondary);
		overflow-y: auto;
	}
	
	.server-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 12px 16px;
		border-bottom: 1px solid var(--bg-modifier-accent);
		cursor: pointer;
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
	
	.dropdown-btn {
		background: none;
		border: none;
		color: var(--text-primary);
		cursor: pointer;
		padding: 0;
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
