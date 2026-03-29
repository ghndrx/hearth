<script lang="ts">
	import { onMount, afterUpdate } from 'svelte';
	import { messages, loadMessages, toggleReaction, addReaction, jumpToMessage } from '$lib/stores/messages';
	import { currentChannel } from '$lib/stores/channels';
	import { user } from '$lib/stores/auth';
	import { currentServer } from '$lib/stores/servers';
	import { messageIn } from '$lib/utils/transitions';
	import Message from './Message.svelte';
	import EmojiPicker from './EmojiPicker.svelte';

	let messageContainer: HTMLElement;
	let shouldScroll = true;
	let previousMessageCount = 0;
	let initialLoadDone = false;

	$: if ($currentChannel) {
		initialLoadDone = false;
		previousMessageCount = 0;
		loadMessages($currentChannel.id);
	}

	$: channelMessages = ($currentChannel?.id ? $messages[$currentChannel.id] : undefined) || [];
	$: console.log('MessageList debug:', {
		channelId: $currentChannel?.id,
		messageCount: channelMessages.length,
		allMessages: $messages
	});

	// Track when initial load completes so we only animate new messages
	$: {
		const count = channelMessages.length;
		if (!initialLoadDone && count > 0) {
			queueMicrotask(() => {
				initialLoadDone = true;
				previousMessageCount = count;
			});
		}
	}

	function handleScroll() {
		const { scrollTop, scrollHeight, clientHeight } = messageContainer;
		shouldScroll = scrollHeight - scrollTop - clientHeight < 100;
	}

	afterUpdate(() => {
		if (shouldScroll && messageContainer) {
			messageContainer.scrollTop = messageContainer.scrollHeight;
		}
	});

	// Handle jump to message from pinned panel or search results
	function scrollToMessage(messageId: string) {
		if (!messageContainer) return;
		const el = messageContainer.querySelector(`[data-message-id="${messageId}"]`);
		if (el) {
			el.scrollIntoView({ behavior: 'smooth', block: 'center' });
			// Flash highlight
			el.classList.add('message-highlight');
			setTimeout(() => el.classList.remove('message-highlight'), 1500);
			// Clear the jump target after scrolling
			jumpToMessage.clear();
		}
	}

	// React to jumpToMessage store changes
	$: if ($jumpToMessage.messageId && $jumpToMessage.channelId === $currentChannel?.id) {
		scrollToMessage($jumpToMessage.messageId);
	}

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

	// Emoji picker state
	let showEmojiPicker = false;
	let emojiPickerMessageId: string | null = null;
	let emojiPickerPosition = { x: 0, y: 0 };

	function handleReact(event: CustomEvent<{ messageId: string; emoji: string }>) {
		if (!$currentChannel) return;
		const { messageId, emoji } = event.detail;
		const channelMessages = $messages[$currentChannel.id] || [];
		const msg = channelMessages.find(m => m.id === messageId);
		const reaction = msg?.reactions?.find(r => r.emoji === emoji);
		toggleReaction(messageId, $currentChannel.id, emoji, reaction?.me ?? false);
	}

	function handleOpenReactionPicker(event: CustomEvent<{ messageId: string }>) {
		emojiPickerMessageId = event.detail.messageId;
		// Position near the message action bar
		const msgEl = messageContainer?.querySelector(`[data-message-id="${event.detail.messageId}"]`);
		if (msgEl) {
			const rect = msgEl.getBoundingClientRect();
			emojiPickerPosition = { x: rect.right - 320, y: rect.top - 10 };
		}
		showEmojiPicker = true;
	}

	function handleEmojiSelect(event: CustomEvent<{ emoji: string }>) {
		if (!$currentChannel || !emojiPickerMessageId) return;
		addReaction(emojiPickerMessageId, $currentChannel.id, event.detail.emoji);
		showEmojiPicker = false;
		emojiPickerMessageId = null;
	}

	function closeEmojiPicker() {
		showEmojiPicker = false;
		emojiPickerMessageId = null;
	}

	function shouldGroupWithPrevious(index: number): boolean {
		if (index === 0) return false;
		const current = channelMessages[index];
		const prev = channelMessages[index - 1];

		// Discord-style: don't group if this is a reply (replies show reply context)
		if (current.reply_to || prev.reply_to) return false;

		if (current.author_id !== prev.author_id) return false;

		const timeDiff = new Date(current.created_at).getTime() - new Date(prev.created_at).getTime();
		return timeDiff < 5 * 60 * 1000; // 5 minutes (Discord-style)
	}
