<script lang="ts">
	import { page } from '$app/stores';
	import { onMount } from 'svelte';
	import { currentChannel, channels, loadDMChannels, getChannel } from '$lib/stores/channels';
	import { currentServer } from '$lib/stores/servers';
	import { sendMessage } from '$lib/stores/messages';
	import { fetchUnreadState, markChannelRead } from '$lib/stores/unread';
	import MessageList from '$lib/components/MessageList.svelte';
	import MessageInput from '$lib/components/MessageInput.svelte';

	$: channelId = $page.params.channelId;

	// Clear server context for DM view
	$: if (channelId) {
		currentServer.set(null);
	}

	// Set currentChannel from URL
	$: if (channelId) {
		const channel = $channels.find(c => c.id === channelId);
		if (channel && $currentChannel?.id !== channelId) {
			currentChannel.set(channel);
		} else if (!channel) {
			// Channel not in store - try to load DM channels
			loadDMChannels().then(() => {
				const loaded = $channels.find(c => c.id === channelId);
				if (loaded) {
					currentChannel.set(loaded);
				} else {
					// Try fetching the individual channel
					getChannel(channelId).then(ch => {
						if (ch) currentChannel.set(ch);
					});
				}
			});
		}
	}

	$: recipientName = $currentChannel?.recipients?.[0]?.display_name
		|| $currentChannel?.recipients?.[0]?.username
		|| $currentChannel?.name
		|| 'Unknown';

	$: pageTitle = $currentChannel
		? `${$currentChannel.type === 3 ? $currentChannel.name || recipientName : recipientName} | Hearth`
		: 'Hearth';

	// Mark channel as read when currentChannel changes
	$: if ($currentChannel) {
		markChannelRead($currentChannel.id);
	}

	onMount(() => {
		fetchUnreadState();
	});

	async function handleSend(event: CustomEvent<{ content: string; attachments: File[] }>) {
		if (!$currentChannel) return;
		const { content, attachments } = event.detail;
		try {
			await sendMessage($currentChannel.id, content, attachments);
		} catch (error) {
			console.error('Failed to send DM:', error);
		}
	}
</script>

<svelte:head>
	<title>{pageTitle}</title>
</svelte:head>

<div class="dm-view">
	<div class="dm-header">
		{#if $currentChannel}
			<div class="dm-info">
				{#if $currentChannel.type === 3}
					<span class="group-icon">👥</span>
				{:else}
					<span class="at">@</span>
				{/if}
				<span class="dm-name">{$currentChannel.type === 3 ? ($currentChannel.name || recipientName) : recipientName}</span>
				{#if $currentChannel.e2ee_enabled}
					<span class="e2ee" title="End-to-End Encrypted">🔒</span>
				{/if}
			</div>
		{/if}
	</div>

	<div class="messages-wrapper">
		<MessageList />
	</div>

	<div class="input-wrapper">
		<MessageInput on:send={handleSend} />
	</div>
</div>

<style>
	.dm-view {
		display: flex;
		flex-direction: column;
		height: 100%;
		width: 100%;
		min-height: 0;
		overflow: hidden;
	}

	.dm-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 0 16px;
		height: 48px;
		min-height: 48px;
		border-bottom: 1px solid var(--bg-modifier-accent, #3f4147);
		background: var(--bg-primary, #313338);
		flex-shrink: 0;
	}

	.dm-info {
		display: flex;
		align-items: center;
		gap: 8px;
		min-width: 0;
	}

	.at, .group-icon {
		color: var(--text-muted, #949ba4);
		font-size: 24px;
		font-weight: 500;
	}

	.dm-name {
		font-weight: 600;
		color: var(--text-primary, #f2f3f5);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.e2ee {
		font-size: 14px;
	}

	.messages-wrapper {
		flex: 1;
		min-height: 0;
		display: flex;
		flex-direction: column;
		overflow: hidden;
	}

	.input-wrapper {
		flex-shrink: 0;
		background: var(--bg-primary, #313338);
	}
</style>
