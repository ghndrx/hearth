<script lang="ts">
	import { user } from '$lib/stores/auth';
</script>

<svelte:head>
	<title>Friends | Hearth</title>
</svelte:head>

<div class="friends-header">
	<div class="header-left">
		<span class="icon">👥</span>
		<span class="title">Friends</span>
	</div>
	
	<div class="header-tabs">
		<button class="tab active">Online</button>
		<button class="tab">All</button>
		<button class="tab">Pending</button>
		<button class="tab">Blocked</button>
		<button class="tab add">Add Friend</button>
	</div>
</div>

<div class="friends-content">
	<div class="empty-state">
		<div class="illustration">🤝</div>
		<h2>Wumpus is waiting on friends</h2>
		<p>You can add friends by their username.</p>
	</div>
</div>

<style>
	.friends-header {
		display: flex;
		align-items: center;
		gap: 16px;
		padding: 0 16px;
		height: 48px;
		border-bottom: 1px solid var(--bg-modifier-accent);
		background: var(--bg-primary);
		flex-shrink: 0;
	}
	
	.header-left {
		display: flex;
		align-items: center;
		gap: 8px;
	}
	
	.icon {
		font-size: 24px;
	}
	
	.title {
		font-weight: 600;
		color: var(--text-primary);
	}
	
	.header-tabs {
		display: flex;
		gap: 16px;
		margin-left: 16px;
		padding-left: 16px;
		border-left: 1px solid var(--bg-modifier-accent);
	}
	
	.tab {
		background: none;
		border: none;
		color: var(--text-muted);
		font-size: 14px;
		font-weight: 500;
		cursor: pointer;
		padding: 4px 8px;
		border-radius: 4px;
	}
	
	.tab:hover {
		background: var(--bg-modifier-hover);
		color: var(--text-primary);
	}
	
	.tab.active {
		background: var(--bg-modifier-selected);
		color: var(--text-primary);
	}
	
	.tab.add {
		background: var(--text-positive);
		color: white;
	}
	
	.tab.add:hover {
		background: #1a8f4a;
	}
	
	.friends-content {
		flex: 1;
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 32px;
	}
	
	.empty-state {
		text-align: center;
		max-width: 400px;
	}
	
	.illustration {
		font-size: 100px;
		margin-bottom: 24px;
	}
	
	.empty-state h2 {
		font-size: 20px;
		color: var(--text-primary);
		margin-bottom: 8px;
	}
	
	.empty-state p {
		color: var(--text-muted);
	}
</style>
