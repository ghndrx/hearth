<script lang="ts">
	import { page } from '$app/stores';
	import { onMount } from 'svelte';
	import { currentServer, servers } from '$lib/stores/servers';
	import { currentChannel, loadServerChannels, channels } from '$lib/stores/channels';
	import { sendMessage } from '$lib/stores/messages';
	import { splitViewStore, canAddSplitPanel, splitViewEnabled } from '$lib/stores/splitView';
	import { fetchUnreadState, markChannelRead } from '$lib/stores/unread';
	import MessageList from '$lib/components/MessageList.svelte';
	import MessageInput from '$lib/components/MessageInput.svelte';
	import Tooltip from '$lib/components/Tooltip.svelte';
	import ForumPostCreateModal from '$lib/components/forum/ForumPostCreateModal.svelte';
	
	$: serverId = $page.params.serverId;
	$: channelId = $page.params.channelId;
	
	// Forum post modal state
	let showForumPostModal = false;
	
	// Debug logging
	$: console.log('[ChannelPage] State:', { 
		serverId, channelId, 
		serversCount: $servers.length, 
		channelsCount: $channels.length,
		currentServerId: $currentServer?.id,
		currentChannelId: $currentChannel?.id 
	});
	
	// Set currentServer from URL if not already set
	$: if (serverId && serverId !== '@me' && $servers.length > 0) {
		const server = $servers.find(s => s.id === serverId);
		if (server && $currentServer?.id !== serverId) {
			console.log('[ChannelPage] Setting currentServer:', server.name);
			currentServer.set(server);
		}
	}
	
	// Set currentChannel from URL - check both channels array and try to load if not found
	$: if (channelId) {
		const channel = $channels.find(c => c.id === channelId);
		if (channel && $currentChannel?.id !== channelId) {
			console.log('[ChannelPage] Setting currentChannel:', channel.name);
			currentChannel.set(channel);
		} else if (!channel && serverId && serverId !== '@me') {
			// Channel not in store - load server channels
			console.log('[ChannelPage] Channel not found, loading server channels');
			loadServerChannels(serverId);
		}
	}
	
	$: pageTitle = $currentChannel
		? `${$currentChannel.type === 1 
			? $currentChannel.recipients?.[0]?.username 
			: '#' + $currentChannel.name} | ${$currentServer?.name || 'Hearth'}`
		: $currentServer?.name || 'Hearth';
	
	// FEAT-003: Split view pin state
	$: isPinnedToSplitView = $currentChannel 
		? splitViewStore.isPinned($currentChannel.id, $currentChannel.type === 1 ? 'dm' : 'channel') 
		: false;
	$: canPinToSplitView = $splitViewEnabled && $currentChannel && ($canAddSplitPanel || isPinnedToSplitView);
	
	// Channel type check
	$: isForumChannel = $currentChannel?.type === 6;
	
	function handleToggleSplitViewPin() {
		if (!$currentChannel) return;
		
		if (isPinnedToSplitView) {
			splitViewStore.unpinByTarget($currentChannel.id, $currentChannel.type === 1 ? 'dm' : 'channel');
		} else if ($currentChannel.type === 1) {
			splitViewStore.pinDM($currentChannel, serverId || '');
		} else if (serverId) {
			splitViewStore.pinChannel($currentChannel, serverId);
		}
	}
	
	onMount(() => {
		if (serverId && serverId !== '@me') {
			loadServerChannels(serverId);
		}
		// Fetch unread state on mount
		fetchUnreadState();
	});
	
	// Mark channel as read when currentChannel changes
	$: if ($currentChannel) {
		markChannelRead($currentChannel.id);
	}
	
	async function handleSend(event: CustomEvent<{ content: string; attachments: File[] }>) {
		console.log('[Page] handleSend called, currentChannel:', $currentChannel?.id, $currentChannel?.name);
		if (!$currentChannel) {
			console.error('[Page] handleSend: No currentChannel!');
			return;
		}
		
		const { content, attachments } = event.detail;
		console.log('[Page] Sending message:', { channelId: $currentChannel.id, content: content?.substring(0, 50) });
		
		try {
			await sendMessage($currentChannel.id, content, attachments);
			console.log('[Page] Message sent successfully');
		} catch (error) {
			console.error('[Page] Failed to send message:', error);
		}
	}
	
	function handleForumPostCreated(event: CustomEvent<{ id: string; name: string }>) {
		console.log('[Page] Forum post created:', event.detail);
		showForumPostModal = false;
		// TODO: Navigate to the created post or refresh the list
	}
</script>

<svelte:head>
	<title>{pageTitle}</title>
</svelte:head>

