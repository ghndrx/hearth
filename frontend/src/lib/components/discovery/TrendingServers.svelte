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
		trend_score?: number;
		growth_rate?: number;
		rank_change?: number;
	}> = [];

	export let loading = false;
	export let joiningServerId: string | null = null;

	const dispatch = createEventDispatcher();

	function handleJoin(serverId: string) {
		dispatch('join', serverId);
	}

	function formatGrowthRate(rate: number): string {
		if (rate >= 100) {
			return '+' + rate.toFixed(0) + '%';
		}
		return '+' + rate.toFixed(1) + '%';
	}
</script>

<section class="trending-servers">
	<div class="section-header">
		<h2>
			<span class="trending-icon">📈</span>
			Trending Now
		</h2>
		<p class="section-subtitle">Servers with the most activity recently</p>
	</div>

	{#if loading}
		<div class="loading-state">
			<div class="spinner"></div>
			<p>Loading trending servers...</p>
		</div>
	{:else if servers.length === 0}
		<div class="empty-state">
			<p>No trending servers at the moment</p>
		</div>
	{:else}
		<div class="trending-list">
			{#each servers.filter(s => s.server_id || s.id) as server, index (server.server_id || server.id)}
				<div class="trending-item">
					<span class="rank">#{index + 1}</span>
					<div class="server-wrapper">
						<ServerCard
							server={server as any}
							variant="compact"
							onJoin={handleJoin}
							{joiningServerId}
						/>
					</div>
					{#if server.growth_rate}
						<span class="growth-badge" class:positive={server.growth_rate > 0}>
							{formatGrowthRate(server.growth_rate)}
						</span>
					{/if}
				</div>
			{/each}
		</div>
	{/if}
</section>

<style>
	.trending-servers {
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

	.trending-icon {
		font-size: 20px;
	}

	.section-subtitle {
		margin: 0;
		font-size: 13px;
		color: #6d6f78;
	}

	.trending-list {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	.trending-item {
		display: flex;
		align-items: center;
		gap: 12px;
		position: relative;
	}

	.rank {
		font-size: 14px;
		font-weight: 600;
		color: #6d6f78;
		min-width: 32px;
		text-align: center;
	}

	.server-wrapper {
		flex: 1;
		min-width: 0;
	}

	.growth-badge {
		position: absolute;
		right: 12px;
		top: 50%;
		transform: translateY(-50%);
		padding: 4px 8px;
		border-radius: 4px;
		font-size: 12px;
		font-weight: 600;
		background: #1e1f22;
		color: #6d6f78;
	}

	.growth-badge.positive {
		background: rgba(35, 165, 89, 0.2);
		color: #23a559;
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
