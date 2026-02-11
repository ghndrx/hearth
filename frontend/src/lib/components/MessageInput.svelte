<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import { currentChannel } from '$lib/stores/channels';
	import { sendTypingIndicator } from '$lib/stores/messages';
	
	const dispatch = createEventDispatcher();
	
	let content = '';
	let files: FileList | null = null;
	let textarea: HTMLTextAreaElement;
	let lastTypingTime = 0;
	
	$: placeholder = getPlaceholder($currentChannel);
	
	function getPlaceholder(channel: any) {
		if (!channel) return 'Select a channel';
		if (channel.type === 1) {
			return `Message @${channel.recipients?.[0]?.username || 'Unknown'}`;
		}
		if (channel.type === 3) {
			return `Message ${channel.name || 'group'}`;
		}
		return `Message #${channel.name}`;
	}
	
	function handleInput() {
		autoResize();
		handleTyping();
	}
	
	function autoResize() {
		textarea.style.height = 'auto';
		textarea.style.height = Math.min(textarea.scrollHeight, 300) + 'px';
	}
	
	function handleTyping() {
		const now = Date.now();
		if (now - lastTypingTime > 3000) {
			lastTypingTime = now;
			if ($currentChannel) {
				sendTypingIndicator($currentChannel.id);
			}
		}
	}
	
	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter' && !e.shiftKey) {
			e.preventDefault();
			send();
		}
	}
	
	function send() {
		if (!content.trim() && !files?.length) return;
		
		dispatch('send', {
			content: content.trim(),
			attachments: files ? Array.from(files) : []
		});
		
		content = '';
		files = null;
		textarea.style.height = 'auto';
	}
	
	function handlePaste(e: ClipboardEvent) {
		const items = e.clipboardData?.items;
		if (!items) return;
		
		for (const item of items) {
			if (item.type.startsWith('image/')) {
				const file = item.getAsFile();
				if (file) {
					const dt = new DataTransfer();
					if (files) {
						for (const f of files) dt.items.add(f);
					}
					dt.items.add(file);
					files = dt.files;
				}
			}
		}
	}
	
	function removeFile(index: number) {
		if (!files) return;
		const dt = new DataTransfer();
		for (let i = 0; i < files.length; i++) {
			if (i !== index) dt.items.add(files[i]);
		}
		files = dt.files;
	}
</script>

<div class="message-input-container">
	{#if files?.length}
		<div class="attachments-preview">
			{#each Array.from(files) as file, i}
				<div class="attachment-preview">
					{#if file.type.startsWith('image/')}
						<img src={URL.createObjectURL(file)} alt={file.name} />
					{:else}
						<div class="file-preview">
							<span class="file-icon">📎</span>
							<span class="file-name">{file.name}</span>
						</div>
					{/if}
					<button class="remove-attachment" on:click={() => removeFile(i)}>×</button>
				</div>
			{/each}
		</div>
	{/if}
	
	<div class="input-wrapper">
		<button class="attach-btn" title="Attach file">
			<label>
				<svg viewBox="0 0 24 24" width="24" height="24" fill="currentColor">
					<path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm5 11h-4v4h-2v-4H7v-2h4V7h2v4h4v2z"/>
				</svg>
				<input type="file" multiple hidden bind:files />
			</label>
		</button>
		
		<textarea
			bind:this={textarea}
			bind:value={content}
			on:input={handleInput}
			on:keydown={handleKeydown}
			on:paste={handlePaste}
			{placeholder}
			rows="1"
			disabled={!$currentChannel}
		></textarea>
		
		<div class="input-buttons">
			<button class="emoji-btn" title="Emoji">
				<svg viewBox="0 0 24 24" width="24" height="24" fill="currentColor">
					<path d="M11.99 2C6.47 2 2 6.48 2 12s4.47 10 9.99 10C17.52 22 22 17.52 22 12S17.52 2 11.99 2zM12 20c-4.42 0-8-3.58-8-8s3.58-8 8-8 8 3.58 8 8-3.58 8-8 8zm3.5-9c.83 0 1.5-.67 1.5-1.5S16.33 8 15.5 8 14 8.67 14 9.5s.67 1.5 1.5 1.5zm-7 0c.83 0 1.5-.67 1.5-1.5S9.33 8 8.5 8 7 8.67 7 9.5 7.67 11 8.5 11zm3.5 6.5c2.33 0 4.31-1.46 5.11-3.5H6.89c.8 2.04 2.78 3.5 5.11 3.5z"/>
				</svg>
			</button>
			
			{#if $currentChannel?.e2ee_enabled}
				<span class="e2ee-indicator" title="End-to-End Encrypted">🔒</span>
			{/if}
		</div>
	</div>
</div>

<style>
	.message-input-container {
		padding: 0 16px 24px;
	}
	
	.attachments-preview {
		display: flex;
		flex-wrap: wrap;
		gap: 8px;
		padding: 8px;
		background: var(--bg-secondary);
		border-radius: 8px 8px 0 0;
	}
	
	.attachment-preview {
		position: relative;
		max-width: 200px;
	}
	
	.attachment-preview img {
		max-width: 200px;
		max-height: 150px;
		border-radius: 4px;
	}
	
	.file-preview {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 8px 12px;
		background: var(--bg-tertiary);
		border-radius: 4px;
	}
	
	.remove-attachment {
		position: absolute;
		top: -8px;
		right: -8px;
		width: 24px;
		height: 24px;
		border-radius: 50%;
		background: var(--status-danger);
		border: none;
		color: white;
		font-size: 16px;
		cursor: pointer;
		display: flex;
		align-items: center;
		justify-content: center;
	}
	
	.input-wrapper {
		display: flex;
		align-items: flex-end;
		gap: 8px;
		background: var(--bg-tertiary);
		border-radius: 8px;
		padding: 4px;
	}
	
	.attachments-preview + .input-wrapper {
		border-radius: 0 0 8px 8px;
	}
	
	.attach-btn {
		background: none;
		border: none;
		color: var(--text-muted);
		cursor: pointer;
		padding: 8px;
		border-radius: 4px;
	}
	
	.attach-btn:hover {
		color: var(--text-primary);
	}
	
	.attach-btn label {
		cursor: pointer;
		display: flex;
	}
	
	textarea {
		flex: 1;
		background: none;
		border: none;
		color: var(--text-primary);
		font-size: 16px;
		padding: 10px 0;
		resize: none;
		max-height: 300px;
		line-height: 1.4;
	}
	
	textarea::placeholder {
		color: var(--text-muted);
	}
	
	textarea:focus {
		outline: none;
	}
	
	textarea:disabled {
		cursor: not-allowed;
	}
	
	.input-buttons {
		display: flex;
		align-items: center;
		gap: 4px;
		padding: 4px;
	}
	
	.emoji-btn {
		background: none;
		border: none;
		color: var(--text-muted);
		cursor: pointer;
		padding: 4px;
		border-radius: 4px;
	}
	
	.emoji-btn:hover {
		color: var(--text-primary);
	}
	
	.e2ee-indicator {
		font-size: 14px;
		padding: 4px;
	}
</style>
