<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import type { Application, SlashCommand } from '$lib/services/slashCommands';

	export let applications: Application[] = [];
	export let selectedApp: Application | null = null;
	export let commands: SlashCommand[] = [];
	export let loading = false;

	const dispatch = createEventDispatcher<{
		selectApp: { app: Application };
		createApp: void;
		editApp: { app: Application };
		deleteApp: { app: Application };
		selectCommand: { command: SlashCommand };
		createCommand: void;
		editCommand: { command: SlashCommand };
		deleteCommand: { command: SlashCommand };
	}>();

	function formatDate(dateStr: string): string {
		if (!dateStr) return 'N/A';
		return new Date(dateStr).toLocaleDateString('en-US', {
			year: 'numeric',
			month: 'short',
			day: 'numeric'
		});
	}
</script>

<div class="developer-dashboard">
	<div class="sidebar">
		<div class="sidebar-header">
			<h2>Developer Dashboard</h2>
			<button class="btn-primary" on:click={() => dispatch('createApp')}>
				+ New App
			</button>
		</div>

		<div class="app-list">
			{#if loading && applications.length === 0}
				<div class="loading-placeholder">Loading applications...</div>
			{:else if applications.length === 0}
				<div class="empty-state">
					<p>No applications yet</p>
					<button class="btn-secondary" on:click={() => dispatch('createApp')}>
						Create your first app
					</button>
				</div>
			{:else}
				{#each applications as app}
					<button
						class="app-item"
						class:selected={selectedApp?.id === app.id}
						on:click={() => dispatch('selectApp', { app })}
					>
						<div class="app-icon">
							{#if app.icon}
								<img src={app.icon} alt={app.name} />
							{:else}
								<span class="default-icon">{app.name.charAt(0).toUpperCase()}</span>
							{/if}
						</div>
						<div class="app-info">
							<span class="app-name">{app.name}</span>
							{#if app.verified}
								<span class="verified-badge">✓</span>
							{/if}
						</div>
					</button>
				{/each}
			{/if}
		</div>
	</div>

	<div class="main-content">
		{#if selectedApp}
			<div class="app-header">
				<div class="app-title">
					<div class="app-icon large">
						{#if selectedApp.icon}
							<img src={selectedApp.icon} alt={selectedApp.name} />
						{:else}
							<span class="default-icon">{selectedApp.name.charAt(0).toUpperCase()}</span>
						{/if}
					</div>
					<div>
						<h3>{selectedApp.name}</h3>
						<p class="app-description">{selectedApp.description || 'No description'}</p>
					</div>
				</div>
				<div class="app-actions">
					<button class="btn-secondary" on:click={() => dispatch('editApp', { app: selectedApp })}>
						Edit
					</button>
					<button class="btn-danger" on:click={() => dispatch('deleteApp', { app: selectedApp })}>
						Delete
					</button>
				</div>
			</div>

			<div class="section">
				<div class="section-header">
					<h4>Slash Commands</h4>
					<button class="btn-primary" on:click={() => dispatch('createCommand')}>
						+ New Command
					</button>
				</div>

				{#if loading}
					<div class="loading-placeholder">Loading commands...</div>
				{:else if commands.length === 0}
					<div class="empty-state">
						<p>No commands yet</p>
						<button class="btn-secondary" on:click={() => dispatch('createCommand')}>
							Create your first command
						</button>
					</div>
				{:else}
					<div class="command-list">
						{#each commands as command}
							<div class="command-item">
								<div class="command-info">
									<span class="command-name">/{command.name}</span>
									<span class="command-description">{command.description}</span>
								</div>
								<div class="command-meta">
									<span class="command-version">v{command.version?.substring(0, 8) || '1'}</span>
									<span class="command-date">{formatDate(command.created_at)}</span>
								</div>
								<div class="command-actions">
									<button
										class="btn-icon"
										title="Edit"
										on:click={() => dispatch('editCommand', { command })}
									>
										✏️
									</button>
									<button
										class="btn-icon"
										title="Delete"
										on:click={() => dispatch('deleteCommand', { command })}
									>
										🗑️
									</button>
								</div>
							</div>
						{/each}
					</div>
				{/if}
			</div>

			<div class="section">
				<h4>Quick Stats</h4>
				<div class="stats-grid">
					<div class="stat-card">
						<span class="stat-value">{commands.length}</span>
						<span class="stat-label">Commands</span>
					</div>
					<div class="stat-card">
						<span class="stat-value">0</span>
						<span class="stat-label">Executions Today</span>
					</div>
					<div class="stat-card">
						<span class="stat-value">0</span>
						<span class="stat-label">Total Executions</span>
					</div>
				</div>
			</div>
		{:else}
			<div class="empty-state large">
				<h3>Select an Application</h3>
				<p>Choose an application from the sidebar to manage its settings and commands.</p>
			</div>
		{/if}
	</div>
</div>

<style>
	.developer-dashboard {
		display: flex;
		height: 100%;
		background: var(--background-primary, #1a1a1a);
		color: var(--text-primary, #ffffff);
	}

	.sidebar {
		width: 280px;
		border-right: 1px solid var(--border-color, #3a3a3a);
		display: flex;
		flex-direction: column;
	}

	.sidebar-header {
		padding: 16px;
		border-bottom: 1px solid var(--border-color, #3a3a3a);
	}

	.sidebar-header h2 {
		font-size: 14px;
		font-weight: 600;
		margin-bottom: 12px;
		color: var(--text-secondary, #b9b9b9);
	}

	.app-list {
		flex: 1;
		overflow-y: auto;
		padding: 8px;
	}

	.app-item {
		display: flex;
		align-items: center;
		gap: 12px;
		width: 100%;
		padding: 10px;
		border: none;
		border-radius: 6px;
		background: transparent;
		color: var(--text-primary, #ffffff);
		cursor: pointer;
		text-align: left;
		transition: background-color 0.15s;
	}

	.app-item:hover {
		background: var(--background-hover, #2a2a2a);
	}

	.app-item.selected {
		background: var(--background-active, #3a3a3a);
	}

	.app-icon {
		width: 36px;
		height: 36px;
		border-radius: 6px;
		background: var(--accent-color, #5865f2);
		display: flex;
		align-items: center;
		justify-content: center;
		overflow: hidden;
	}

	.app-icon.large {
		width: 48px;
		height: 48px;
	}

	.app-icon img {
		width: 100%;
		height: 100%;
		object-fit: cover;
	}

	.default-icon {
		font-size: 16px;
		font-weight: 600;
		color: white;
	}

	.app-info {
		display: flex;
		align-items: center;
		gap: 6px;
	}

	.app-name {
		font-size: 14px;
		font-weight: 500;
	}

	.verified-badge {
		color: var(--success-color, #23a559);
		font-size: 12px;
	}

	.main-content {
		flex: 1;
		padding: 20px;
		overflow-y: auto;
	}

	.app-header {
		display: flex;
		justify-content: space-between;
		align-items: flex-start;
		margin-bottom: 24px;
		padding-bottom: 16px;
		border-bottom: 1px solid var(--border-color, #3a3a3a);
	}

	.app-title {
		display: flex;
		gap: 16px;
		align-items: center;
	}

	.app-title h3 {
		font-size: 20px;
		font-weight: 600;
		margin-bottom: 4px;
	}

	.app-description {
		font-size: 13px;
		color: var(--text-secondary, #b9b9b9);
	}

	.app-actions {
		display: flex;
		gap: 8px;
	}

	.section {
		margin-bottom: 24px;
	}

	.section-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-bottom: 12px;
	}

	.section h4 {
		font-size: 14px;
		font-weight: 600;
		color: var(--text-secondary, #b9b9b9);
	}

	.command-list {
		display: flex;
		flex-direction: column;
		gap: 4px;
	}

	.command-item {
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 12px;
		background: var(--background-secondary, #2a2a2a);
		border-radius: 6px;
	}

	.command-info {
		flex: 1;
	}

	.command-name {
		font-size: 14px;
		font-weight: 500;
		font-family: monospace;
	}

	.command-description {
		display: block;
		font-size: 12px;
		color: var(--text-secondary, #b9b9b9);
		margin-top: 2px;
	}

	.command-meta {
		display: flex;
		gap: 12px;
		font-size: 12px;
		color: var(--text-muted, #727067);
	}

	.command-actions {
		display: flex;
		gap: 4px;
	}

	.btn-icon {
		padding: 6px 8px;
		background: transparent;
		border: none;
		border-radius: 4px;
		cursor: pointer;
		opacity: 0.7;
		transition: opacity 0.15s;
	}

	.btn-icon:hover {
		opacity: 1;
	}

	.stats-grid {
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		gap: 12px;
	}

	.stat-card {
		padding: 16px;
		background: var(--background-secondary, #2a2a2a);
		border-radius: 6px;
		text-align: center;
	}

	.stat-value {
		display: block;
		font-size: 24px;
		font-weight: 600;
		color: var(--accent-color, #5865f2);
	}

	.stat-label {
		font-size: 12px;
		color: var(--text-secondary, #b9b9b9);
	}

	.empty-state {
		text-align: center;
		padding: 24px;
		color: var(--text-secondary, #b9b9b9);
	}

	.empty-state.large {
		padding: 48px;
	}

	.empty-state p {
		margin-bottom: 12px;
	}

	.loading-placeholder {
		padding: 24px;
		text-align: center;
		color: var(--text-muted, #727067);
	}

	.btn-primary {
		padding: 8px 16px;
		background: var(--accent-color, #5865f2);
		color: white;
		border: none;
		border-radius: 4px;
		font-size: 13px;
		font-weight: 500;
		cursor: pointer;
		transition: background-color 0.15s;
	}

	.btn-primary:hover {
		background: var(--accent-color-hover, #4752c4);
	}

	.btn-secondary {
		padding: 8px 16px;
		background: var(--background-secondary, #3a3a3a);
		color: var(--text-primary, #ffffff);
		border: none;
		border-radius: 4px;
		font-size: 13px;
		font-weight: 500;
		cursor: pointer;
		transition: background-color 0.15s;
	}

	.btn-secondary:hover {
		background: var(--background-hover, #4a4a4a);
	}

	.btn-danger {
		padding: 8px 16px;
		background: var(--danger-color, #da373c);
		color: white;
		border: none;
		border-radius: 4px;
		font-size: 13px;
		font-weight: 500;
		cursor: pointer;
		transition: background-color 0.15s;
	}

	.btn-danger:hover {
		background: var(--danger-color-hover, #c42b31);
	}
</style>
