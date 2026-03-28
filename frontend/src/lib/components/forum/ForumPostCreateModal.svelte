<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import { api } from '$lib/api';
	import ForumTagPicker from './ForumTagPicker.svelte';

	export let channelId: string;
	export let show = false;

	const dispatch = createEventDispatcher<{
		close: void;
		created: {
			id: string;
			name: string;
			applied_tags: string[];
		};
	}>();

	let title = '';
	let content = '';
	let selectedTagIds: string[] = [];
	let loading = false;
	let error = '';

	$: if (!show) {
		// Reset form when modal closes
		title = '';
		content = '';
		selectedTagIds = [];
		error = '';
	}

	async function handleSubmit() {
		if (!title.trim() || loading) return;

		loading = true;
		error = '';

		try {
			// Create the thread/post first
			const thread = await api.post<{
				id: string;
				name: string;
				applied_tags: string[];
			}>(`/channels/${channelId}/threads`, {
				name: title.trim(),
				tag_ids: selectedTagIds
			});

			// If there's content, send it as the first message
			if (content.trim()) {
				await api.post(`/channels/${channelId}/messages`, {
					content: content.trim(),
					reply_to: thread.id
				});
			}

			dispatch('created', {
				id: thread.id,
				name: thread.name,
				applied_tags: selectedTagIds
			});
			dispatch('close');
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to create post';
		} finally {
			loading = false;
		}
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') {
			dispatch('close');
		}
	}
</script>

{#if show}
	<!-- svelte-ignore a11y-no-noninteractive-element-interactions -->
	<div
		class="modal-backdrop"
		on:click={() => dispatch('close')}
		on:keydown={handleKeydown}
		role="dialog"
		aria-modal="true"
		aria-labelledby="forum-post-title"
	>
		<div class="modal" on:click|stopPropagation role="document">
			<header class="modal-header">
				<h2 id="forum-post-title">Create Post</h2>
				<button
					class="close-btn"
					on:click={() => dispatch('close')}
					aria-label="Close"
					type="button"
				>
					×
				</button>
			</header>

			<form class="modal-body" on:submit|preventDefault={handleSubmit}>
				{#if error}
					<div class="error-banner" role="alert">{error}</div>
				{/if}

				<div class="form-group">
					<label for="post-title">Title <span class="required">*</span></label>
					<input
						id="post-title"
						type="text"
						bind:value={title}
						placeholder="Post title..."
						maxlength="100"
						required
						disabled={loading}
						autofocus
					/>
					<span class="char-count">{title.length}/100</span>
				</div>

				<div class="form-group">
					<label>Tags</label>
					<ForumTagPicker
						{channelId}
						bind:selectedTagIds
						maxSelectable={5}
					/>
				</div>

				<div class="form-group">
					<label for="post-content">Content</label>
					<textarea
						id="post-content"
						bind:value={content}
						placeholder="Write your post..."
						rows="6"
						maxlength="2000"
						disabled={loading}
					></textarea>
					<span class="char-count">{content.length}/2000</span>
				</div>

				<div class="form-actions">
					<button
						type="button"
						class="btn-secondary"
						on:click={() => dispatch('close')}
						disabled={loading}
					>
						Cancel
					</button>
					<button
						type="submit"
						class="btn-primary"
						disabled={!title.trim() || loading}
					>
						{loading ? 'Creating...' : 'Create Post'}
					</button>
				</div>
			</form>
		</div>
	</div>
{/if}

<style>
	.modal-backdrop {
		position: fixed;
		inset: 0;
		z-index: 1000;
		background: rgba(0, 0, 0, 0.7);
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 20px;
	}

	.modal {
		width: 100%;
		max-width: 560px;
		max-height: 90vh;
		background: var(--bg-secondary, #2b2d31);
		border-radius: var(--radius-lg, 8px);
		box-shadow: 0 16px 48px rgba(0, 0, 0, 0.5);
		display: flex;
		flex-direction: column;
		overflow: hidden;
	}

	.modal-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 16px 20px;
		border-bottom: 1px solid var(--border-color, #4f545c);
	}

	.modal-header h2 {
		margin: 0;
		font-size: 18px;
		font-weight: 600;
		color: var(--text-primary, #f2f3f5);
	}

	.close-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 32px;
		height: 32px;
		padding: 0;
		background: transparent;
		border: none;
		border-radius: var(--radius-sm, 4px);
		color: var(--text-muted, #b5bac1);
		font-size: 24px;
		cursor: pointer;
		transition: background 0.15s, color 0.15s;
	}

	.close-btn:hover {
		background: var(--bg-modifier-accent, #4e5058);
		color: var(--text-primary, #f2f3f5);
	}

	.modal-body {
		padding: 20px;
		overflow-y: auto;
		display: flex;
		flex-direction: column;
		gap: 16px;
	}

	.error-banner {
		padding: 10px 14px;
		background: rgba(237, 66, 69, 0.15);
		border: 1px solid var(--text-danger, #ed4245);
		border-radius: var(--radius-sm, 4px);
		color: var(--text-danger, #ed4245);
		font-size: var(--font-size-sm, 13px);
	}

	.form-group {
		display: flex;
		flex-direction: column;
		gap: 6px;
		position: relative;
	}

	.form-group label {
		font-size: var(--font-size-sm, 13px);
		font-weight: 500;
		color: var(--text-secondary, #b5bac1);
	}

	.required {
		color: var(--text-danger, #ed4245);
	}

	input[type='text'],
	textarea {
		width: 100%;
		padding: 10px 12px;
		background: var(--bg-primary, #36393f);
		border: 1px solid var(--border-color, #4f545c);
		border-radius: var(--radius-sm, 4px);
		color: var(--text-primary, #f2f3f5);
		font-size: var(--font-size-sm, 14px);
		font-family: inherit;
		transition: border-color 0.15s;
		box-sizing: border-box;
	}

	input[type='text']:focus,
	textarea:focus {
		outline: none;
		border-color: var(--brand-primary, #5865f2);
	}

	input[type='text']:disabled,
	textarea:disabled {
		opacity: 0.6;
		cursor: not-allowed;
	}

	textarea {
		resize: vertical;
		min-height: 100px;
	}

	.char-count {
		position: absolute;
		right: 8px;
		bottom: -18px;
		font-size: var(--font-size-xs, 10px);
		color: var(--text-muted, #b5bac1);
	}

	.form-actions {
		display: flex;
		justify-content: flex-end;
		gap: 10px;
		padding-top: 8px;
	}

	.btn-primary,
	.btn-secondary {
		padding: 8px 16px;
		border-radius: var(--radius-sm, 4px);
		font-size: var(--font-size-sm, 14px);
		font-weight: 500;
		cursor: pointer;
		transition: background 0.15s, opacity 0.15s;
	}

	.btn-primary {
		background: var(--brand-primary, #5865f2);
		border: none;
		color: white;
	}

	.btn-primary:hover:not(:disabled) {
		background: var(--brand-hover, #4752c4);
	}

	.btn-primary:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.btn-secondary {
		background: transparent;
		border: 1px solid var(--border-color, #4f545c);
		color: var(--text-primary, #f2f3f5);
	}

	.btn-secondary:hover:not(:disabled) {
		background: var(--bg-modifier-accent, #4e5058);
	}

	.btn-secondary:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}
</style>
