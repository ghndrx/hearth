<script lang="ts">
	import { onMount, afterUpdate } from 'svelte';
	import { messages, loadMessages } from '$lib/stores/messages';
	import { currentChannel } from '$lib/stores/channels';
	import { user } from '$lib/stores/auth';
	import Message from './Message.svelte';
	
	let messageContainer: HTMLElement;
	let shouldScroll = true;
	
	$: if ($currentChannel) {
		loadMessages($currentChannel.id);
	}
	
	$: channelMessages = $messages[$currentChannel?.id] || [];
	
	function handleScroll() {
		const { scrollTop, scrollHeight, clientHeight } = messageContainer;
		shouldScroll = scrollHeight - scrollTop - clientHeight < 100;
	}
	
	afterUpdate(() => {
		if (shouldScroll && messageContainer) {
			messageContainer.scrollTop = messageContainer.scrollHeight;
		}
	});
	
	function formatDate(date: string) {
		const d = new Date(date);
		const now = new Date();
		const diffDays = Math.floor((now.getTime() - d.getTime()) / (1000 * 60 * 60 * 24));
		
		if (diffDays === 0) return 'Today';
		if (diffDays === 1) return 'Yesterday';
		return d.toLocaleDateString(undefined, { weekday: 'long', month: 'long', day: 'numeric' });
	}
	
	function shouldShowDate(index: number): boolean {
		if (index === 0) return true;
		const current = new Date(channelMessages[index].created_at).toDateString();
		const prev = new Date(channelMessages[index - 1].created_at).toDateString();
		return current !== prev;
	}
	
	function shouldGroupWithPrevious(index: number): boolean {
		if (index === 0) return false;
		const current = channelMessages[index];
		const prev = channelMessages[index - 1];
		
		if (current.author_id !== prev.author_id) return false;
		
		const timeDiff = new Date(current.created_at).getTime() - new Date(prev.created_at).getTime();
		return timeDiff < 7 * 60 * 1000; // 7 minutes
	}
</script>

<div class="message-list" bind:this={messageContainer} on:scroll={handleScroll}>
	{#if $currentChannel}
		<div class="channel-welcome">
			<div class="channel-icon-large">
				{#if $currentChannel.type === 1 || $currentChannel.type === 3}
					<div class="dm-avatar-large"></div>
				{:else}
					<span>#</span>
				{/if}
			</div>
			<h1>
				{#if $currentChannel.type === 1}
					{$currentChannel.recipients?.[0]?.display_name || $currentChannel.recipients?.[0]?.username || 'Unknown'}
				{:else}
					Welcome to #{$currentChannel.name}!
				{/if}
			</h1>
			<p>
				{#if $currentChannel.type === 1}
					This is the beginning of your direct message history with <strong>{$currentChannel.recipients?.[0]?.username}</strong>.
					{#if $currentChannel.e2ee_enabled}
						<span class="e2ee-notice">🔒 Messages are end-to-end encrypted.</span>
					{/if}
				{:else if $currentChannel.type === 3}
					This is the beginning of this group DM.
				{:else}
					This is the start of the #{$currentChannel.name} channel.
					{$currentChannel.topic || ''}
				{/if}
			</p>
		</div>
		
		{#each channelMessages as message, i (message.id)}
			{#if shouldShowDate(i)}
				<div class="date-divider">
					<span>{formatDate(message.created_at)}</span>
				</div>
			{/if}
			
			<Message 
				{message} 
				grouped={shouldGroupWithPrevious(i)}
				isOwn={message.author_id === $user?.id}
			/>
		{/each}
	{:else}
		<div class="no-channel">
			<p>Select a channel to start chatting</p>
		</div>
	{/if}
</div>

<style>
	.message-list {
		flex: 1;
		overflow-y: auto;
		padding: 0 16px;
		display: flex;
		flex-direction: column;
	}
	
	.channel-welcome {
		margin: 16px 0;
		padding: 16px 0;
	}
	
	.channel-icon-large {
		width: 68px;
		height: 68px;
		border-radius: 50%;
		background: var(--bg-modifier-accent);
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 42px;
		color: var(--text-primary);
		margin-bottom: 8px;
	}
	
	.channel-welcome h1 {
		font-size: 32px;
		font-weight: 700;
		color: var(--text-primary);
		margin: 8px 0;
	}
	
	.channel-welcome p {
		font-size: 14px;
		color: var(--text-muted);
		margin: 0;
	}
	
	.e2ee-notice {
		display: block;
		margin-top: 8px;
		color: var(--text-positive);
	}
	
	.date-divider {
		display: flex;
		align-items: center;
		margin: 16px 0 8px;
	}
	
	.date-divider::before,
	.date-divider::after {
		content: '';
		flex: 1;
		height: 1px;
		background: var(--bg-modifier-accent);
	}
	
	.date-divider span {
		padding: 0 8px;
		font-size: 12px;
		font-weight: 600;
		color: var(--text-muted);
	}
	
	.no-channel {
		display: flex;
		align-items: center;
		justify-content: center;
		height: 100%;
		color: var(--text-muted);
	}
</style>
