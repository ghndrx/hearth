<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	
	export let message: any;
	export let grouped = false;
	export let isOwn = false;
	
	const dispatch = createEventDispatcher();
	
	let showActions = false;
	let editing = false;
	let editContent = message.content;
	
	function formatTime(date: string) {
		return new Date(date).toLocaleTimeString(undefined, {
			hour: 'numeric',
			minute: '2-digit'
		});
	}
	
	function handleReaction(emoji: string) {
		dispatch('react', { messageId: message.id, emoji });
	}
	
	function startEdit() {
		editing = true;
		editContent = message.content;
	}
	
	function cancelEdit() {
		editing = false;
		editContent = message.content;
	}
	
	function saveEdit() {
		dispatch('edit', { messageId: message.id, content: editContent });
		editing = false;
	}
	
	function handleDelete() {
		if (confirm('Delete this message?')) {
			dispatch('delete', { messageId: message.id });
		}
	}
	
	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') cancelEdit();
		if (e.key === 'Enter' && !e.shiftKey) {
			e.preventDefault();
			saveEdit();
		}
	}
</script>

<div 
	class="message"
	class:grouped
	class:own={isOwn}
	on:mouseenter={() => showActions = true}
	on:mouseleave={() => showActions = false}
>
	{#if !grouped}
		<div class="avatar">
			{#if message.author?.avatar}
				<img src={message.author.avatar} alt={message.author.username} />
			{:else}
				<div class="avatar-placeholder">
					{(message.author?.username || '?')[0].toUpperCase()}
				</div>
			{/if}
		</div>
	{:else}
		<div class="timestamp-gutter">
			<span class="hover-timestamp">{formatTime(message.created_at)}</span>
		</div>
	{/if}
	
	<div class="content">
		{#if !grouped}
			<div class="header">
				<span class="author" style="color: {message.author?.role_color || 'inherit'}">
					{message.author?.display_name || message.author?.username || 'Unknown'}
				</span>
				<span class="timestamp">{formatTime(message.created_at)}</span>
				{#if message.edited_at}
					<span class="edited" title="Edited {new Date(message.edited_at).toLocaleString()}">(edited)</span>
				{/if}
				{#if message.encrypted}
					<span class="encrypted-badge" title="End-to-End Encrypted">🔒</span>
				{/if}
			</div>
		{/if}
		
		{#if message.reply_to}
			<div class="reply-context">
				<span class="reply-icon">↳</span>
				<span class="reply-author">@{message.reply_to_author?.username}</span>
				<span class="reply-content">{message.reply_to_content}</span>
			</div>
		{/if}
		
		{#if editing}
			<div class="edit-container">
				<textarea
					bind:value={editContent}
					on:keydown={handleKeydown}
					autofocus
				></textarea>
				<div class="edit-actions">
					<span>escape to <button on:click={cancelEdit}>cancel</button> • enter to <button on:click={saveEdit}>save</button></span>
				</div>
			</div>
		{:else}
			<div class="body">
				{@html parseMessage(message.content)}
			</div>
		{/if}
		
		{#if message.attachments?.length > 0}
			<div class="attachments">
				{#each message.attachments as attachment}
					{#if attachment.content_type?.startsWith('image/')}
						<img 
							src={attachment.url} 
							alt={attachment.filename}
							class="attachment-image"
						/>
					{:else}
						<a href={attachment.url} class="attachment-file" download>
							<span class="file-icon">📎</span>
							<span class="file-name">{attachment.filename}</span>
							<span class="file-size">{formatSize(attachment.size)}</span>
						</a>
					{/if}
				{/each}
			</div>
		{/if}
		
		{#if message.reactions?.length > 0}
			<div class="reactions">
				{#each message.reactions as reaction}
					<button 
						class="reaction"
						class:reacted={reaction.me}
						on:click={() => handleReaction(reaction.emoji)}
					>
						<span class="emoji">{reaction.emoji}</span>
						<span class="count">{reaction.count}</span>
					</button>
				{/each}
			</div>
		{/if}
	</div>
	
	{#if showActions && !editing}
		<div class="actions">
			<button title="Add Reaction" on:click={() => handleReaction('👍')}>😀</button>
			<button title="Reply">↩️</button>
			{#if isOwn}
				<button title="Edit" on:click={startEdit}>✏️</button>
			{/if}
			{#if isOwn}
				<button title="Delete" on:click={handleDelete}>🗑️</button>
			{/if}
			<button title="More">⋯</button>
		</div>
	{/if}
</div>

<script context="module" lang="ts">
	function parseMessage(content: string): string {
		if (!content) return '';
		
		// Escape HTML
		let html = content
			.replace(/&/g, '&amp;')
			.replace(/</g, '&lt;')
			.replace(/>/g, '&gt;');
		
		// Parse markdown-like formatting
		html = html
			// Bold
			.replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>')
			// Italic
			.replace(/\*(.+?)\*/g, '<em>$1</em>')
			// Code
			.replace(/`(.+?)`/g, '<code>$1</code>')
			// Links
			.replace(/(https?:\/\/[^\s]+)/g, '<a href="$1" target="_blank" rel="noopener">$1</a>')
			// Newlines
			.replace(/\n/g, '<br>');
		
		return html;
	}
	
	function formatSize(bytes: number): string {
		if (bytes < 1024) return bytes + ' B';
		if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
		return (bytes / (1024 * 1024)).toFixed(1) + ' MB';
	}
</script>

<style>
	.message {
		display: flex;
		padding: 2px 48px 2px 72px;
		position: relative;
		margin-top: 16px;
	}
	
	.message.grouped {
		margin-top: 0;
	}
	
	.message:hover {
		background: var(--bg-modifier-hover);
	}
	
	.avatar {
		position: absolute;
		left: 16px;
		width: 40px;
		height: 40px;
		border-radius: 50%;
		overflow: hidden;
		cursor: pointer;
	}
	
	.avatar img {
		width: 100%;
		height: 100%;
		object-fit: cover;
	}
	
	.avatar-placeholder {
		width: 100%;
		height: 100%;
		background: var(--brand-primary);
		display: flex;
		align-items: center;
		justify-content: center;
		color: white;
		font-weight: 600;
		font-size: 18px;
	}
	
	.timestamp-gutter {
		position: absolute;
		left: 16px;
		width: 40px;
		text-align: center;
	}
	
	.hover-timestamp {
		display: none;
		font-size: 11px;
		color: var(--text-muted);
	}
	
	.message:hover .hover-timestamp {
		display: block;
	}
	
	.content {
		flex: 1;
		min-width: 0;
	}
	
	.header {
		display: flex;
		align-items: baseline;
		gap: 8px;
		margin-bottom: 4px;
	}
	
	.author {
		font-weight: 600;
		cursor: pointer;
	}
	
	.author:hover {
		text-decoration: underline;
	}
	
	.timestamp {
		font-size: 12px;
		color: var(--text-muted);
	}
	
	.edited {
		font-size: 10px;
		color: var(--text-muted);
	}
	
	.encrypted-badge {
		font-size: 12px;
	}
	
	.reply-context {
		display: flex;
		align-items: center;
		gap: 4px;
		font-size: 14px;
		color: var(--text-muted);
		margin-bottom: 4px;
		padding: 4px 0;
	}
	
	.reply-icon {
		font-size: 12px;
	}
	
	.reply-author {
		color: var(--text-primary);
		font-weight: 500;
	}
	
	.reply-content {
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	
	.body {
		color: var(--text-primary);
		line-height: 1.4;
		word-wrap: break-word;
	}
	
	.body :global(code) {
		background: var(--bg-tertiary);
		padding: 2px 4px;
		border-radius: 3px;
		font-size: 14px;
	}
	
	.body :global(a) {
		color: var(--text-link);
	}
	
	.attachments {
		margin-top: 8px;
		display: flex;
		flex-wrap: wrap;
		gap: 4px;
	}
	
	.attachment-image {
		max-width: 400px;
		max-height: 300px;
		border-radius: 8px;
		cursor: pointer;
	}
	
	.attachment-file {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 10px;
		background: var(--bg-secondary);
		border-radius: 8px;
		color: var(--text-primary);
		text-decoration: none;
	}
	
	.reactions {
		display: flex;
		flex-wrap: wrap;
		gap: 4px;
		margin-top: 4px;
	}
	
	.reaction {
		display: flex;
		align-items: center;
		gap: 4px;
		padding: 2px 6px;
		background: var(--bg-tertiary);
		border: 1px solid transparent;
		border-radius: 8px;
		cursor: pointer;
		font-size: 14px;
	}
	
	.reaction:hover {
		border-color: var(--text-muted);
	}
	
	.reaction.reacted {
		background: rgba(88, 101, 242, 0.3);
		border-color: var(--brand-primary);
	}
	
	.count {
		color: var(--text-muted);
		font-size: 12px;
	}
	
	.actions {
		position: absolute;
		right: 16px;
		top: -16px;
		display: flex;
		gap: 4px;
		background: var(--bg-primary);
		border: 1px solid var(--bg-modifier-accent);
		border-radius: 4px;
		padding: 2px;
	}
	
	.actions button {
		background: none;
		border: none;
		cursor: pointer;
		padding: 4px 8px;
		border-radius: 4px;
		font-size: 16px;
	}
	
	.actions button:hover {
		background: var(--bg-modifier-hover);
	}
	
	.edit-container textarea {
		width: 100%;
		min-height: 44px;
		padding: 10px;
		background: var(--bg-tertiary);
		border: none;
		border-radius: 8px;
		color: var(--text-primary);
		font-size: 16px;
		resize: none;
	}
	
	.edit-actions {
		font-size: 12px;
		color: var(--text-muted);
		margin-top: 4px;
	}
	
	.edit-actions button {
		background: none;
		border: none;
		color: var(--text-link);
		cursor: pointer;
		padding: 0;
	}
</style>
