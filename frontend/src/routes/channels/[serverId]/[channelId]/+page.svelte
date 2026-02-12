<script lang="ts">
	import { page } from '$app/stores';
	import { onMount } from 'svelte';
	import { currentServer } from '$lib/stores/servers';
	import { currentChannel, loadServerChannels } from '$lib/stores/channels';
	import { sendMessage } from '$lib/stores/messages';
	import MessageList from '$lib/components/MessageList.svelte';
	import MessageInput from '$lib/components/MessageInput.svelte';
	
	$: serverId = $page.params.serverId;
	$: channelId = $page.params.channelId;
	
	$: pageTitle = $currentChannel
		? `${$currentChannel.type === 1 
			? $currentChannel.recipients?.[0]?.username 
			: '#' + $currentChannel.name} | ${$currentServer?.name || 'Hearth'}`
		: $currentServer?.name || 'Hearth';
	
	onMount(() => {
		if (serverId && serverId !== '@me') {
			loadServerChannels(serverId);
		}
	});
	
	async function handleSend(event: CustomEvent<{ content: string; attachments: File[] }>) {
		if (!$currentChannel) return;
		
		const { content, attachments } = event.detail;
		
		try {
			await sendMessage($currentChannel.id, content, attachments);
		} catch (error) {
			console.error('Failed to send message:', error);
		}
	}
</script>

<svelte:head>
	<title>{pageTitle}</title>
</svelte:head>

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
		<button title="Search">🔍</button>
		<button title="Members">👥</button>
	</div>
</div>

<MessageList />

<MessageInput on:send={handleSend} />

<style>
	.channel-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 0 16px;
		height: 48px;
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
	
	.header-actions button {
		background: none;
		border: none;
		cursor: pointer;
		padding: 4px;
		font-size: 18px;
		opacity: 0.8;
	}
	
	.header-actions button:hover {
		opacity: 1;
	}
</style>
