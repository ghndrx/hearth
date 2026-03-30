<script lang="ts">
	import ServerCard from './ServerCard.svelte';
	import { createEventDispatcher } from 'svelte';

	export let servers: Array<{
		id?: string;
		server_id?: string;
		name: string;
		description?: string;
		short_description?: string;
		icon_url?: string;
		banner_url?: string;
		member_count: number;
		category?: string;
		tags?: string[];
		is_featured?: boolean;
		reason?: string;
		mutual_member_count?: number;
		mutual_servers?: string[];
	}> = [];

	export let loading = false;
	export let joiningServerId: string | null = null;
	export let isAuthenticated = false;

	const dispatch = createEventDispatcher();

	function handleJoin(serverId: string) {
		dispatch('join', serverId);
	}

	$: validServers = (servers.filter(s => s.server_id || s.id) as Array<{
		id: string;
		server_id?: string;
		name: string;
		description?: string;
		short_description?: string;
		icon_url?: string;
		banner_url?: string;
		member_count: number;
		category?: string;
		tags?: string[];
		is_featured?: boolean;
		reason?: string;
		mutual_member_count?: number;
		mutual_servers?: string[];
	}>);
</script>

<section class="recommended-servers">
	<div class="section-header">
		<h2>
			<span class="recommended-icon">✨</span>
			Recommended for You
		</h2>
		<p class="section-subtitle">
			{#if isAuthenticated}
				Based on servers you've joined
			{:else}
				Popular servers you might like
			{/if}
		</p>
	</div>

	{#if !isAuthenticated}
		<div class="login-prompt">
			<p>Sign in to get personalized recommendations based on your interests</p>
			<a href="/login" class="login-btn">Sign In</a>
		</div>
	{/if}

	{#if loading}
		<div class="loading-state">
			<div class="spinner"></div>
			<p>Loading recommendations...</p>
		</div>
	{:else if servers.length === 0}
		<div class="empty-state">
			<p>Join some servers to get personalized recommendations</p>
		</div>
	{:else}
		<div class="recommended-grid">
			{#each validServers as server (server.server_id || server.id)}
				<div class="recommended-item">
					{#if server.reason}
						<div class="reason-badge">
							<span class="reason-icon">💡</span>
							{server.reason}
						</div>
					{/if}
					<ServerCard
						{server}
						variant="featured"
						onJoin={handleJoin}
						{joiningServerId}
					/>
				</div>
			{/each}
		</div>
	{/if}
</section>

<style>
	.recommended-servers {
		margin-bottom: 32px;
	}

	.section-header {
		margin-bottom: 16px;
	}

	.section-header h2 {
		display: flex;
		align-items: center;
		gap: 8px;
		margin: 0 0 4px 0;
		font-size: 18px;
		font-weight: 600;
		color: #f2f3f5;
	}

	.recommended-icon {
		font-size: 20px;
	}

	.section-subtitle {
		margin: 0;
		font-size: 13px;
		color: #6d6f78;
	}

	.login-prompt {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 16px;
		background: #2b2d31;
		border-radius: 8px;
		margin-bottom: 16px;
	}

	.login-prompt p {
		margin: 0;
		color: #949ba4;
		font-size: 14px;
	}

	.login-btn {
		padding: 8px 16px;
		background: #5865f2;
		border-radius: 6px;
		color: white;
		text-decoration: none;
		font-size: 14px;
		font-weight: 500;
		transition: background-color 0.15s;
	}

	.login-btn:hover {
		background: #4752c4;
	}

	.recommended-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
		gap: 16px;
	}

	.recommended-item {
		position: relative;
	}

	.reason-badge {
		position: absolute;
		top: 8px;
		left: 8px;
		z-index: 10;
		display: flex;
		align-items: center;
		gap: 6px;
		padding: 6px 10px;
		background: rgba(0, 0, 0, 0.7);
		border-radius: 6px;
		font-size: 12px;
		color: #f2f3f5;
		backdrop-filter: blur(4px);
	}

	.reason-icon {
		font-size: 14px;
	}

	.loading-state,
	.empty-state {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: 32px;
		color: #6d6f78;
	}

	.spinner {
		width: 32px;
		height: 32px;
		border: 3px solid #3f4147;
		border-top-color: #5865f2;
		border-radius: 50%;
		animation: spin 1s linear infinite;
		margin-bottom: 12px;
	}

	@keyframes spin {
		to { transform: rotate(360deg); }
	}
</style>
