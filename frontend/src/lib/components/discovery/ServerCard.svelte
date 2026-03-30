<script lang="ts">
	export let server: {
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
		featured_at?: string;
		is_verified?: boolean;
		reason?: string;
		trend_score?: number;
		growth_rate?: number;
		invite_code?: string;
	};

	export let variant: 'default' | 'featured' | 'compact' | 'list' = 'default';
	export let onJoin: (serverId: string) => void = () => {};
	export let joiningServerId: string | null = null;

	$: displayId = server.server_id || server.id;
	$: isJoining = joiningServerId === displayId;
	$: description = server.short_description || server.description || '';

	function formatMemberCount(count: number): string {
		if (count >= 1000000) {
			return (count / 1000000).toFixed(1) + 'M';
		}
		if (count >= 1000) {
			return (count / 1000).toFixed(1) + 'K';
		}
		return count.toString();
	}

	function getInitials(name: string): string {
		return name
			.split(' ')
			.map(word => word[0])
			.join('')
			.toUpperCase()
			.slice(0, 2);
	}

	function getRandomColor(id: string): string {
		const colors = [
			'#5865f2', '#eb459e', '#3ba55d', '#f23f43', '#faa61a',
			'#2d7d46', '#91a6e6', '#f37b68', '#4f5d7e', '#72767d'
		];
		let hash = 0;
		for (let i = 0; i < id.length; i++) {
			hash = id.charCodeAt(i) + ((hash << 5) - hash);
		}
		return colors[Math.abs(hash) % colors.length];
	}

	function handleJoin() {
		onJoin(displayId);
	}
</script>

