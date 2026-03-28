<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import { 
		loadApplicationCommands, 
		registerCommand, 
		updateCommand, 
		deleteCommand,
		type SlashCommand,
		type RegisterCommandRequest,
		type CommandOption,
		getOptionTypeLabel
	} from '$lib/services/slashCommands';
	import CommandOptionEditor from './CommandOptionEditor.svelte';
	
	export let appId: string;
	export let appName: string = 'Application';
	export let visible: boolean = false;
	
	const dispatch = createEventDispatcher<{
		close: void;
		commandCreated: { command: SlashCommand };
		commandUpdated: { command: SlashCommand };
		commandDeleted: { commandId: string };
	}>();
	
	let commands: SlashCommand[] = [];
	let loading = false;
	let error: string | null = null;
	
	// Editor state
	let showEditor = false;
	let editingCommand: SlashCommand | null = null;
	let editorForm: RegisterCommandRequest = {
		name: '',
		description: '',
		type: 1,
		options: [],
		default_permission: true
	};
	let editorErrors: Record<string, string> = {};
	
	// Confirmation dialog
	let confirmDelete: SlashCommand | null = null;
	
	$: if (visible && appId) {
		loadCommands();
	}
	
	async function loadCommands() {
		loading = true;
		error = null;
		try {
			commands = await loadApplicationCommands(appId);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load commands';
		} finally {
			loading = false;
		}
	}
	
	function openCreateEditor() {
		editingCommand = null;
		editorForm = {
			name: '',
			description: '',
			type: 1,
			options: [],
			default_permission: true
		};
		editorErrors = {};
		showEditor = true;
	}
	
	function openEditEditor(cmd: SlashCommand) {
		editingCommand = cmd;
		editorForm = {
			name: cmd.name,
			description: cmd.description,
			type: cmd.type as 1 | 2 | 3,
			options: cmd.options ? [...cmd.options] : [],
			default_permission: cmd.default_permission,
			guild_id: cmd.guild_id
		};
		editorErrors = {};
		showEditor = true;
	}
	
	function closeEditor() {
		showEditor = false;
		editingCommand = null;
	}
	
	function validateEditor(): boolean {
		editorErrors = {};
		
		if (!editorForm.name || editorForm.name.trim() === '') {
			editorErrors.name = 'Command name is required';
		} else if (editorForm.name.length > 32) {
			editorErrors.name = 'Command name must be 32 characters or less';
		} else if (!/^[\w-]{1,32}$/.test(editorForm.name)) {
			editorErrors.name = 'Command name can only contain letters, numbers, hyphens, and underscores';
		}
		
		if (!editorForm.description || editorForm.description.trim() === '') {
			editorErrors.description = 'Description is required';
		} else if (editorForm.description.length > 100) {
			editorErrors.description = 'Description must be 100 characters or less';
		}
		
		// Validate options
		if (editorForm.options) {
			for (const opt of editorForm.options) {
				if (!opt.name || opt.name.trim() === '') {
					editorErrors[`option_${opt.name}`] = 'Option name is required';
				}
				if (!opt.description || opt.description.trim() === '') {
					editorErrors[`option_desc_${opt.name}`] = 'Option description is required';
				}
			}
		}
		
		return Object.keys(editorErrors).length === 0;
	}
	
	async function handleSave() {
		if (!validateEditor()) return;
		
		loading = true;
		error = null;
		
		try {
			if (editingCommand) {
				const updated = await updateCommand(appId, editingCommand.id, editorForm);
				commands = commands.map(c => c.id === editingCommand!.id ? updated : c);
				dispatch('commandUpdated', { command: updated });
			} else {
				const created = await registerCommand(appId, editorForm);
				commands = [...commands, created];
				dispatch('commandCreated', { command: created });
			}
			closeEditor();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to save command';
		} finally {
			loading = false;
		}
	}
	
	async function handleDelete(cmd: SlashCommand) {
		loading = true;
		error = null;
		
		try {
			await deleteCommand(appId, cmd.id);
			commands = commands.filter(c => c.id !== cmd.id);
			dispatch('commandDeleted', { commandId: cmd.id });
			confirmDelete = null;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to delete command';
		} finally {
			loading = false;
		}
	}
	
	function handleAddOption() {
		editorForm.options = [
			...(editorForm.options || []),
			{
				type: 3, // String
				name: '',
				description: '',
				required: false
			}
		];
	}
	
	function handleRemoveOption(index: number) {
		editorForm.options = (editorForm.options || []).filter((_, i) => i !== index);
	}
	
	function handleUpdateOption(index: number, option: CommandOption) {
		editorForm.options = (editorForm.options || []).map((o, i) => i === index ? option : o);
	}
