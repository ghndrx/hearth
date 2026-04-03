<script lang="ts">
	import { page } from '$app/stores';
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { currentServer, servers } from '$lib/stores/servers';
	import { currentChannel, loadServerChannels, channels } from '$lib/stores/channels';
	import { splitViewStore, splitViewEnabled, canAddSplitPanel } from '$lib/stores/splitView';
	import { fetchUnreadState, markChannelRead } from '$lib/stores/unread';
	import Tooltip from '$lib/components/Tooltip.svelte';
	import ForumPostView from '$lib/components/forum/ForumPostView.svelte';

	$: serverId = $page.params.serverId;
	$: channelId = $page.params.channelId;
	$: threadId = $page.params.threadId;

	// Set currentServer from URL if not already set
	$: if (serverId && serverId !== '@me' && $servers.length > 0) {
		const server = $servers.find(s => s.id === serverId);
		if (server && $currentServer?.id !== serverId) {
			currentServer.set(server);
		}
	}

	// Set currentChannel from URL - check both channels array and try to load if not found
	$: if (channelId) {
		const channel = $channels.find(c => c.id === channelId);
		if (channel && $currentChannel?.id !== channelId) {
			currentChannel.set(channel);
		} else if (!channel && serverId && serverId !== '@me') {
			loadServerChannels(serverId);
		}
	}

	$: pageTitle = $currentChannel
		? `Thread | ${$currentChannel.name} | ${$currentServer?.name || 'Hearth'}`
		: $currentServer?.name || 'Hearth';

	$: isForumChannel = $currentChannel?.type === 6;

	onMount(() => {
		if (serverId && serverId !== '@me') {
			loadServerChannels(serverId);
		}
		fetchUnreadState();
	});

	// Mark channel as read when currentChannel changes
	$: if ($currentChannel) {
		markChannelRead($currentChannel.id);
	}

	function handleBackToForum() {
		goto(`/channels/${serverId}/${channelId}`, { replaceState: false });
	}
</script>

<svelte:head>
	<title>{pageTitle}</title>
</svelte:head>

<div class="thread-view">
	{#if isForumChannel && threadId}
		<ForumPostView
			channelId={channelId!}
			threadId={threadId!}
			on:back={handleBackToForum}
		/>
	{:else}
		<div class="not-found">
			<span>Post not found</span>
			<button on:click={handleBackToForum}>Back to channel</button>
		</div>
	{/if}
</div>

<style>
	.thread-view {
		display: flex;
		flex-direction: column;
		height: 100%;
		width: 100%;
		min-height: 0;
		overflow: hidden;
	}

	.not-found {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 12px;
		height: 100%;
		color: var(--text-muted, #b5bac1);
	}

	.not-found button {
		padding: 8px 16px;
		background: var(--brand-primary, #5865f2);
		border: none;
		border-radius: var(--radius-sm, 4px);
		color: white;
		font-size: var(--font-size-sm, 13px);
		cursor: pointer;
	}
</style>