{#if variant === 'featured'}
	<article class="server-card featured">
		<div class="banner" style="background: linear-gradient(135deg, {getRandomColor(displayId)}40, {getRandomColor(displayId)}20);">
			{#if server.banner_url}
				<img src={server.banner_url} alt="" loading="lazy" />
			{/if}
			{#if server.is_featured}
				<span class="featured-badge">Featured</span>
			{/if}
		</div>
		<div class="content">
			<div class="icon-large" style="background-color: {getRandomColor(displayId)};">
				{#if server.icon_url}
					<img src={server.icon_url} alt="" loading="lazy" />
				{:else}
					<span>{getInitials(server.name)}</span>
				{/if}
			</div>
			<h3 class="name">{server.name}</h3>
			<p class="description">{description}</p>
			<div class="meta">
				<span class="member-count">
					<svg viewBox="0 0 24 24" fill="none" stroke="currentColor">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
					</svg>
					{formatMemberCount(server.member_count)} members
				</span>
				{#if server.is_verified}
					<span class="verified">
						<svg viewBox="0 0 24 24" fill="currentColor">
							<path d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
						</svg>
						Verified
					</span>
				{/if}
			</div>
			{#if server.tags && server.tags.length > 0}
				<div class="tags">
					{#each server.tags.slice(0, 3) as tag}
						<span class="tag">{tag}</span>
					{/each}
				</div>
			{/if}
			<button class="join-btn" on:click={handleJoin} disabled={isJoining}>
				{#if isJoining}
					<span class="spinner"></span>
				{:else}
					Join Server
				{/if}
			</button>
		</div>
	</article>
{:else if variant === 'compact'}
	<article class="server-card compact">
		<div class="icon" style="background-color: {getRandomColor(displayId)};">
			{#if server.icon_url}
				<img src={server.icon_url} alt="" loading="lazy" />
			{:else}
				<span>{getInitials(server.name)}</span>
			{/if}
		</div>
		<div class="info">
			<h3 class="name">{server.name}</h3>
			<p class="description">{description}</p>
			<div class="meta">
				<span class="member-count">
					<svg viewBox="0 0 24 24" fill="none" stroke="currentColor">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
					</svg>
					{formatMemberCount(server.member_count)}
				</span>
			</div>
		</div>
		<button class="join-btn compact-btn" on:click={handleJoin} disabled={isJoining}>
			{#if isJoining}
				<span class="spinner"></span>
			{:else}
				Join
			{/if}
		</button>
	</article>
{:else if variant === 'list'}
	<article class="server-card list">
		<div class="icon" style="background-color: {getRandomColor(displayId)};">
			{#if server.icon_url}
				<img src={server.icon_url} alt="" loading="lazy" />
			{:else}
				<span>{getInitials(server.name)}</span>
			{/if}
		</div>
		<div class="info">
			<h3 class="name">{server.name}</h3>
			<div class="meta">
				<span class="member-count">
					<svg viewBox="0 0 24 24" fill="none" stroke="currentColor">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
					</svg>
					{formatMemberCount(server.member_count)}
				</span>
				{#if server.category}
					<span class="category">{server.category}</span>
				{/if}
			</div>
		</div>
		<button class="join-btn compact-btn" on:click={handleJoin} disabled={isJoining}>
			{#if isJoining}
				<span class="spinner"></span>
			{:else}
				Join
			{/if}
		</button>
	</article>
{:else}
	<article class="server-card default">
		<div class="icon" style="background-color: {getRandomColor(displayId)};">
			{#if server.icon_url}
				<img src={server.icon_url} alt="" loading="lazy" />
			{:else}
				<span>{getInitials(server.name)}</span>
			{/if}
		</div>
		<div class="content">
			<h3 class="name">{server.name}</h3>
			<p class="description">{description}</p>
			<div class="meta">
				<span class="member-count">
					<svg viewBox="0 0 24 24" fill="none" stroke="currentColor">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
					</svg>
					{formatMemberCount(server.member_count)}
				</span>
			</div>
			{#if server.tags && server.tags.length > 0}
				<div class="tags">
					{#each server.tags.slice(0, 3) as tag}
						<span class="tag">{tag}</span>
					{/each}
				</div>
			{/if}
			<button class="join-btn" on:click={handleJoin} disabled={isJoining}>
				{#if isJoining}
					<span class="spinner"></span>
				{:else}
					Join
				{/if}
			</button>
		</div>
	</article>
{/if}

<style>
	.server-card {
		background: #2b2d31;
		border-radius: 8px;
		overflow: hidden;
		transition: transform 0.15s, box-shadow 0.15s;
		border: 1px solid #1e1f22;
	}

	.server-card:hover {
		transform: translateY(-2px);
		box-shadow: 0 8px 24px rgba(0, 0, 0, 0.3);
	}

	/* Featured variant */
	.server-card.featured {
		background: #2b2d31;
	}

	.banner {
		height: 80px;
		background: linear-gradient(135deg, #5865f240, #eb459e20);
		position: relative;
	}

	.banner img {
		width: 100%;
		height: 100%;
		object-fit: cover;
	}

	.featured-badge {
		position: absolute;
		top: 8px;
		right: 8px;
		padding: 4px 8px;
		background: rgba(0, 0, 0, 0.6);
		border-radius: 4px;
		font-size: 11px;
		font-weight: 600;
		color: #fff;
	}

	.content {
		padding: 0 16px 16px;
	}

	.icon-large {
		width: 72px;
		height: 72px;
		border-radius: 16px;
		background: #5865f2;
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 28px;
		font-weight: 700;
		color: white;
		margin-top: -36px;
		border: 4px solid #2b2d31;
		overflow: hidden;
	}

	.icon-large img {
		width: 100%;
		height: 100%;
		object-fit: cover;
	}

	/* Default/Compact variants */
	.server-card.default,
	.server-card.compact {
		padding: 12px;
		display: flex;
		gap: 12px;
	}

	.icon {
		width: 48px;
		height: 48px;
		border-radius: 12px;
		background: #5865f2;
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 16px;
		font-weight: 600;
		color: white;
		flex-shrink: 0;
		overflow: hidden;
	}

	.icon img {
		width: 100%;
		height: 100%;
		object-fit: cover;
	}

	.info {
		flex: 1;
		min-width: 0;
	}

	.name {
		margin: 0 0 4px 0;
		font-size: 16px;
		font-weight: 600;
		color: #f2f3f5;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.description {
		margin: 0 0 8px 0;
		font-size: 13px;
		color: #949ba4;
		line-height: 1.4;
		display: -webkit-box;
		-webkit-line-clamp: 2;
		-webkit-box-orient: vertical;
		overflow: hidden;
	}

	.meta {
		display: flex;
		align-items: center;
		gap: 12px;
		margin-bottom: 8px;
	}

	.member-count {
		display: flex;
		align-items: center;
		gap: 4px;
		font-size: 12px;
		color: #949ba4;
	}

	.member-count svg {
		width: 14px;
		height: 14px;
	}

	.verified {
		display: flex;
		align-items: center;
		gap: 4px;
		font-size: 11px;
		color: #23a559;
	}

	.verified svg {
		width: 12px;
		height: 12px;
	}

	.category {
		font-size: 11px;
		color: #6d6f78;
		text-transform: capitalize;
	}

	.tags {
		display: flex;
		flex-wrap: wrap;
		gap: 6px;
		margin-bottom: 12px;
	}

	.tag {
		padding: 2px 8px;
		background: #1e1f22;
		border-radius: 4px;
		font-size: 11px;
		color: #949ba4;
		font-weight: 500;
	}

	.join-btn {
		width: 100%;
		padding: 10px;
		background: #23a559;
		border: none;
		border-radius: 6px;
		color: white;
		font-size: 14px;
		font-weight: 500;
		cursor: pointer;
		transition: background-color 0.15s;
		display: flex;
		align-items: center;
		justify-content: center;
		min-height: 36px;
	}

	.join-btn:hover:not(:disabled) {
		background: #1a8f4a;
	}

	.join-btn:disabled {
		opacity: 0.7;
		cursor: not-allowed;
	}

	.compact-btn {
		width: auto;
		padding: 8px 16px;
		min-height: 32px;
		align-self: center;
	}

	.spinner {
		display: inline-block;
		width: 16px;
		height: 16px;
		border: 2px solid rgba(255, 255, 255, 0.3);
		border-top-color: white;
		border-radius: 50%;
		animation: spin 0.8s linear infinite;
	}

	@keyframes spin {
		to { transform: rotate(360deg); }
	}

	/* List variant */
	.server-card.list {
		padding: 12px 16px;
		display: flex;
		align-items: center;
		gap: 12px;
	}

	.server-card.list .icon {
		width: 48px;
		height: 48px;
		border-radius: 50%;
	}
</style>
