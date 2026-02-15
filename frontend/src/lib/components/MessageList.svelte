<script lang="ts">
	import { afterUpdate } from 'svelte';
	import { messages, loadMessages, sendMessage } from '$lib/stores/messages';
	import { currentChannel } from '$lib/stores/channels';
	import { user } from '$lib/stores/auth';
	import { currentServer } from '$lib/stores/servers';
	import MessageGroup from './MessageGroup.svelte';
	import MessageInput from './MessageInput.svelte';

	let replyTo: { id: string; content: string; author: { username: string } } | null = null;

	async function handleSend(event: CustomEvent<{ content: string; attachments: File[]; replyTo?: string }>) {
		if (!$currentChannel) return;
		
		const { content, attachments, replyTo: replyToId } = event.detail;
		
		try {
			await sendMessage($currentChannel.id, content, attachments, replyToId);
			replyTo = null;
		} catch (error) {
			console.error('Failed to send message:', error);
		}
	}

	function handleReply(event: CustomEvent<{ message: { id: string; content: string; author?: { username: string } } }>) {
		const message = event.detail.message;
		replyTo = {
			id: message.id,
			content: message.content,
			author: message.author || { username: 'Unknown' }
		};
	}

	function handleReact(event: CustomEvent<{ messageId: string; emoji: string }>) {
		// TODO: Implement reaction functionality
		console.log('React:', event.detail);
	}

	function handleEdit(event: CustomEvent<{ messageId: string; content: string }>) {
		// TODO: Implement edit functionality
		console.log('Edit:', event.detail);
	}

	function handleDelete(event: CustomEvent<{ messageId: string }>) {
		// TODO: Implement delete functionality
		console.log('Delete:', event.detail);
	}

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

	// Convert messages to MessageGroup format with date dividers
	$: groupedMessagesWithDates = channelMessages.reduce((acc, message, index) => {
		// Add date divider if needed
		if (shouldShowDate(index)) {
			acc.push({
				type: 'date' as const,
				date: formatDate(message.created_at)
			});
		}
		// Add message
		acc.push({
			type: 'message' as const,
			message
		});
		return acc;
	}, [] as Array<{ type: 'date'; date: string } | { type: 'message'; message: typeof channelMessages[0] }>);
</script>

<div class="flex-1 flex flex-col min-w-0">
	<!-- Channel Header -->
	<div class="h-12 px-4 flex items-center border-b border-[#1e1f22] bg-[#313338] shrink-0">
		{#if $currentChannel}
			{#if $currentChannel.type === 1 || $currentChannel.type === 3}
				<!-- DM Header -->
				<div class="flex items-center gap-3">
					<div
						class="w-6 h-6 rounded-full bg-[#5865f2] flex items-center justify-center overflow-hidden"
					>
						{#if $currentChannel.recipients?.[0]?.avatar}
							<img
								src={$currentChannel.recipients[0].avatar}
								alt=""
								class="w-full h-full object-cover"
							/>
						{:else}
							<span class="text-white text-xs font-medium">
								{($currentChannel.recipients?.[0]?.username || '?')[0].toUpperCase()}
							</span>
						{/if}
					</div>
					<span class="text-[#f2f3f5] font-semibold text-base">
						{$currentChannel.recipients?.[0]?.display_name ||
							$currentChannel.recipients?.[0]?.username ||
							'Unknown'}
					</span>
				</div>
			{:else}
				<!-- Server Channel Header -->
				<div class="flex items-center gap-2">
					<span class="text-[#b5bac1] text-2xl font-light">#</span>
					<span class="text-[#f2f3f5] font-semibold text-base">{$currentChannel.name}</span>
				</div>
				{#if $currentChannel.topic}
					<div class="ml-4 px-2 text-[#b5bac1] text-sm truncate">{$currentChannel.topic}</div>
				{/if}
			{/if}
		{:else}
			<span class="text-[#949ba4] text-base">Select a channel</span>
		{/if}
	</div>

	<!-- Messages -->
	<div class="flex-1 overflow-y-auto px-4" bind:this={messageContainer} on:scroll={handleScroll}>
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

			<!-- Message Groups with Date Dividers -->
			{#each groupedMessagesWithDates as item}
				{#if item.type === 'date'}
					<div class="flex items-center my-4">
						<div class="flex-1 h-px bg-[#3f4147]"></div>
						<span class="px-2 text-xs font-semibold text-[#949ba4] uppercase">{item.date}</span>
						<div class="flex-1 h-px bg-[#3f4147]"></div>
					</div>
				{:else}
					<!-- Use MessageGroup for proper message grouping -->
					<MessageGroup
						messages={[item.message]}
						currentUserId={$user?.id}
						on:react={handleReact}
						on:edit={handleEdit}
						on:delete={handleDelete}
						on:reply={handleReply}
					/>
				{/if}
			{/each}
		{:else}
			<div class="flex items-center justify-center h-full">
				<p class="text-[#949ba4] text-base">Select a channel to start chatting</p>
			</div>
		{/if}
	</div>

	<!-- Message Input -->
	{#if $currentChannel}
		<MessageInput 
			{replyTo}
			on:send={handleSend}
		/>
	{/if}
</div>
