<script lang="ts">
	import { onMount } from 'svelte';
	import {
		loadChannelTags,
		createForumTag,
		updateForumTag,
		deleteForumTag,
		forumTagsByChannel,
		forumTagsLoading,
		type ForumTag
	} from '$lib/stores/forumTags';

	export let channelId: string;
	export let canManage = false;

	let tags: ForumTag[] = [];
	let loading = false;
	let error: string | null = null;

	// New tag form
	let showCreateForm = false;
	let newTag = { name: '', color: '#5865f2', emoji_name: '', moderated: false };
	let creating = false;

	// Edit state
	let editingTag: ForumTag | null = null;
	let editForm = { name: '', color: '', emoji_name: '', moderated: false };

	const TAG_COLORS = [
		{ value: '#5865f2', label: 'Blurple' },
		{ value: '#57f287', label: 'Green' },
		{ value: '#fee75c', label: 'Yellow' },
		{ value: '#ed4245', label: 'Red' },
		{ value: '#eb459e', label: 'Fuchsia' },
		{ value: '#ffffff', label: 'White' },
		{ value: '#9b59b6', label: 'Purple' },
		{ value: '#e67e22', label: 'Orange' },
		{ value: '#3498db', label: 'Blue' },
		{ value: '#1abc9c', label: 'Teal' },
	];

	$: tags = $forumTagsByChannel[channelId] || [];

	onMount(async () => {
		loading = true;
		try {
			await loadChannelTags(channelId);
		} catch (e) {
			error = 'Failed to load tags';
		} finally {
			loading = false;
		}
	});

	async function handleCreate() {
		if (!newTag.name.trim()) return;
		creating = true;
		error = null;
		try {
			await createForumTag(channelId, {
				name: newTag.name.trim(),
				color: newTag.color || undefined,
				emoji_name: newTag.emoji_name || undefined,
				moderated: newTag.moderated
			});
			newTag = { name: '', color: '#5865f2', emoji_name: '', moderated: false };
			showCreateForm = false;
		} catch (e: any) {
			error = e?.data?.error || e?.message || 'Failed to create tag';
		} finally {
			creating = false;
		}
	}

	function startEdit(tag: ForumTag) {
		editingTag = tag;
		editForm = {
			name: tag.name,
			color: tag.color || '#5865f2',
			emoji_name: tag.emoji_name || '',
			moderated: tag.moderated
		};
	}

	async function saveEdit() {
		if (!editingTag || !editForm.name.trim()) return;
		error = null;
		try {
			await updateForumTag(editingTag.id, channelId, {
				name: editForm.name.trim(),
				color: editForm.color || undefined,
				emoji_name: editForm.emoji_name || undefined,
				moderated: editForm.moderated
			});
			editingTag = null;
		} catch (e: any) {
			error = e?.data?.error || e?.message || 'Failed to update tag';
		}
	}

	async function handleDelete(tag: ForumTag) {
		if (!confirm(`Delete tag "${tag.name}"? This will remove it from all posts.`)) return;
		error = null;
		try {
			await deleteForumTag(tag.id, channelId);
		} catch (e: any) {
			error = e?.data?.error || e?.message || 'Failed to delete tag';
		}
	}

	function cancelEdit() {
		editingTag = null;
	}

	function cancelCreate() {
		showCreateForm = false;
		newTag = { name: '', color: '#5865f2', emoji_name: '', moderated: false };
	}
</script>

