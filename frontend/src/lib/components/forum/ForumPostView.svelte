<script lang="ts">
	import { createEventDispatcher, onMount } from 'svelte';
	import { api } from '$lib/api';
	import { forumTags, type ForumTag } from '$lib/stores/forumTags';
	import ForumTagBadge from './ForumTagBadge.svelte';

	const dispatch = createEventDispatcher<{
		back: void;
	}>();

	export let channelId: string;
	export let threadId: string;

	interface ThreadPost {
		id: string;
		name: string;
		owner_id: string;
		owner?: {
			id: string;
			username: string;
			display_name?: string;
			avatar?: string;
		};
		message_count: number;
		applied_tags: string[];
		created_at: string;
		is_pinned: boolean;
		archived: boolean;
	}

	interface ThreadMessage {
		id: string;
		content: string;
		author_id: string;
		author?: {
			id: string;
			username: string;
			display_name?: string;
			avatar?: string;
		};
		created_at: string;
		edited_at?: string;
	}

	let thread: ThreadPost | null = null;
	let messages: ThreadMessage[] = [];
	let tags: ForumTag[] = [];
	let loading = false;
	let messagesLoading = false;
	let error = '';
	let sendError = '';
	let newMessage = '';
	let sending = false;
	let inputEl: HTMLTextAreaElement | null = null;

	onMount(async () => {
		await Promise.all([loadThread(), loadMessages()]);
	});

	async function loadThread() {
		loading = true;
		error = '';
		try {
			thread = await api.get<ThreadPost>(`/threads/${threadId}`);
			// Load tags for the thread
			const tagResponse = await api.get<{ tags: ForumTag[] }>(`/threads/${threadId}/tags`);
			tags = tagResponse.tags || [];
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load post';
			console.error('[ForumPostView] Failed to load thread:', err);
		} finally {
			loading = false;
		}
	}

	async function loadMessages() {
		messagesLoading = true;
		try {
			const response = await api.get<{ messages: ThreadMessage[] }>(
				`/threads/${threadId}/messages?limit=50`
			);
			messages = response.messages || [];
		} catch (err) {
			console.error('[ForumPostView] Failed to load messages:', err);
		} finally {
			messagesLoading = false;
		}
	}

	async function handleSendMessage() {
		if (!newMessage.trim() || sending) return;

		sending = true;
		sendError = '';

		try {
			const msg = await api.post<ThreadMessage>(`/threads/${threadId}/messages`, {
				content: newMessage.trim()
			});
			messages = [...messages, msg];
			newMessage = '';
			// Scroll to bottom
			setTimeout(() => {
				const container = document.querySelector('.post-messages');
				if (container) container.scrollTop = container.scrollHeight;
			}, 50);
		} catch (err) {
			sendError = err instanceof Error ? err.message : 'Failed to send message';
		} finally {
			sending = false;
		}
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter' && !e.shiftKey) {
			e.preventDefault();
			handleSendMessage();
		}
	}

	function formatTimeAgo(dateStr: string): string {
		const date = new Date(dateStr);
		const now = new Date();
		const diff = now.getTime() - date.getTime();
		const minutes = Math.floor(diff / 60000);
		const hours = Math.floor(diff / 3600000);
		const days = Math.floor(diff / 86400000);

		if (minutes < 1) return 'just now';
		if (minutes < 60) return `${minutes}m ago`;
		if (hours < 24) return `${hours}h ago`;
		if (days < 7) return `${days}d ago`;
		return date.toLocaleDateString();
	}

	function getTagById(tagId: string): ForumTag | undefined {
		return tags.find(t => t.id === tagId);
	}
</script>