</script>

{#if visible}
	<div class="modal-backdrop" on:click={() => dispatch('close')} on:keydown={(e) => e.key === 'Escape' && dispatch('close')} role="button" tabindex="0">
		<div class="modal-content" on:click|stopPropagation role="dialog" aria-modal="true" aria-labelledby="dashboard-title">
			<div class="modal-header">
				<h2 id="dashboard-title">Slash Commands — {appName}</h2>
				<button class="close-btn" on:click={() => dispatch('close')} aria-label="Close">
					<svg width="20" height="20" viewBox="0 0 20 20" fill="currentColor">
						<path d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"/>
					</svg>
				</button>
			</div>
			
			{#if error}
				<div class="error-banner">{error}</div>
			{/if}
			
			{#if !showEditor}
				<div class="commands-list">
					{#if loading && commands.length === 0}
						<div class="loading-state">Loading commands...</div>
					{:else if commands.length === 0}
						<div class="empty-state">
							<div class="empty-icon">/</div>
							<h3>No commands yet</h3>
							<p>Create your first slash command to get started</p>
						</div>
					{:else}
						<div class="command-header">
							<span class="header-count">{commands.length} command{commands.length !== 1 ? 's' : ''}</span>
							<button class="create-btn" on:click={openCreateEditor}>
								<svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
									<path d="M8 2a.75.75 0 01.75.75v4.5h4.5a.75.75 0 010 1.5h-4.5v4.5a.75.75 0 01-1.5 0v-4.5h-4.5a.75.75 0 010-1.5h4.5v-4.5A.75.75 0 018 2z"/>
								</svg>
								New Command
							</button>
						</div>
						
						{#each commands as cmd (cmd.id)}
							<div class="command-card">
								<div class="command-info">
									<div class="command-name">
										<span class="command-icon">/</span>
										{cmd.name}
									</div>
									<div class="command-desc">{cmd.description}</div>
									<div class="command-meta">
										<span class="cmd-type">
											{cmd.type === 1 ? 'Slash' : cmd.type === 2 ? 'User' : 'Message'}
										</span>
										{#if cmd.guild_id}
											<span class="cmd-scope">Server-specific</span>
										{:else}
											<span class="cmd-scope">Global</span>
										{/if}
										{#if cmd.options && cmd.options.length > 0}
											<span class="cmd-options">{cmd.options.length} option{cmd.options.length !== 1 ? 's' : ''}</span>
										{/if}
									</div>
								</div>
								<div class="command-actions">
									<button class="action-btn edit" on:click={() => openEditEditor(cmd)} title="Edit">
										<svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
											<path d="M11.013 1.427a1.75 1.75 0 012.474 0l1.086 1.086a1.75 1.75 0 010 2.474l-8.61 8.61c-.21.21-.47.364-.756.445l-3.251.93a.75.75 0 01-.927-.928l.929-3.25c.081-.286.235-.547.445-.758l8.61-8.61zm1.414 1.06a.25.25 0 00-.354 0L10.811 3.75l1.439 1.44 1.263-1.263a.25.25 0 000-.354l-1.086-1.086zM11.189 6.25L9.75 4.81l-6.286 6.287a.25.25 0 00-.064.108l-.558 1.953 1.953-.558a.25.25 0 00.108-.064l6.286-6.286z"/>
										</svg>
									</button>
									<button class="action-btn delete" on:click={() => confirmDelete = cmd} title="Delete">
										<svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
											<path d="M6.5 1.75a.25.25 0 01.25-.25h2.5a.25.25 0 01.25.25V3h-3V1.75zm4.5 0V3h2.25a.75.75 0 010 1.5H2.75a.75.75 0 010-1.5H5V1.75C5 .784 5.784 0 6.75 0h2.5C10.216 0 11 .784 11 1.75zM4.496 6.675a.75.75 0 10-1.492.15l.66 6.6A1.75 1.75 0 005.405 15h5.19a1.75 1.75 0 001.741-1.575l.66-6.6a.75.75 0 00-1.492-.15l-.66 6.6a.25.25 0 01-.249.225h-5.19a.25.25 0 01-.249-.225l-.66-6.6z"/>
										</svg>
									</button>
								</div>
							</div>
						{/each}
					{/if}
				</div>
			{:else}
				<!-- Command Editor -->
				<div class="editor">
					<div class="editor-header">
						<h3>{editingCommand ? 'Edit Command' : 'Create Command'}</h3>
						<button class="cancel-btn" on:click={closeEditor}>Cancel</button>
					</div>
					
					<div class="editor-form">
						<div class="form-group">
							<label for="cmd-name">Command Name *</label>
							<input
								id="cmd-name"
								type="text"
								bind:value={editorForm.name}
								placeholder="command-name"
								maxlength="32"
								class:error={editorErrors.name}
							/>
							{#if editorErrors.name}
								<span class="field-error">{editorErrors.name}</span>
							{/if}
							<span class="field-hint">1-32 characters, letters, numbers, hyphens, underscores</span>
						</div>
						
						<div class="form-group">
							<label for="cmd-desc">Description *</label>
							<input
								id="cmd-desc"
								type="text"
								bind:value={editorForm.description}
								placeholder="What does this command do?"
								maxlength="100"
								class:error={editorErrors.description}
							/>
							{#if editorErrors.description}
								<span class="field-error">{editorErrors.description}</span>
							{/if}
							<span class="field-hint">1-100 characters</span>
						</div>
						
						<div class="form-group">
							<label for="cmd-type">Command Type</label>
							<select id="cmd-type" bind:value={editorForm.type}>
								<option value={1}>Slash Command</option>
								<option value={2}>User Context Menu</option>
								<option value={3}>Message Context Menu</option>
							</select>
						</div>
						
						<div class="form-group">
							<label>
								<input type="checkbox" bind:checked={editorForm.default_permission} />
								Allow everyone to use this command
							</label>
						</div>
						
						<div class="options-section">
							<div class="options-header">
								<label>Options</label>
								<button class="add-option-btn" on:click={handleAddOption}>
									+ Add Option
								</button>
							</div>
							
							{#if editorForm.options && editorForm.options.length > 0}
								<div class="options-list">
									{#each editorForm.options as option, index}
										<CommandOptionEditor
											{option}
											on:update={(e) => handleUpdateOption(index, e.detail)}
											on:remove={() => handleRemoveOption(index)}
										/>
									{/each}
								</div>
							{:else}
								<p class="no-options">No options yet. Add options to accept user input.</p>
							{/if}
						</div>
					</div>
					
					<div class="editor-footer">
						<button class="save-btn primary" on:click={handleSave} disabled={loading}>
							{loading ? 'Saving...' : (editingCommand ? 'Save Changes' : 'Create Command')}
						</button>
					</div>
				</div>
			{/if}
		</div>
	</div>
{/if}

<!-- Delete Confirmation Modal -->
{#if confirmDelete}
	<div class="confirm-backdrop" on:click={() => confirmDelete = null} on:keydown={(e) => e.key === 'Escape' && (confirmDelete = null)} role="button" tabindex="0">
		<div class="confirm-dialog" on:click|stopPropagation role="alertdialog" aria-modal="true">
			<h3>Delete Command</h3>
			<p>Are you sure you want to delete <strong>/{confirmDelete.name}</strong>? This action cannot be undone.</p>
			<div class="confirm-actions">
				<button class="cancel-btn" on:click={() => confirmDelete = null}>Cancel</button>
				<button class="confirm-delete-btn" on:click={() => handleDelete(confirmDelete!)}>Delete</button>
			</div>
		</div>
	</div>
{/if}

<style>
	.modal-backdrop {
		position: fixed;
		inset: 0;
		background: rgba(0, 0, 0, 0.7);
		display: flex;
		align-items: center;
		justify-content: center;
		z-index: 1000;
		padding: 20px;
	}
	
	.modal-content {
		background: var(--bg-secondary, #2b2d31);
		border-radius: 8px;
		width: 100%;
		max-width: 720px;
		max-height: 80vh;
		overflow: hidden;
		display: flex;
		flex-direction: column;
	}
	
	.modal-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 16px 20px;
		border-bottom: 1px solid var(--bg-tertiary, #1e1f22);
	}
	
	.modal-header h2 {
		font-size: 16px;
		font-weight: 600;
		color: var(--text-normal, #dbdee1);
		margin: 0;
	}
	
	.close-btn {
		background: transparent;
		border: none;
		color: var(--text-muted, #949ba4);
		cursor: pointer;
		padding: 4px;
		border-radius: 4px;
		transition: all 0.15s;
	}
	
	.close-btn:hover {
		background: var(--bg-modifier-hover, #3f434a);
		color: var(--text-normal, #dbdee1);
	}
	
	.error-banner {
		background: var(--status-danger, #ed4245);
		color: white;
		padding: 10px 20px;
		font-size: 13px;
	}
	
	.commands-list {
		flex: 1;
		overflow-y: auto;
		padding: 12px;
	}
	
	.loading-state,
	.empty-state {
		text-align: center;
		padding: 40px 20px;
		color: var(--text-muted, #949ba4);
	}
	
	.empty-icon {
		font-size: 48px;
		font-weight: 700;
		color: var(--text-muted, #949ba4);
		margin-bottom: 12px;
	}
	
	.empty-state h3 {
		margin: 0 0 8px;
		color: var(--text-normal, #dbdee1);
		font-size: 16px;
	}
	
	.empty-state p {
		margin: 0;
		font-size: 13px;
	}
	
	.command-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-bottom: 12px;
	}
	
	.header-count {
		font-size: 12px;
		color: var(--text-muted, #949ba4);
	}
	
	.create-btn {
		display: flex;
		align-items: center;
		gap: 6px;
		padding: 8px 16px;
		background: var(--text-link, #5865f2);
		border: none;
		border-radius: 4px;
		color: white;
		font-size: 13px;
		font-weight: 500;
		cursor: pointer;
		transition: background-color 0.15s;
	}
	
	.create-btn:hover {
		background: #4752c4;
	}
	
	.command-card {
		display: flex;
		justify-content: space-between;
		align-items: flex-start;
		padding: 12px;
		background: var(--bg-primary, #313338);
		border-radius: 6px;
		margin-bottom: 8px;
	}
	
	.command-card:hover {
		background: var(--bg-modifier-hover, #3f434a);
	}
	
	.command-name {
		display: flex;
		align-items: center;
		gap: 4px;
		font-weight: 600;
		font-size: 14px;
		color: var(--text-normal, #dbdee1);
	}
	
	.command-icon {
		color: var(--text-muted, #949ba4);
	}
	
	.command-desc {
		font-size: 12px;
		color: var(--text-muted, #949ba4);
		margin-top: 4px;
	}
	
	.command-meta {
		display: flex;
		gap: 8px;
		margin-top: 6px;
		font-size: 10px;
	}
	
	.cmd-type,
	.cmd-scope,
	.cmd-options {
		background: var(--bg-tertiary, #1e1f22);
		padding: 2px 6px;
		border-radius: 4px;
		color: var(--text-muted, #949ba4);
	}
	
	.command-actions {
		display: flex;
		gap: 4px;
	}
	
	.action-btn {
		padding: 6px;
		background: transparent;
		border: none;
		border-radius: 4px;
		color: var(--text-muted, #949ba4);
		cursor: pointer;
		transition: all 0.15s;
	}
	
	.action-btn:hover {
		background: var(--bg-modifier-hover, #3f434a);
	}
	
	.action-btn.delete:hover {
		background: var(--status-danger, #ed4245);
		color: white;
	}
	
	/* Editor */
	.editor {
		display: flex;
		flex-direction: column;
		height: 100%;
	}
	
	.editor-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 16px 20px;
		border-bottom: 1px solid var(--bg-tertiary, #1e1f22);
	}
	
	.editor-header h3 {
		margin: 0;
		font-size: 14px;
		font-weight: 600;
		color: var(--text-normal, #dbdee1);
	}
	
	.editor-form {
		flex: 1;
		overflow-y: auto;
		padding: 20px;
	}
	
	.form-group {
		margin-bottom: 16px;
	}
	
	.form-group label {
		display: block;
		font-size: 12px;
		font-weight: 500;
		color: var(--text-normal, #dbdee1);
		margin-bottom: 6px;
	}
	
	.form-group input[type="text"],
	.form-group select {
		width: 100%;
		padding: 10px 12px;
		background: var(--bg-primary, #313338);
		border: 1px solid var(--bg-tertiary, #1e1f22);
		border-radius: 4px;
		color: var(--text-normal, #dbdee1);
		font-size: 14px;
	}
	
	.form-group input:focus,
	.form-group select:focus {
		outline: none;
		border-color: var(--text-link, #5865f2);
	}
	
	.form-group input.error {
		border-color: var(--status-danger, #ed4245);
	}
	
	.field-error {
		display: block;
		font-size: 11px;
		color: var(--status-danger, #ed4245);
		margin-top: 4px;
	}
	
	.field-hint {
		display: block;
		font-size: 11px;
		color: var(--text-muted, #949ba4);
		margin-top: 4px;
	}
	
	.options-section {
		margin-top: 20px;
		padding-top: 20px;
		border-top: 1px solid var(--bg-tertiary, #1e1f22);
	}
	
	.options-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-bottom: 12px;
	}
	
	.options-header label {
		font-size: 12px;
		font-weight: 500;
		color: var(--text-normal, #dbdee1);
	}
	
	.add-option-btn {
		padding: 4px 10px;
		background: transparent;
		border: 1px solid var(--text-link, #5865f2);
		border-radius: 4px;
		color: var(--text-link, #5865f2);
		font-size: 12px;
		cursor: pointer;
		transition: all 0.15s;
	}
	
	.add-option-btn:hover {
		background: var(--text-link, #5865f2);
		color: white;
	}
	
	.no-options {
		font-size: 13px;
		color: var(--text-muted, #949ba4);
		text-align: center;
		padding: 20px;
	}
	
	.editor-footer {
		padding: 16px 20px;
		border-top: 1px solid var(--bg-tertiary, #1e1f22);
		display: flex;
		justify-content: flex-end;
	}
	
	.save-btn {
		padding: 10px 20px;
		border: none;
		border-radius: 4px;
		font-size: 14px;
		font-weight: 500;
		cursor: pointer;
		transition: all 0.15s;
	}
	
	.save-btn.primary {
		background: var(--text-link, #5865f2);
		color: white;
	}
	
	.save-btn.primary:hover {
		background: #4752c4;
	}
	
	.save-btn:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}
	
	.cancel-btn {
		padding: 10px 16px;
		background: transparent;
		border: none;
		color: var(--text-muted, #949ba4);
		font-size: 14px;
		cursor: pointer;
		transition: color 0.15s;
	}
	
	.cancel-btn:hover {
		color: var(--text-normal, #dbdee1);
	}
	
	/* Confirm dialog */
	.confirm-backdrop {
		position: fixed;
		inset: 0;
		background: rgba(0, 0, 0, 0.8);
		display: flex;
		align-items: center;
		justify-content: center;
		z-index: 1100;
	}
	
	.confirm-dialog {
		background: var(--bg-secondary, #2b2d31);
		border-radius: 8px;
		padding: 24px;
		max-width: 400px;
		width: 90%;
	}
	
	.confirm-dialog h3 {
		margin: 0 0 12px;
		font-size: 16px;
		font-weight: 600;
		color: var(--text-normal, #dbdee1);
	}
	
	.confirm-dialog p {
		margin: 0 0 20px;
		font-size: 14px;
		color: var(--text-muted, #949ba4);
	}
	
	.confirm-dialog strong {
		color: var(--text-normal, #dbdee1);
	}
	
	.confirm-actions {
		display: flex;
		justify-content: flex-end;
		gap: 8px;
	}
	
	.confirm-delete-btn {
		padding: 8px 16px;
		background: var(--status-danger, #ed4245);
		border: none;
		border-radius: 4px;
		color: white;
		font-size: 14px;
		font-weight: 500;
		cursor: pointer;
	}
	
	.confirm-delete-btn:hover {
		background: #c93b3e;
	}
</style>
