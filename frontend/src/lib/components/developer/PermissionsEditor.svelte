<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import type { SlashCommand, CommandPermissionOverride } from '$lib/services/slashCommands';

	export let command: SlashCommand;
	export let permissions: CommandPermissionOverride[] = [];
	export let guildId: string = '';
	export let loading = false;

	const dispatch = createEventDispatcher<{
		save: { permissions: CommandPermissionOverride[] };
		cancel: void;
	}>();

	// Local state
	let overrides: CommandPermissionOverride[] = permissions.length > 0
		? [...permissions]
		: [];
	let selectedType: 1 | 2 = 1; // 1 = role, 2 = user
	let selectedId = '';
	let selectedAllow = false;
	let selectedDeny = false;

	function addOverride() {
		if (!selectedId.trim()) return;

		const override: CommandPermissionOverride = {
			id: selectedId.trim() as any,
			type: selectedType,
			allow: selectedAllow,
			deny: selectedDeny
		};

		// Check for duplicates
		const existingIndex = overrides.findIndex(
			o => o.id === override.id && o.type === override.type
		);

		if (existingIndex >= 0) {
			overrides[existingIndex] = override;
		} else {
			overrides = [...overrides, override];
		}

		// Reset form
		selectedId = '';
		selectedAllow = false;
		selectedDeny = false;
	}

	function removeOverride(index: number) {
		overrides = overrides.filter((_, i) => i !== index);
	}

	function updateOverride(index: number, field: 'allow' | 'deny') {
		overrides = overrides.map((o, i) => {
			if (i !== index) return o;
			return {
				...o,
				allow: field === 'allow' ? !o.allow : o.allow,
				deny: field === 'deny' ? !o.deny : o.deny
			};
		});
	}

	function handleSubmit() {
		dispatch('save', { permissions: overrides });
	}

	function getPermissionTypeLabel(typeValue: number): string {
		return typeValue === 1 ? 'Role' : 'User';
	}

	function formatId(id: string): string {
		// Truncate UUID for display
		if (id.length > 36) {
			return id.substring(0, 8) + '...' + id.substring(id.length - 8);
		}
		return id;
	}
</script>

