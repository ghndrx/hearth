<script lang="ts">
	import { channels, currentChannel, type Channel, dmChannels } from '$lib/stores/channels';
	import { user } from '$lib/stores/auth';
	import { goto } from '$app/navigation';
	import { createEventDispatcher } from 'svelte';
	import { getChannelUnread, getChannelMentionCount, markChannelRead } from '$lib/stores';
	import Avatar from './Avatar.svelte';

	const dispatch = createEventDispatcher<{
		openNewDM: void;
	}>();

	function selectDM(dm: Channel) {
		currentChannel.set(dm);
		markChannelRead(dm.id).catch((err: unknown) => console.error('Failed to mark DM as read:', err));
		goto(`/channels/@me/${dm.id}`);
	}

	function openFriends() {
		currentChannel.set(null);
		goto('/channels/@me');
	}

	function handleNewDM() {
		dispatch('openNewDM');
	}

	function getRecipientName(dm: Channel): string {
		if (dm.type === 3 && dm.name) return dm.name;
		return dm.recipients?.map(r => r.display_name || r.username).join(', ') || 'Unknown';
	}

	function getRecipientAvatar(dm: Channel): string | null {
		return dm.recipients?.[0]?.avatar || null;
	}

	function getRecipientInitial(dm: Channel): string {
		const name = dm.recipients?.[0]?.username || dm.name || '?';
		return name[0].toUpperCase();
	}
</script>

<div class="dm-sidebar">
	<div class="dm-search-bar">
		<button class="search-button" type="button" on:click={handleNewDM}>
			Find or start a conversation
		</button>
	</div>

	<div class="dm-nav">
		<button
			class="nav-item"
			class:active={!$currentChannel}
			on:click={openFriends}
			type="button"
		>
			<span class="nav-icon">👥</span>
			<span>Friends</span>
		</button>
	</div>

	<div class="dm-section-header">
		<span>DIRECT MESSAGES</span>
		<button class="add-dm-btn" type="button" on:click={handleNewDM} title="Create DM" aria-label="Create new direct message">
			+
		</button>
	</div>

	<ul class="dm-list" role="list" aria-label="Direct messages">
		{#each $dmChannels as dm (dm.id)}
			<li role="listitem">
				<button
					class="dm-item"
					class:active={$currentChannel?.id === dm.id}
					class:unread={getChannelUnread(dm.id)}
					on:click={() => selectDM(dm)}
					type="button"
					aria-current={$currentChannel?.id === dm.id ? 'page' : undefined}
				>
					<div class="dm-avatar">
						{#if dm.type === 3}
							<div class="group-avatar">👥</div>
						{:else if getRecipientAvatar(dm)}
							<img src={getRecipientAvatar(dm)} alt="" class="avatar-img" />
						{:else}
							<div class="avatar-placeholder">
								{getRecipientInitial(dm)}
							</div>
						{/if}
					</div>
					<div class="dm-info">
						<span class="dm-name">{getRecipientName(dm)}</span>
						{#if dm.type === 3}
							<span class="dm-members">{dm.recipients?.length || 0} Members</span>
						{/if}
					</div>
					<div class="dm-badges">
						{#if dm.e2ee_enabled}
							<span class="e2ee-badge" title="Encrypted">🔒</span>
						{/if}
						{#if getChannelUnread(dm.id)}
							{#if getChannelMentionCount(dm.id) > 0}
								<span class="mention-badge">{getChannelMentionCount(dm.id) > 99 ? '99+' : getChannelMentionCount(dm.id)}</span>
							{:else}
								<span class="unread-dot"></span>
							{/if}
						{/if}
					</div>
				</button>
			</li>
		{/each}
	</ul>
</div>

<style>
	.dm-sidebar {
		display: flex;
		flex-direction: column;
		height: 100%;
		overflow: hidden;
	}

	.dm-search-bar {
		padding: 10px 10px 6px;
		flex-shrink: 0;
	}

	.search-button {
		width: 100%;
		padding: 6px 8px;
		background: #1e1f22;
		border: none;
		border-radius: 4px;
		color: #949ba4;
		font-size: 13px;
		text-align: left;
		cursor: pointer;
	}

	.search-button:hover {
		background: #1a1b1e;
	}

	.dm-nav {
		padding: 6px 8px;
	}

	.nav-item {
		display: flex;
		align-items: center;
		gap: 12px;
		width: 100%;
		padding: 8px 12px;
		background: none;
		border: none;
		border-radius: 4px;
		color: #949ba4;
		font-size: 15px;
		font-weight: 500;
		cursor: pointer;
	}

	.nav-item:hover {
		background: #35373c;
		color: #dbdee1;
	}

	.nav-item.active {
		background: #43444b;
		color: #f2f3f5;
	}

	.nav-icon {
		font-size: 20px;
	}

	.dm-section-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 16px 16px 4px;
		font-size: 11px;
		font-weight: 700;
		text-transform: uppercase;
		letter-spacing: 0.02em;
		color: #949ba4;
	}

	.add-dm-btn {
		background: none;
		border: none;
		color: #949ba4;
		font-size: 18px;
		cursor: pointer;
		padding: 0 2px;
		line-height: 1;
	}

	.add-dm-btn:hover {
		color: #dbdee1;
	}

	.dm-list {
		flex: 1;
		overflow-y: auto;
		padding: 0 8px;
		list-style: none;
		margin: 0;
	}

	.dm-item {
		display: flex;
		align-items: center;
		gap: 12px;
		width: 100%;
		padding: 6px 8px;
		background: none;
		border: none;
		border-radius: 4px;
		color: #949ba4;
		cursor: pointer;
		text-align: left;
	}

	.dm-item:hover {
		background: #35373c;
		color: #dbdee1;
	}

	.dm-item.active {
		background: #43444b;
		color: #f2f3f5;
	}

	.dm-item.unread {
		color: #f2f3f5;
	}

	.dm-avatar {
		flex-shrink: 0;
		width: 32px;
		height: 32px;
	}

	.avatar-img {
		width: 32px;
		height: 32px;
		border-radius: 50%;
		object-fit: cover;
	}

	.avatar-placeholder, .group-avatar {
		width: 32px;
		height: 32px;
		border-radius: 50%;
		background: #5865f2;
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 14px;
		font-weight: 600;
		color: white;
	}

	.group-avatar {
		background: #3ba55d;
		font-size: 16px;
	}

	.dm-info {
		flex: 1;
		min-width: 0;
	}

	.dm-name {
		display: block;
		font-size: 15px;
		font-weight: 500;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.dm-members {
		display: block;
		font-size: 12px;
		color: #949ba4;
	}

	.dm-badges {
		display: flex;
		align-items: center;
		gap: 4px;
		flex-shrink: 0;
	}

	.e2ee-badge {
		font-size: 12px;
	}

	.mention-badge {
		background: #da373c;
		color: white;
		font-size: 11px;
		font-weight: 700;
		min-width: 16px;
		height: 16px;
		padding: 0 4px;
		border-radius: 8px;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.unread-dot {
		width: 8px;
		height: 8px;
		background: #f2f3f5;
		border-radius: 50%;
	}
</style>
