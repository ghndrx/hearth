<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { messages, activeChannel, websocket } from '$stores';
	import type { Message } from '$lib/types';

	let messageInput = '';
	let messagesContainer: HTMLDivElement;
	let unsubscribeWS: (() => void) | null = null;

	$: channelMessages = $activeChannel
		? $messages.messages.get($activeChannel.id) || []
		: [];

	$: if ($activeChannel) {
		messages.loadMessages($activeChannel.id);
	}

	onMount(() => {
		unsubscribeWS = websocket.on('MESSAGE_CREATE', (event) => {
			const message = event.data as Message;
			messages.addMessage(message);
			scrollToBottom();
		});
	});

	onDestroy(() => {
		unsubscribeWS?.();
	});

	function scrollToBottom() {
		setTimeout(() => {
			if (messagesContainer) {
				messagesContainer.scrollTop = messagesContainer.scrollHeight;
			}
		}, 0);
	}

	async function sendMessage() {
		if (!messageInput.trim() || !$activeChannel) return;

		const content = messageInput.trim();
		messageInput = '';
		await messages.sendMessage($activeChannel.id, content);
	}

	function handleKeydown(event: KeyboardEvent) {
		if (event.key === 'Enter' && !event.shiftKey) {
			event.preventDefault();
			sendMessage();
		}
	}

	function formatTime(dateStr: string): string {
		const date = new Date(dateStr);
		return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
	}

	function formatDate(dateStr: string): string {
		const date = new Date(dateStr);
		const today = new Date();
		const yesterday = new Date(today);
		yesterday.setDate(yesterday.getDate() - 1);

		if (date.toDateString() === today.toDateString()) {
			return 'Today';
		} else if (date.toDateString() === yesterday.toDateString()) {
			return 'Yesterday';
		}
		return date.toLocaleDateString();
	}

	function shouldShowDateDivider(messages: Message[], index: number): boolean {
		if (index === 0) return true;
		const current = new Date(messages[index].createdAt).toDateString();
		const previous = new Date(messages[index - 1].createdAt).toDateString();
		return current !== previous;
	}

	function shouldGroupWithPrevious(messages: Message[], index: number): boolean {
		if (index === 0) return false;
		const current = messages[index];
		const previous = messages[index - 1];

		// Same author and within 7 minutes
		if (current.authorId !== previous.authorId) return false;
		const timeDiff = new Date(current.createdAt).getTime() - new Date(previous.createdAt).getTime();
		return timeDiff < 7 * 60 * 1000;
	}
</script>

<div class="flex-1 flex flex-col bg-dark-700 min-w-0">
	{#if $activeChannel}
		<!-- Channel header -->
		<header class="h-12 px-4 flex items-center border-b border-dark-900 shadow-sm shrink-0">
			<span class="text-xl text-gray-400 mr-2">#</span>
			<h1 class="font-semibold text-white">{$activeChannel.name}</h1>
			{#if $activeChannel.topic}
				<div class="mx-4 w-px h-6 bg-dark-500"></div>
				<p class="text-sm text-gray-400 truncate">{$activeChannel.topic}</p>
			{/if}
		</header>

		<!-- Messages -->
		<div
			class="flex-1 overflow-y-auto px-4 py-4"
			bind:this={messagesContainer}
		>
			{#each channelMessages as message, index (message.id)}
				{#if shouldShowDateDivider(channelMessages, index)}
					<div class="flex items-center my-4">
						<div class="flex-1 h-px bg-dark-500"></div>
						<span class="px-4 text-xs text-gray-500 font-medium">
							{formatDate(message.createdAt)}
						</span>
						<div class="flex-1 h-px bg-dark-500"></div>
					</div>
				{/if}

				<div
					class="group hover:bg-dark-600/30 -mx-4 px-4 py-0.5"
					class:mt-4={!shouldGroupWithPrevious(channelMessages, index)}
				>
					{#if !shouldGroupWithPrevious(channelMessages, index)}
						<div class="flex items-start gap-4">
							<div class="w-10 h-10 rounded-full bg-hearth-500 flex items-center justify-center shrink-0">
								<span class="text-sm font-medium text-white">
									{message.author?.username?.charAt(0).toUpperCase() || 'U'}
								</span>
							</div>
							<div class="flex-1 min-w-0">
								<div class="flex items-baseline gap-2">
									<span class="font-medium text-white hover:underline cursor-pointer">
										{message.author?.username || 'Unknown'}
									</span>
									<span class="text-xs text-gray-500">
										{formatTime(message.createdAt)}
									</span>
								</div>
								<p class="text-gray-200 break-words">{message.content}</p>
							</div>
						</div>
					{:else}
						<div class="flex items-start gap-4">
							<span class="w-10 text-center text-xs text-gray-600 opacity-0 group-hover:opacity-100 pt-1">
								{formatTime(message.createdAt)}
							</span>
							<p class="text-gray-200 break-words">{message.content}</p>
						</div>
					{/if}
				</div>
			{/each}

			{#if channelMessages.length === 0 && !$messages.loading}
				<div class="flex flex-col items-center justify-center h-full text-center">
					<div class="w-16 h-16 rounded-full bg-dark-600 flex items-center justify-center mb-4">
						<span class="text-4xl">#</span>
					</div>
					<h3 class="text-2xl font-bold text-white mb-2">
						Welcome to #{$activeChannel.name}!
					</h3>
					<p class="text-gray-400">
						This is the start of the #{$activeChannel.name} channel.
					</p>
				</div>
			{/if}
		</div>

		<!-- Message input -->
		<div class="px-4 pb-6 shrink-0">
			<div class="bg-dark-600 rounded-lg flex items-end">
				<button class="p-3 text-gray-400 hover:text-gray-200">
					<svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
							d="M12 4v16m8-8H4" />
					</svg>
				</button>
				<textarea
					class="flex-1 bg-transparent border-0 resize-none py-3 px-2 text-gray-100
					       placeholder-gray-500 focus:ring-0 max-h-48"
					placeholder="Message #{$activeChannel.name}"
					rows="1"
					bind:value={messageInput}
					on:keydown={handleKeydown}
				></textarea>
				<div class="flex items-center gap-1 p-2">
					<button class="p-1.5 text-gray-400 hover:text-gray-200">
						<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
								d="M14.828 14.828a4 4 0 01-5.656 0M9 10h.01M15 10h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
						</svg>
					</button>
				</div>
			</div>
		</div>
	{:else}
		<!-- No channel selected -->
		<div class="flex flex-col items-center justify-center h-full text-center p-8">
			<div class="w-24 h-24 rounded-full bg-dark-600 flex items-center justify-center mb-6">
				<svg class="w-12 h-12 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
						d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
				</svg>
			</div>
			<h2 class="text-2xl font-bold text-white mb-2">Welcome to Hearth</h2>
			<p class="text-gray-400 max-w-md">
				Select a server from the sidebar and choose a channel to start chatting.
			</p>
		</div>
	{/if}
</div>