<div class="permissions-editor">
	<form on:submit|preventDefault={handleSubmit}>
		<div class="form-section">
			<h3>Command Permissions</h3>
			<p class="description">
				Configure who can use <strong>/{command?.name || 'command'}</strong> in this server.
			</p>

			{#if !command?.default_permission}
				<div class="warning-banner">
					⚠️ This command is disabled by default. Add permissions below to enable it for specific roles or users.
				</div>
			{:else}
				<div class="info-banner">
					ℹ️ This command is enabled for everyone by default. Add denies below to restrict access.
				</div>
			{/if}
		</div>

		<div class="form-section">
			<h4>Current Permissions</h4>

			{#if overrides.length === 0}
				<div class="empty-permissions">
					<p>No specific permissions configured.</p>
					<p class="hint">Add role or user permissions below.</p>
				</div>
			{:else}
				<div class="permissions-list">
					<div class="permissions-header">
						<span class="col-type">Type</span>
						<span class="col-id">ID</span>
						<span class="col-allow">Allow</span>
						<span class="col-deny">Deny</span>
						<span class="col-actions">Actions</span>
					</div>
					{#each overrides as override, index (index)}
						<div class="permission-row">
							<span class="col-type">
								<span class="type-badge" class:role={override.type === 1} class:user={override.type === 2}>
									{getPermissionTypeLabel(override.type)}
								</span>
							</span>
							<span class="col-id">
								<code>{formatId(override.id)}</code>
							</span>
							<span class="col-allow">
								<button
									type="button"
									class="toggle-btn"
									class:active={override.allow}
									on:click={() => updateOverride(index, 'allow')}
								>
									{override.allow ? '✓' : '—'}
								</button>
							</span>
							<span class="col-deny">
								<button
									type="button"
									class="toggle-btn"
									class:active={override.deny}
									class:deny={override.deny}
									on:click={() => updateOverride(index, 'deny')}
								>
									{override.deny ? '✗' : '—'}
								</button>
							</span>
							<span class="col-actions">
								<button
									type="button"
									class="btn-icon danger"
									on:click={() => removeOverride(index)}
								>
									×
								</button>
							</span>
						</div>
					{/each}
				</div>
			{/if}
		</div>

		<div class="form-section">
			<h4>Add Permission</h4>

			<div class="add-form">
				<div class="form-row">
					<div class="form-group">
						<label for="permType">Type</label>
						<select id="permType" bind:value={selectedType}>
							<option value={1}>Role</option>
							<option value={2}>User</option>
						</select>
					</div>

					<div class="form-group flex-1">
						<label for="permId">{selectedType === 1 ? 'Role' : 'User'} ID</label>
						<input
							id="permId"
							type="text"
							bind:value={selectedId}
							placeholder="Enter {selectedType === 1 ? 'role' : 'user'} ID"
						/>
					</div>
				</div>

				<div class="form-row checkboxes">
					<label class="checkbox-label">
						<input type="checkbox" bind:checked={selectedAllow} />
						<span>Allow</span>
					</label>
					<label class="checkbox-label">
						<input type="checkbox" bind:checked={selectedDeny} />
						<span>Deny</span>
					</label>
				</div>

				<button type="button" class="btn-secondary" on:click={addOverride}>
					Add Permission
				</button>
			</div>
		</div>

		{#if guildId}
			<div class="form-section">
				<div class="info-banner">
					These permissions apply to server: <code>{guildId}</code>
				</div>
			</div>
		{/if}

		<div class="form-actions">
			<button type="button" class="btn-secondary" on:click={() => dispatch('cancel')}>
				Cancel
			</button>
			<button type="submit" class="btn-primary" disabled={loading}>
				{#if loading}
					Saving...
				{:else}
					Save Permissions
				{/if}
			</button>
		</div>
	</form>
</div>

<style>
	.permissions-editor {
		padding: 20px;
		max-width: 700px;
		margin: 0 auto;
	}

	.form-section {
		margin-bottom: 24px;
		padding-bottom: 20px;
		border-bottom: 1px solid var(--border-color, #3a3a3a);
	}

	.form-section:last-of-type {
		border-bottom: none;
	}

	h3 {
		font-size: 16px;
		font-weight: 600;
		margin-bottom: 8px;
		color: var(--text-primary, #ffffff);
	}

	h4 {
		font-size: 14px;
		font-weight: 600;
		margin-bottom: 12px;
		color: var(--text-secondary, #b9b9b9);
	}

	.description {
		font-size: 13px;
		color: var(--text-secondary, #b9b9b9);
		margin-bottom: 12px;
	}

	.warning-banner {
		padding: 12px;
		background: rgba(218, 55, 60, 0.1);
		border: 1px solid var(--danger-color, #da373c);
		border-radius: 6px;
		font-size: 13px;
		color: var(--danger-color, #da373c);
	}

	.info-banner {
		padding: 12px;
		background: rgba(88, 101, 242, 0.1);
		border: 1px solid var(--accent-color, #5865f2);
		border-radius: 6px;
		font-size: 13px;
		color: var(--accent-color, #5865f2);
	}

	.empty-permissions {
		padding: 24px;
		text-align: center;
		background: var(--background-secondary, #2a2a2a);
		border-radius: 6px;
		color: var(--text-secondary, #b9b9b9);
	}

	.empty-permissions .hint {
		font-size: 12px;
		color: var(--text-muted, #727067);
		margin-top: 4px;
	}

	.permissions-list {
		background: var(--background-secondary, #2a2a2a);
		border-radius: 6px;
		overflow: hidden;
	}

	.permissions-header {
		display: flex;
		padding: 10px 12px;
		background: var(--background-tertiary, #3a3a3a);
		font-size: 11px;
		font-weight: 600;
		text-transform: uppercase;
		color: var(--text-muted, #727067);
	}

	.permission-row {
		display: flex;
		align-items: center;
		padding: 10px 12px;
		border-top: 1px solid var(--border-color, #3a3a3a);
	}

	.col-type { width: 70px; }
	.col-id { flex: 1; }
	.col-allow { width: 60px; text-align: center; }
	.col-deny { width: 60px; text-align: center; }
	.col-actions { width: 40px; }

	.type-badge {
		display: inline-block;
		padding: 2px 8px;
		border-radius: 4px;
		font-size: 11px;
		font-weight: 500;
	}

	.type-badge.role {
		background: rgba(88, 101, 242, 0.2);
		color: var(--accent-color, #5865f2);
	}

	.type-badge.user {
		background: rgba(35, 165, 89, 0.2);
		color: var(--success-color, #23a559);
	}

	code {
		font-family: monospace;
		font-size: 12px;
		color: var(--text-secondary, #b9b9b9);
	}

	.toggle-btn {
		padding: 4px 12px;
		background: var(--background-tertiary, #3a3a3a);
		border: none;
		border-radius: 4px;
		color: var(--text-muted, #727067);
		cursor: pointer;
		font-size: 14px;
		transition: all 0.15s;
	}

	.toggle-btn.active {
		background: var(--success-color, #23a559);
		color: white;
	}

	.toggle-btn.deny {
		background: var(--danger-color, #da373c);
		color: white;
	}

	.btn-icon {
		padding: 4px 8px;
		background: transparent;
		border: none;
		border-radius: 4px;
		color: var(--text-secondary, #b9b9b9);
		cursor: pointer;
		font-size: 16px;
	}

	.btn-icon.danger:hover {
		background: var(--danger-color, #da373c);
		color: white;
	}

	.add-form {
		background: var(--background-secondary, #2a2a2a);
		padding: 16px;
		border-radius: 6px;
	}

	.form-row {
		display: flex;
		gap: 12px;
		margin-bottom: 12px;
	}

	.form-row.checkboxes {
		align-items: center;
	}

	.flex-1 {
		flex: 1;
	}

	.form-group {
		display: flex;
		flex-direction: column;
		gap: 6px;
	}

	.form-group label {
		font-size: 12px;
		font-weight: 500;
		color: var(--text-secondary, #b9b9b9);
	}

	.form-group select,
	.form-group input {
		padding: 8px 10px;
		background: var(--background-tertiary, #3a3a3a);
		border: 1px solid var(--border-color, #3a3a3a);
		border-radius: 4px;
		color: var(--text-primary, #ffffff);
		font-size: 13px;
	}

	.form-group select:focus,
	.form-group input:focus {
		outline: none;
		border-color: var(--accent-color, #5865f2);
	}

	.checkbox-label {
		display: flex;
		align-items: center;
		gap: 6px;
		font-size: 13px;
		color: var(--text-primary, #ffffff);
		cursor: pointer;
	}

	.form-actions {
		display: flex;
		justify-content: flex-end;
		gap: 12px;
		padding-top: 16px;
	}

	.btn-primary {
		padding: 10px 20px;
		background: var(--accent-color, #5865f2);
		color: white;
		border: none;
		border-radius: 4px;
		font-size: 14px;
		font-weight: 500;
		cursor: pointer;
		transition: background-color 0.15s;
	}

	.btn-primary:hover:not(:disabled) {
		background: var(--accent-color-hover, #4752c4);
	}

	.btn-primary:disabled {
		opacity: 0.6;
		cursor: not-allowed;
	}

	.btn-secondary {
		padding: 10px 20px;
		background: var(--background-tertiary, #3a3a3a);
		color: var(--text-primary, #ffffff);
		border: none;
		border-radius: 4px;
		font-size: 14px;
		font-weight: 500;
		cursor: pointer;
		transition: background-color 0.15s;
	}

	.btn-secondary:hover {
		background: var(--background-hover, #4a4a4a);
	}
</style>