<div class="channel-view">
<div class="channel-header">
	{#if $currentChannel}
		<div class="channel-info">
			{#if $currentChannel.type === 0}
				<span class="hash">#</span>
			{:else if $currentChannel.type === 2}
				<span class="voice-icon">🔊</span>
			{:else}
				<span class="at">@</span>
			{/if}
			<span class="channel-name">
				{$currentChannel.type === 1 
					? $currentChannel.recipients?.[0]?.display_name || $currentChannel.recipients?.[0]?.username
					: $currentChannel.name}
			</span>
			{#if $currentChannel.topic}
				<span class="divider"></span>
				<span class="topic">{$currentChannel.topic}</span>
			{/if}
			{#if $currentChannel.e2ee_enabled}
				<span class="e2ee" title="End-to-End Encrypted">🔒</span>
			{/if}
		</div>
	{/if}
	
	<div class="header-actions">
		<!-- Create Post button (forum channels) -->
		{#if isForumChannel}
			<button class="create-post-btn" on:click={() => (showForumPostModal = true)}>
				<svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor">
					<path d="M20 2H4C2.9 2 2 2.9 2 4V22L6 18H20C21.1 18 22 17.1 22 16V4C22 2.9 21.1 2 20 2Z" />
				</svg>
				Create Post
			</button>
		{/if}
		<!-- FEAT-003: Pin to Split View button -->
		{#if canPinToSplitView}
			<Tooltip text={isPinnedToSplitView ? 'Unpin from Split View' : 'Pin to Split View'}>
				<button 
					class="action-btn"
					class:pinned={isPinnedToSplitView}
					on:click={handleToggleSplitViewPin}
					title={isPinnedToSplitView ? 'Unpin from Split View' : 'Pin to Split View'}
					aria-pressed={isPinnedToSplitView}
				>
					{#if isPinnedToSplitView}
						📌
					{:else}
						📍
					{/if}
				</button>
			</Tooltip>
		{/if}
		<button class="action-btn" title="Search">🔍</button>
		<button class="action-btn" title="Members">👥</button>
	</div>
</div>

<div class="messages-wrapper">
	<MessageList />
</div>

<div class="input-wrapper">
	<MessageInput on:send={handleSend} />
</div>
</div>

<!-- Forum Post Creation Modal -->
<ForumPostCreateModal
	channelId={channelId || ''}
	show={showForumPostModal}
	on:close={() => (showForumPostModal = false)}
	on:created={handleForumPostCreated}
/>

<style>
	.channel-view {
		display: flex;
		flex-direction: column;
		height: 100%;
		width: 100%;
		min-height: 0; /* Critical for flex children to shrink */
		overflow: hidden;
	}

	.channel-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 0 16px;
		height: 48px;
		min-height: 48px;
		border-bottom: 1px solid var(--bg-modifier-accent);
		background: var(--bg-primary);
		flex-shrink: 0;
	}
	
	.channel-info {
		display: flex;
		align-items: center;
		gap: 8px;
		min-width: 0;
	}
	
	.hash, .voice-icon, .at {
		color: var(--text-muted);
		font-size: 24px;
		font-weight: 500;
	}
	
	.channel-name {
		font-weight: 600;
		color: var(--text-primary);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	
	.divider {
		width: 1px;
		height: 24px;
		background: var(--bg-modifier-accent);
	}
	
	.topic {
		color: var(--text-muted);
		font-size: 14px;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	
	.e2ee {
		font-size: 14px;
	}
	
	.header-actions {
		display: flex;
		gap: 8px;
	}
	
	.header-actions .action-btn {
		background: none;
		border: none;
		cursor: pointer;
		padding: 4px 6px;
		font-size: 18px;
		opacity: 0.8;
		border-radius: 4px;
		transition: opacity 0.15s, background-color 0.15s;
	}
	
	.header-actions .action-btn:hover {
		opacity: 1;
		background-color: var(--bg-modifier-hover, rgba(79, 84, 92, 0.16));
	}
	
	/* FEAT-003: Split view pin button pinned state */
	.header-actions .action-btn.pinned {
		opacity: 1;
		color: var(--brand-primary, #5865f2);
	}
	
	.header-actions .action-btn.pinned:hover {
		color: var(--red, #da373c);
	}

	/* Messages wrapper fills remaining space */
	.messages-wrapper {
		flex: 1;
		min-height: 0; /* Critical for overflow scrolling */
		display: flex;
		flex-direction: column;
		overflow: hidden;
	}

	/* Input stays at bottom, doesn't grow */
	.input-wrapper {
		flex-shrink: 0;
		background: var(--bg-primary, #313338);
	}

	/* Forum channel create post button */
	.create-post-btn {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		padding: 6px 14px;
		margin-right: 8px;
		background: var(--brand-primary, #5865f2);
		border: none;
		border-radius: var(--radius-sm, 4px);
		color: white;
		font-size: var(--font-size-sm, 13px);
		font-weight: 500;
		cursor: pointer;
		transition: background 0.15s;
	}

	.create-post-btn:hover {
		background: var(--brand-hover, #4752c4);
	}

	.create-post-btn svg {
		width: 18px;
		height: 18px;
	}
</style>