</script>

<!-- Messages container - flex-1 to fill space between header and input -->
<div
	class="message-list-container"
	bind:this={messageContainer}
	on:scroll={handleScroll}
	role="log"
	aria-label="Message history"
	aria-live="polite"
	aria-relevant="additions"
>
		{#if $currentChannel}
			<!-- Channel Welcome -->
			<div class="pt-4 pb-5">
				<div
					class="w-[68px] h-[68px] rounded-full bg-[#5865f2] flex items-center justify-center text-[#f2f3f5] text-[32px] font-medium mb-2"
				>
					{#if $currentChannel.type === 1 || $currentChannel.type === 3}
						{#if $currentChannel.recipients?.[0]?.avatar}
							<img
								src={$currentChannel.recipients[0].avatar}
								alt=""
								class="w-full h-full rounded-full object-cover"
							/>
						{:else}
							{($currentChannel.recipients?.[0]?.username || '?')[0].toUpperCase()}
						{/if}
					{:else}
						<span class="text-[42px] font-light">#</span>
					{/if}
				</div>
				<h1 class="text-[32px] font-bold text-[#f2f3f5] mb-1">
					{#if $currentChannel.type === 1}
						{$currentChannel.recipients?.[0]?.display_name ||
							$currentChannel.recipients?.[0]?.username ||
							'Unknown'}
					{:else}
						Welcome to #{$currentChannel.name}!
					{/if}
				</h1>
				<p class="text-[#b5bac1] text-base">
					{#if $currentChannel.type === 1}
						This is the beginning of your direct message history with <strong class="text-[#f2f3f5]"
							>{$currentChannel.recipients?.[0]?.username}</strong
						>.
						{#if $currentChannel.e2ee_enabled}
							<span class="block mt-2 text-[#23a559]">🔒 Messages are end-to-end encrypted.</span>
						{/if}
					{:else if $currentChannel.type === 3}
						This is the beginning of this group DM.
					{:else}
						This is the start of the #{$currentChannel.name} channel.
					{/if}
				</p>
			</div>

			<!-- Message List -->
			{#each channelMessages as message, i (message.id)}
				{#if shouldShowDate(i)}
					<div class="flex items-center my-4">
						<div class="flex-1 h-px bg-[#3f4147]"></div>
						<span class="px-2 text-xs font-semibold text-[#949ba4] uppercase"
							>{formatDate(message.created_at)}</span
						>
						<div class="flex-1 h-px bg-[#3f4147]"></div>
					</div>
				{/if}

				{#if initialLoadDone && i >= previousMessageCount}
					<div in:messageIn={{ duration: 250 }}>
						<Message
							{message}
							grouped={shouldGroupWithPrevious(i)}
							isOwn={message.author_id === $user?.id}
						/>
					</div>
				{:else}
					<Message
						{message}
						grouped={shouldGroupWithPrevious(i)}
						isOwn={message.author_id === $user?.id}
					/>
				{/if}
			{/each}
		{:else}
			<div class="flex items-center justify-center h-full">
				<p class="text-[#949ba4] text-base">Select a channel to start chatting</p>
			</div>
		{/if}
</div>

<style>
	.message-list-container {
		flex: 1;
		overflow-y: auto;
		overflow-x: hidden;
		padding: 0 16px;
		min-height: 0; /* Critical for flex overflow scrolling */
		/* Add padding at bottom so last message isn't cut off */
		padding-bottom: 8px;
	}

	/* Custom scrollbar styling */
	.message-list-container::-webkit-scrollbar {
		width: 8px;
	}

	.message-list-container::-webkit-scrollbar-track {
		background: transparent;
	}

	.message-list-container::-webkit-scrollbar-thumb {
		background: #1a1b1e;
		border-radius: 4px;
	}

	.message-list-container::-webkit-scrollbar-thumb:hover {
		background: #232428;
	}

	/* Highlight animation for jump-to-message */
	.message-highlight {
		animation: highlight-flash 1.5s ease-out;
	}
	
	@keyframes highlight-flash {
		0% { background-color: rgba(239, 68, 68, 0.3); }
		100% { background-color: transparent; }
	}
</style>