<div class="forum-tag-editor">
	<div class="tag-header">
		<div>
			<h3>Forum Tags</h3>
			<p class="tag-description">Tags help members filter and find posts. Up to 20 tags per channel.</p>
		</div>
		{#if canManage && !showCreateForm}
			<button
				class="btn btn-primary btn-sm"
				on:click={() => showCreateForm = true}
				disabled={tags.length >= 20}
			>
				Add Tag
			</button>
		{/if}
	</div>

	{#if error}
		<div class="tag-error">{error}</div>
	{/if}

	{#if loading}
		<div class="tag-loading">Loading tags...</div>
	{:else}
		{#if showCreateForm && canManage}
			<div class="tag-form">
				<div class="form-row">
					<div class="form-field flex-1">
						<label for="new-tag-name">Name</label>
						<input
							id="new-tag-name"
							type="text"
							bind:value={newTag.name}
							placeholder="e.g. Bug Report"
							maxlength="100"
						/>
					</div>
					<div class="form-field">
						<label for="new-tag-emoji">Emoji</label>
						<input
							id="new-tag-emoji"
							type="text"
							bind:value={newTag.emoji_name}
							placeholder="optional"
							maxlength="2"
							class="emoji-input"
						/>
					</div>
				</div>

				<div class="form-field">
					<label>Color</label>
					<div class="color-grid">
						{#each TAG_COLORS as c}
							<button
								class="color-swatch"
								class:selected={newTag.color === c.value}
								style="background-color: {c.value}"
								title={c.label}
								on:click={() => newTag.color = c.value}
							/>
						{/each}
					</div>
				</div>

				<label class="checkbox-field">
					<input type="checkbox" bind:checked={newTag.moderated} />
					<span>Moderated (only moderators can apply)</span>
				</label>

				<div class="form-actions">
					<button class="btn btn-secondary btn-sm" on:click={cancelCreate}>Cancel</button>
					<button
						class="btn btn-primary btn-sm"
						on:click={handleCreate}
						disabled={creating || !newTag.name.trim()}
					>
						{creating ? 'Creating...' : 'Create Tag'}
					</button>
				</div>
			</div>
		{/if}

		{#if tags.length === 0 && !showCreateForm}
			<div class="tag-empty">No tags yet. {canManage ? 'Add a tag to get started.' : ''}</div>
		{:else}
			<div class="tag-list">
				{#each tags as tag (tag.id)}
					{#if editingTag?.id === tag.id}
						<div class="tag-form tag-edit-form">
							<div class="form-row">
								<div class="form-field flex-1">
									<label for="edit-tag-name">Name</label>
									<input
										id="edit-tag-name"
										type="text"
										bind:value={editForm.name}
										maxlength="100"
									/>
								</div>
								<div class="form-field">
									<label for="edit-tag-emoji">Emoji</label>
									<input
										id="edit-tag-emoji"
										type="text"
										bind:value={editForm.emoji_name}
										placeholder="optional"
										maxlength="2"
										class="emoji-input"
									/>
								</div>
							</div>

							<div class="form-field">
								<label>Color</label>
								<div class="color-grid">
									{#each TAG_COLORS as c}
										<button
											class="color-swatch"
											class:selected={editForm.color === c.value}
											style="background-color: {c.value}"
											title={c.label}
											on:click={() => editForm.color = c.value}
										/>
									{/each}
								</div>
							</div>

							<label class="checkbox-field">
								<input type="checkbox" bind:checked={editForm.moderated} />
								<span>Moderated</span>
							</label>

							<div class="form-actions">
								<button class="btn btn-secondary btn-sm" on:click={cancelEdit}>Cancel</button>
								<button
									class="btn btn-primary btn-sm"
									on:click={saveEdit}
									disabled={!editForm.name.trim()}
								>
									Save
								</button>
							</div>
						</div>
					{:else}
						<div class="tag-item">
							<div class="tag-preview" style="background-color: {tag.color || '#5865f2'}">
								{#if tag.emoji_name}
									<span class="tag-emoji">{tag.emoji_name}</span>
								{/if}
								<span class="tag-name">{tag.name}</span>
							</div>
							{#if tag.moderated}
								<span class="tag-badge">Mod only</span>
							{/if}
							{#if canManage}
								<div class="tag-actions">
									<button class="btn-icon" on:click={() => startEdit(tag)} title="Edit tag">
										&#9998;
									</button>
									<button class="btn-icon danger" on:click={() => handleDelete(tag)} title="Delete tag">
										&#128465;
									</button>
								</div>
							{/if}
						</div>
					{/if}
				{/each}
			</div>
		{/if}

		{#if tags.length > 0}
			<div class="tag-count">{tags.length}/20 tags used</div>
		{/if}
	{/if}
</div>

<style>
	.forum-tag-editor {
		display: flex;
		flex-direction: column;
		gap: 12px;
	}

	.tag-header {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		gap: 16px;
	}

	.tag-header h3 {
		margin: 0 0 4px;
		font-size: 16px;
		font-weight: 600;
		color: var(--text-primary);
	}

	.tag-description {
		margin: 0;
		font-size: 13px;
		color: var(--text-muted);
	}

	.tag-error {
		padding: 8px 12px;
		background: rgba(237, 66, 69, 0.1);
		border: 1px solid rgba(237, 66, 69, 0.3);
		border-radius: 4px;
		color: #ed4245;
		font-size: 13px;
	}

	.tag-loading, .tag-empty {
		padding: 24px;
		text-align: center;
		color: var(--text-muted);
		font-size: 14px;
	}

	.tag-form {
		padding: 16px;
		background: var(--bg-secondary);
		border-radius: 8px;
		display: flex;
		flex-direction: column;
		gap: 12px;
	}

	.tag-edit-form {
		margin-bottom: 4px;
	}

	.form-row {
		display: flex;
		gap: 12px;
	}

	.form-field {
		display: flex;
		flex-direction: column;
		gap: 4px;
	}

	.form-field.flex-1 {
		flex: 1;
	}

	.form-field label {
		font-size: 12px;
		font-weight: 600;
		text-transform: uppercase;
		color: var(--text-muted);
	}

	.form-field input[type="text"] {
		padding: 8px 12px;
		background: var(--bg-tertiary);
		border: 1px solid transparent;
		border-radius: 4px;
		color: var(--text-primary);
		font-size: 14px;
		outline: none;
	}

	.form-field input[type="text"]:focus {
		border-color: var(--brand-primary);
	}

	.emoji-input {
		width: 60px;
		text-align: center;
		font-size: 18px !important;
	}

	.color-grid {
		display: flex;
		gap: 6px;
		flex-wrap: wrap;
	}

	.color-swatch {
		width: 28px;
		height: 28px;
		border-radius: 50%;
		border: 3px solid transparent;
		cursor: pointer;
		transition: border-color 0.15s, transform 0.15s;
	}

	.color-swatch:hover {
		transform: scale(1.1);
	}

	.color-swatch.selected {
		border-color: var(--text-primary);
	}

	.checkbox-field {
		display: flex;
		align-items: center;
		gap: 8px;
		font-size: 14px;
		color: var(--text-secondary);
		cursor: pointer;
	}

	.checkbox-field input[type="checkbox"] {
		width: 18px;
		height: 18px;
		accent-color: var(--brand-primary);
	}

	.form-actions {
		display: flex;
		justify-content: flex-end;
		gap: 8px;
	}

	.tag-list {
		display: flex;
		flex-direction: column;
		gap: 4px;
	}

	.tag-item {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 8px 12px;
		background: var(--bg-secondary);
		border-radius: 6px;
		transition: background 0.15s;
	}

	.tag-item:hover {
		background: var(--bg-tertiary);
	}

	.tag-preview {
		display: inline-flex;
		align-items: center;
		gap: 4px;
		padding: 2px 10px;
		border-radius: 99px;
		font-size: 13px;
		font-weight: 500;
		color: #fff;
	}

	.tag-emoji {
		font-size: 14px;
	}

	.tag-name {
		line-height: 22px;
	}

	.tag-badge {
		font-size: 11px;
		padding: 1px 6px;
		border-radius: 4px;
		background: var(--bg-tertiary);
		color: var(--text-muted);
	}

	.tag-actions {
		display: flex;
		gap: 4px;
		margin-left: auto;
		opacity: 0;
		transition: opacity 0.15s;
	}

	.tag-item:hover .tag-actions {
		opacity: 1;
	}

	.btn-icon {
		width: 28px;
		height: 28px;
		display: flex;
		align-items: center;
		justify-content: center;
		background: none;
		border: none;
		border-radius: 4px;
		color: var(--text-muted);
		cursor: pointer;
		font-size: 14px;
	}

	.btn-icon:hover {
		background: var(--bg-secondary);
		color: var(--text-primary);
	}

	.btn-icon.danger:hover {
		color: #ed4245;
	}

	.tag-count {
		font-size: 12px;
		color: var(--text-muted);
		text-align: right;
	}

	.btn {
		padding: 8px 16px;
		border: none;
		border-radius: 4px;
		font-size: 14px;
		font-weight: 500;
		cursor: pointer;
		transition: background 0.15s, opacity 0.15s;
	}

	.btn:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.btn-primary {
		background: var(--brand-primary, #5865f2);
		color: white;
	}

	.btn-primary:hover:not(:disabled) {
		background: var(--brand-primary-hover, #4752c4);
	}

	.btn-secondary {
		background: var(--bg-tertiary);
		color: var(--text-primary);
	}

	.btn-secondary:hover:not(:disabled) {
		background: var(--bg-modifier-hover, #4e505899);
	}

	.btn-sm {
		padding: 6px 12px;
		font-size: 13px;
	}
</style>