<div class="forum-post-view">
	<!-- Back button and post header -->
	<div class="post-header">
		<button class="back-btn" on:click={() => dispatch('back')}>
			← Back to posts
		</button>
	</div>

	{#if loading}
		<div class="loading-state">
			<div class="spinner"></div>
			<span>Loading post...</span>
		</div>
	{:else if error}
		<div class="error-state">
			<span class="error-icon">⚠️</span>
			<span class="error-text">{error}</span>
			<button class="retry-btn" on:click={loadThread}>Retry</button>
		</div>
	{:else if thread}
		<!-- Thread info banner -->
		<div class="thread-banner">
			<div class="thread-title-row">
				{#if thread.is_pinned}
					<span class="pin-icon" title="Pinned">📌</span>
				{/if}
				<h1 class="thread-title">{thread.name}</h1>
				{#if thread.archived}
					<span class="archived-badge">Archived</span>
				{/if}
			</div>

			<!-- Applied tags -->
			{#if thread.applied_tags && thread.applied_tags.length > 0}
				<div class="thread-tags">
					{#each thread.applied_tags as tagId}
						{@const tag = getTagById(tagId)}
						{#if tag}
							<ForumTagBadge {tag} size="md" />
						{/if}
					{/each}
				</div>
			{/if}

			<!-- Thread meta -->
			<div class="thread-meta">
				<span class="thread-author">
					{#if thread.owner}
						<span class="author-avatar">
							{thread.owner.avatar ? '🧑' : '👤'}
						</span>
						<span class="author-name">
							{thread.owner.display_name || thread.owner.username}
						</span>
					{/if}
				</span>
				<span class="thread-time">{formatTimeAgo(thread.created_at)}</span>
				<span class="thread-replies">💬 {thread.message_count} replies</span>
			</div>
		</div>

		<!-- Messages container -->
		<div class="post-messages">
			{#if messagesLoading}
				<div class="messages-loading">
					<div class="spinner"></div>
				</div>
			{:else if messages.length === 0}
				<div class="no-messages">
					<span>No replies yet. Be the first to reply!</span>
				</div>
			{:else}
				{#each messages as msg (msg.id)}
					<div class="message-item">
						<div class="message-avatar">
							{msg.author?.avatar ? '🧑' : '👤'}
						</div>
						<div class="message-body">
							<div class="message-header">
								<span class="message-author">
									{msg.author?.display_name || msg.author?.username || 'Unknown'}
								</span>
								<span class="message-time">{formatTimeAgo(msg.created_at)}</span>
								{#if msg.edited_at}
									<span class="message-edited">(edited)</span>
								{/if}
							</div>
							<div class="message-content">{msg.content}</div>
						</div>
					</div>
				{/each}
			{/if}
		</div>

		<!-- Reply input -->
		{#if !thread.archived}
			<div class="reply-area">
				{#if sendError}
					<div class="send-error" role="alert">{sendError}</div>
				{/if}
				<div class="reply-input-wrapper">
					<textarea
						bind:this={inputEl}
						bind:value={newMessage}
						on:keydown={handleKeydown}
						placeholder="Write a reply..."
						rows="2"
						maxlength="2000"
						disabled={sending}
						class="reply-textarea"
					></textarea>
					<button
						class="send-btn"
						on:click={handleSendMessage}
						disabled={!newMessage.trim() || sending}
					>
						{sending ? '...' : 'Send'}
					</button>
				</div>
			</div>
		{:else}
			<div class="archived-notice">
				This post is archived and no longer accepting replies.
			</div>
		{/if}
	{/if}
</div>

<style>
	.forum-post-view {
		display: flex;
		flex-direction: column;
		height: 100%;
		overflow: hidden;
	}

	.post-header {
		padding: 8px 16px;
		border-bottom: 1px solid var(--border-color, #4f545c);
		background: var(--bg-secondary, #2b2d31);
		flex-shrink: 0;
	}

	.back-btn {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		padding: 6px 12px;
		background: transparent;
		border: 1px solid var(--border-color, #4f545c);
		border-radius: var(--radius-sm, 4px);
		color: var(--text-primary, #f2f3f5);
		font-size: var(--font-size-sm, 13px);
		cursor: pointer;
		transition: background 0.15s, border-color 0.15s;
	}

	.back-btn:hover {
		background: var(--bg-modifier-accent, #4e5058);
		border-color: var(--text-muted, #b5bac1);
	}

	.loading-state,
	.error-state,
	.messages-loading {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 12px;
		flex: 1;
		color: var(--text-muted, #b5bac1);
	}

	.error-state {
		color: var(--text-danger, #ed4245);
	}

	.spinner {
		width: 32px;
		height: 32px;
		border: 3px solid var(--border-color, #4f545c);
		border-top-color: var(--brand-primary, #5865f2);
		border-radius: 50%;
		animation: spin 0.8s linear infinite;
	}

	@keyframes spin {
		to { transform: rotate(360deg); }
	}

	.retry-btn {
		padding: 6px 16px;
		background: var(--brand-primary, #5865f2);
		border: none;
		border-radius: var(--radius-sm, 4px);
		color: white;
		font-size: var(--font-size-sm, 13px);
		cursor: pointer;
	}

	/* Thread banner */
	.thread-banner {
		padding: 16px 20px;
		border-bottom: 1px solid var(--border-color, #4f545c);
		background: var(--bg-secondary, #2b2d31);
		flex-shrink: 0;
	}

	.thread-title-row {
		display: flex;
		align-items: center;
		gap: 8px;
		margin-bottom: 8px;
	}

	.pin-icon {
		font-size: 16px;
	}

	.thread-title {
		margin: 0;
		font-size: 20px;
		font-weight: 700;
		color: var(--text-primary, #f2f3f5);
		word-break: break-word;
	}

	.archived-badge {
		font-size: var(--font-size-xs, 10px);
		padding: 2px 8px;
		background: var(--bg-modifier-accent, #4e5058);
		border-radius: 3px;
		color: var(--text-muted, #b5bac1);
		text-transform: uppercase;
		font-weight: 600;
	}

	.thread-tags {
		display: flex;
		flex-wrap: wrap;
		gap: 6px;
		margin-bottom: 10px;
	}

	.thread-meta {
		display: flex;
		align-items: center;
		gap: 12px;
		font-size: var(--font-size-sm, 13px);
		color: var(--text-muted, #b5bac1);
	}

	.thread-author {
		display: flex;
		align-items: center;
		gap: 4px;
	}

	.author-avatar {
		font-size: 16px;
	}

	.author-name {
		font-weight: 500;
		color: var(--text-primary, #f2f3f5);
	}

	/* Messages */
	.post-messages {
		flex: 1;
		overflow-y: auto;
		padding: 12px 16px;
		display: flex;
		flex-direction: column;
		gap: 2px;
	}

	.no-messages {
		display: flex;
		align-items: center;
		justify-content: center;
		flex: 1;
		color: var(--text-muted, #b5bac1);
		font-style: italic;
	}

	.message-item {
		display: flex;
		gap: 12px;
		padding: 8px;
		border-radius: var(--radius-sm, 4px);
		transition: background 0.1s;
	}

	.message-item:hover {
		background: var(--bg-modifier-accent, rgba(79, 84, 92, 0.3));
	}

	.message-avatar {
		width: 36px;
		height: 36px;
		border-radius: 50%;
		background: var(--bg-modifier-accent, #4e5058);
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 18px;
		flex-shrink: 0;
	}

	.message-body {
		flex: 1;
		min-width: 0;
	}

	.message-header {
		display: flex;
		align-items: baseline;
		gap: 8px;
		margin-bottom: 2px;
	}

	.message-author {
		font-weight: 600;
		color: var(--text-primary, #f2f3f5);
		font-size: var(--font-size-sm, 13px);
	}

	.message-time {
		font-size: var(--font-size-xs, 11px);
		color: var(--text-muted, #b5bac1);
	}

	.message-edited {
		font-size: var(--font-size-xs, 10px);
		color: var(--text-muted, #b5bac1);
		font-style: italic;
	}

	.message-content {
		font-size: var(--font-size-md, 14px);
		color: var(--text-primary, #f2f3f5);
		white-space: pre-wrap;
		word-break: break-word;
		line-height: 1.5;
	}

	/* Reply area */
	.reply-area {
		padding: 12px 16px;
		border-top: 1px solid var(--border-color, #4f545c);
		background: var(--bg-secondary, #2b2d31);
		flex-shrink: 0;
	}

	.send-error {
		padding: 8px 12px;
		margin-bottom: 8px;
		background: rgba(237, 66, 69, 0.15);
		border: 1px solid var(--text-danger, #ed4245);
		border-radius: var(--radius-sm, 4px);
		color: var(--text-danger, #ed4245);
		font-size: var(--font-size-sm, 13px);
	}

	.reply-input-wrapper {
		display: flex;
		gap: 8px;
		align-items: flex-end;
	}

	.reply-textarea {
		flex: 1;
		padding: 10px 12px;
		background: var(--bg-primary, #36393f);
		border: 1px solid var(--border-color, #4f545c);
		border-radius: var(--radius-sm, 4px);
		color: var(--text-primary, #f2f3f5);
		font-size: var(--font-size-sm, 14px);
		font-family: inherit;
		resize: none;
		transition: border-color 0.15s;
	}

	.reply-textarea:focus {
		outline: none;
		border-color: var(--brand-primary, #5865f2);
	}

	.reply-textarea:disabled {
		opacity: 0.6;
		cursor: not-allowed;
	}

	.send-btn {
		padding: 10px 20px;
		background: var(--brand-primary, #5865f2);
		border: none;
		border-radius: var(--radius-sm, 4px);
		color: white;
		font-size: var(--font-size-sm, 14px);
		font-weight: 500;
		cursor: pointer;
		transition: background 0.15s, opacity 0.15s;
		white-space: nowrap;
	}

	.send-btn:hover:not(:disabled) {
		background: var(--brand-hover, #4752c4);
	}

	.send-btn:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.archived-notice {
		padding: 12px 16px;
		border-top: 1px solid var(--border-color, #4f545c);
		background: var(--bg-secondary, #2b2d31);
		color: var(--text-muted, #b5bac1);
		font-size: var(--font-size-sm, 13px);
		text-align: center;
		flex-shrink: 0;
	}
</style>
