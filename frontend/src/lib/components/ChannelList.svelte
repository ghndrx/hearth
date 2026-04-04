<script lang="ts">
	import { channels, currentChannel, type Channel, type ChannelTypeString, categorizedChannels, reorderChannels, updateChannel, deleteChannel } from '$lib/stores/channels';
	import { currentServer, leaveServer } from '$lib/stores/servers';
	import { user } from '$lib/stores/auth';
	import { settings } from '$lib/stores/settings';
	import { voiceChannelStates, voiceState, voiceActions, isInVoice } from '$lib/stores/voice';
	import { getVoiceConnectionManager } from '$lib/voice/connection';
	import { api } from '$lib/api';
	import { createEventDispatcher, onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import UserPanel from './UserPanel.svelte';
	import ServerHeader from './ServerHeader.svelte';
	import ChannelCategory from './ChannelCategory.svelte';
	import ChannelItem from './ChannelItem.svelte';
	import { unreadChannels, markChannelRead, getChannelUnread, getChannelMentionCount } from '$lib/stores';
	import VoiceMiniPlayer from './VoiceMiniPlayer.svelte';
	import InviteModal from './InviteModal.svelte';
	import CreateChannelModal from './CreateChannelModal.svelte';
	import NewDMModal from './NewDMModal.svelte';

	const dispatch = createEventDispatcher<{
		openQuickSwitcher: void;
	}>();

	let showNewDMModal = false;

	function openQuickSwitcher() {
		dispatch('openQuickSwitcher');
	}

	let collapsedCategories: Record<string, boolean> = {};
	let showInviteModal = false;
	let showCreateChannelModal = false;
	let createChannelType: ChannelTypeString = 'text';
	let createChannelCategoryId: string | undefined = undefined;
	let inviteModalComponent: InviteModal;

	// Drag-drop state
	let draggedChannel: Channel | null = null;
	let dragOverCategoryId: string | null = null;
	let dragOverPosition: number | null = null;

	// Get voice users from store, transformed to match ChannelItem's expected format
	$: voiceConnectedUsers = Object.fromEntries(
		Object.entries($voiceChannelStates).map(([channelId, users]) => [
			channelId,
			users.map(u => ({
				id: u.id,
				username: u.display_name || u.username,
				avatar: u.avatar || null,
				speaking: u.speaking
			}))
		])
	);

	// Fetch voice states when server changes
	onMount(() => {
		if ($currentServer?.id) {
			fetchVoiceStates($currentServer.id);
		}
	});

	$: if ($currentServer?.id) {
		fetchVoiceStates($currentServer.id);
	}

	async function fetchVoiceStates(serverId: string) {
		try {
			const states = await api.get<Array<{
				user_id: string;
				username: string;
				display_name?: string;
				avatar?: string;
				channel_id: string;
				self_muted: boolean;
				self_deafened: boolean;
				self_video: boolean;
				self_stream: boolean;
				muted: boolean;
				deafened: boolean;
			}>>(`/servers/${serverId}/voice-states`);

			// Group by channel
			const byChannel: Record<string, typeof states> = {};
			for (const state of states) {
				if (!byChannel[state.channel_id]) {
					byChannel[state.channel_id] = [];
				}
				byChannel[state.channel_id].push(state);
			}

			// Update voice channel states store
			for (const [channelId, channelUsers] of Object.entries(byChannel)) {
				voiceChannelStates.setChannelUsers(channelId, channelUsers.map(u => ({
					id: u.user_id,
					username: u.username,
					display_name: u.display_name,
					avatar: u.avatar,
					self_muted: u.self_muted,
					self_deafened: u.self_deafened,
					self_video: u.self_video,
					self_stream: u.self_stream,
					muted: u.muted,
					deafened: u.deafened,
					speaking: false
				})));
			}
		} catch (error) {
			console.error('Failed to fetch voice states:', error);
		}
	}

	$: isOwner = $currentServer?.owner_id === $user?.id;

	function selectChannel(channel: Channel) {
		currentChannel.set(channel);
		markChannelRead(channel.id).catch((err: unknown) => console.error('Failed to mark channel as read:', err));
		if (channel.server_id) {
			goto(`/channels/${channel.server_id}/${channel.id}`);
		} else {
			goto(`/channels/@me/${channel.id}`);
		}
	}

	function openServerSettings() {
		settings.openServerSettings();
	}

	async function handleLeaveServer() {
		if (!$currentServer) return;
		if (!confirm(`Are you sure you want to leave ${$currentServer.name}?`)) return;
		try {
			await leaveServer($currentServer.id);
			currentServer.set(null);
			goto('/channels/@me');
		} catch (error) {
			console.error('Failed to leave server:', error);
		}
	}

	function handleInvitePeople() {
		showInviteModal = true;
	}

	function handleInviteClose() {
		showInviteModal = false;
	}

	function handleInviteCreated(event: CustomEvent<{ code: string; maxUses: number; expiresIn: number }>) {
		// Invite creation handled by parent component
	}

	async function handleGenerateInvite(event: CustomEvent<{ maxUses: number; expiresIn: number }>) {
		if (!$currentServer) return;

		try {
			const channelId = $currentChannel?.id;
			const response = await api.post<{ code: string }>(`/servers/${$currentServer.id}/invites`, {
				channel_id: channelId,
				max_uses: event.detail.maxUses || 0,
				max_age: event.detail.expiresIn || 604800
			});

			if (inviteModalComponent && response?.code) {
				inviteModalComponent.onInviteGenerated(response.code);
			}
		} catch (error) {
			console.error('Failed to generate invite:', error);
		}
	}

	function handleAddChannel(categoryId?: string) {
		createChannelType = 'text';
		createChannelCategoryId = categoryId;
		showCreateChannelModal = true;
	}

	function handleAddCategory() {
		createChannelType = 'category';
		createChannelCategoryId = undefined;
		showCreateChannelModal = true;
	}

	function handleCreateChannelClose() {
		showCreateChannelModal = false;
	}

	function handleChannelCreated(event: CustomEvent<Channel>) {
		const channel = event.detail;
		if (channel.server_id && channel.type !== 4) {
			goto(`/channels/${channel.server_id}/${channel.id}`);
		}
	}

	function handleChannelSettings(event: CustomEvent<Channel>) {
		// Channel settings functionality to be implemented
	}

	function handleOpenNewDM() {
		showNewDMModal = true;
	}

	function handleNewDMClose() {
		showNewDMModal = false;
	}

	function handleDMSelected(event: CustomEvent<{ channelId: string }>) {
		showNewDMModal = false;
	}

	function toggleCategory(categoryId: string) {
		collapsedCategories[categoryId] = !collapsedCategories[categoryId];
		collapsedCategories = collapsedCategories;
	}

	// Drag-drop handlers
	function handleDragStart(event: DragEvent, channel: Channel) {
		if (!event.dataTransfer) return;
		draggedChannel = channel;
		event.dataTransfer.effectAllowed = 'move';
		event.dataTransfer.setData('text/plain', channel.id);
	}

	function handleDragOver(event: DragEvent, categoryId: string | null, position: number) {
		event.preventDefault();
		if (!event.dataTransfer) return;
		event.dataTransfer.dropEffect = 'move';
		dragOverCategoryId = categoryId;
		dragOverPosition = position;
	}

	function handleDragLeave() {
		dragOverCategoryId = null;
		dragOverPosition = null;
	}

	async function handleDrop(event: DragEvent, targetCategoryId: string | null, targetPosition: number) {
		event.preventDefault();
		if (!draggedChannel) return;

		const serverChs = $channels
			.filter(c => c.server_id === $currentServer?.id && c.type !== 4)
			.sort((a, b) => a.position - b.position);

		// Build reorder entries: move dragged channel to new position/category
		const entries = [];
		const targetChannels = serverChs.filter(c => c.parent_id === targetCategoryId && c.id !== draggedChannel!.id);

		// Insert at position
		targetChannels.splice(targetPosition, 0, draggedChannel);

		for (let i = 0; i < targetChannels.length; i++) {
			entries.push({
				id: targetChannels[i].id,
				category_id: targetCategoryId,
				position: i
			});
		}

		// If channel moved out of a category, re-index the old category
		if (draggedChannel.parent_id !== targetCategoryId && draggedChannel.parent_id) {
			const oldCategoryChannels = serverChs
				.filter(c => c.parent_id === draggedChannel!.parent_id && c.id !== draggedChannel!.id);
			for (let i = 0; i < oldCategoryChannels.length; i++) {
				if (!entries.find(e => e.id === oldCategoryChannels[i].id)) {
					entries.push({
						id: oldCategoryChannels[i].id,
						category_id: draggedChannel.parent_id,
						position: i
					});
				}
			}
		}

		// Also handle uncategorized source
		if (draggedChannel.parent_id !== targetCategoryId && !draggedChannel.parent_id) {
			const oldUncategorized = serverChs
				.filter(c => !c.parent_id && c.id !== draggedChannel!.id);
			for (let i = 0; i < oldUncategorized.length; i++) {
				if (!entries.find(e => e.id === oldUncategorized[i].id)) {
					entries.push({
						id: oldUncategorized[i].id,
						category_id: null,
						position: i
					});
				}
			}
		}

		try {
			await reorderChannels(entries);
		} catch (error) {
			console.error('Failed to reorder:', error);
		}

		draggedChannel = null;
		dragOverCategoryId = null;
		dragOverPosition = null;
	}

	function handleDragEnd() {
		draggedChannel = null;
		dragOverCategoryId = null;
		dragOverPosition = null;
	}

	// Category management
	async function handleRenameCategory(event: CustomEvent<{ id: string; name: string }>) {
		try {
			await updateChannel(event.detail.id, { name: event.detail.name });
		} catch (error) {
			console.error('Failed to rename category:', error);
		}
	}

	async function handleDeleteCategory(event: CustomEvent<{ id: string }>) {
		if (!confirm('Delete this category? Channels inside will be moved to uncategorized.')) return;

		// Move children to uncategorized first
		const categoryChannels = $channels.filter(c => c.parent_id === event.detail.id);
		if (categoryChannels.length > 0) {
			const entries = categoryChannels.map((c, i) => ({
				id: c.id,
				category_id: null,
				position: i
			}));
			try {
				await reorderChannels(entries);
			} catch (error) {
				console.error('Failed to move channels:', error);
			}
		}

		try {
			await deleteChannel(event.detail.id);
		} catch (error) {
			console.error('Failed to delete category:', error);
		}
	}
</script>

<nav class="channel-list" aria-label="Channels and direct messages">
	<div class="channel-list-content">
		{#if $currentServer}
			<ServerHeader
				server={$currentServer}
				{isOwner}
				on:openSettings={openServerSettings}
				on:leaveServer={handleLeaveServer}
				on:invitePeople={handleInvitePeople}
			/>

			<!-- Uncategorized channels (top-level, no parent_id) -->
			{#if $categorizedChannels.uncategorized.length > 0}
				<div class="uncategorized-channels" role="group" aria-label="Channels">
					{#each $categorizedChannels.uncategorized as channel, idx (channel.id)}
						<div
							class="drag-target"
							class:drag-over={dragOverCategoryId === '__uncategorized' && dragOverPosition === idx}
							draggable="true"
							on:dragstart={(e) => handleDragStart(e, channel)}
							on:dragover={(e) => handleDragOver(e, null, idx)}
							on:dragleave={handleDragLeave}
							on:drop={(e) => handleDrop(e, null, idx)}
							on:dragend={handleDragEnd}
							role="listitem"
						>
							<ChannelItem
								{channel}
								active={$currentChannel?.id === channel.id}
								unread={$unreadChannels.has(channel.id)}
								connectedUsers={channel.type === 2 ? (voiceConnectedUsers[channel.id] || []) : []}
								on:select={(e) => selectChannel(e.detail)}
								on:openSettings={handleChannelSettings}
							/>
						</div>
					{/each}
				</div>
			{/if}

			<!-- Server-defined categories -->
			{#each $categorizedChannels.categories as category (category.id)}
				<ChannelCategory
					name={category.name}
					categoryId={category.id}
					collapsed={collapsedCategories[category.id] || false}
					editable={true}
					on:toggle={() => toggleCategory(category.id)}
					on:addChannel={() => handleAddChannel(category.id)}
					on:rename={handleRenameCategory}
					on:deleteCategory={handleDeleteCategory}
				>
					{#if category.channels.length > 0}
						{#each category.channels as channel, idx (channel.id)}
							<div
								class="drag-target"
								class:drag-over={dragOverCategoryId === category.id && dragOverPosition === idx}
								draggable="true"
								on:dragstart={(e) => handleDragStart(e, channel)}
								on:dragover={(e) => handleDragOver(e, category.id, idx)}
								on:dragleave={handleDragLeave}
								on:drop={(e) => handleDrop(e, category.id, idx)}
								on:dragend={handleDragEnd}
								role="listitem"
							>
								<ChannelItem
									{channel}
									active={$currentChannel?.id === channel.id}
									unread={$unreadChannels.has(channel.id)}
									connectedUsers={channel.type === 2 ? (voiceConnectedUsers[channel.id] || []) : []}
									on:select={(e) => selectChannel(e.detail)}
									on:openSettings={handleChannelSettings}
								/>
							</div>
						{/each}
					{:else}
						<!-- Empty drop target for empty categories -->
						<div
							class="empty-category-drop"
							class:drag-over={dragOverCategoryId === category.id && dragOverPosition === 0}
							on:dragover={(e) => handleDragOver(e, category.id, 0)}
							on:dragleave={handleDragLeave}
							on:drop={(e) => handleDrop(e, category.id, 0)}
							role="listitem"
						>
							<span class="no-channels-hint">No channels</span>
						</div>
					{/if}
				</ChannelCategory>
			{/each}

			<!-- Add Category button -->
			<div class="add-category-row">
				<button
					class="add-category-btn"
					on:click={handleAddCategory}
					title="Create Category"
					type="button"
				>
					<svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor" aria-hidden="true">
						<path d="M19 13h-6v6h-2v-6H5v-2h6V5h2v6h6v2z"/>
					</svg>
					<span>Create Category</span>
				</button>
				{#if $categorizedChannels.categories.length === 0 && $categorizedChannels.uncategorized.length === 0}
					<button
						class="add-category-btn"
						on:click={() => handleAddChannel()}
						title="Create Channel"
						type="button"
					>
						<svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor" aria-hidden="true">
							<path d="M19 13h-6v6h-2v-6H5v-2h6V5h2v6h6v2z"/>
						</svg>
						<span>Create Channel</span>
					</button>
				{/if}
			</div>
		{:else}
			<!-- DM List -->
			<div class="dm-header">
				<button class="dm-search" aria-label="Find or start a conversation" type="button" on:click={handleOpenNewDM}>Find or start a conversation</button>
			</div>

			<div class="dm-section" role="heading" aria-level="2">
				<span>DIRECT MESSAGES</span>
			</div>

			<ul role="list" aria-label="Direct messages" class="dm-list">
				{#each $channels.filter((c) => c.type === 1 || c.type === 3) as dm (dm.id)}
					<li role="listitem">
						<button
							class="dm-item"
							class:active={$currentChannel?.id === dm.id}
							class:unread={getChannelUnread(dm.id)}
							on:click={() => selectChannel(dm)}
							aria-current={$currentChannel?.id === dm.id ? 'page' : undefined}
							aria-label="Direct message with {dm.name || dm.recipients?.map((r) => r.display_name || r.username).join(', ') || 'Unknown'}{dm.e2ee_enabled ? ', encrypted' : ''}{getChannelUnread(dm.id) ? ', unread messages' : ''}"
							type="button"
						>
							<div class="dm-avatar">
								{#if dm.recipients?.[0]?.avatar}
									<img src={dm.recipients[0].avatar} alt="" />
								{:else}
									<div class="avatar-placeholder">
										{(dm.recipients?.[0]?.username || '?')[0].toUpperCase()}
									</div>
								{/if}
							</div>
							<span class="dm-name">
								{dm.name || dm.recipients?.map((r) => r.display_name || r.username).join(', ') || 'Unknown'}
							</span>
							{#if dm.e2ee_enabled}
								<span class="e2ee-indicator" aria-hidden="true">🔒</span>
							{/if}
							{#if getChannelUnread(dm.id)}
								{#if getChannelMentionCount(dm.id) > 0}
									<span class="mention-badge">{getChannelMentionCount(dm.id) > 99 ? '99+' : getChannelMentionCount(dm.id)}</span>
								{:else}
									<span class="unread-dot" aria-hidden="true"></span>
								{/if}
							{/if}
						</button>
					</li>
				{/each}
			</ul>
		{/if}
	</div>

	<!-- FEAT-002: Voice Mini-Player - Shows when in a voice channel -->
	<VoiceMiniPlayer
		on:disconnect={() => {}}
		on:expandFullView={() => {}}
	/>

	<UserPanel />
</nav>

<!-- Invite Modal -->
{#if $currentServer}
	<InviteModal
		bind:this={inviteModalComponent}
		open={showInviteModal}
		serverName={$currentServer.name}
		serverId={$currentServer.id}
		channelName={$currentChannel?.name ?? ''}
		_channelId={$currentChannel?.id ?? ''}
		on:close={handleInviteClose}
		on:invite={handleInviteCreated}
		on:generateInvite={handleGenerateInvite}
	/>

	<!-- Create Channel Modal -->
	<CreateChannelModal
		open={showCreateChannelModal}
		defaultType={createChannelType}
		categoryId={createChannelCategoryId}
		on:close={handleCreateChannelClose}
		on:created={handleChannelCreated}
	/>
{/if}

<!-- New DM Modal -->
<NewDMModal
	open={showNewDMModal}
	on:close={handleNewDMClose}
	on:select={handleDMSelected}
/>

<style>
	.channel-list {
		display: flex;
		flex-direction: column;
		width: 240px;
		height: 100%;
		background: #2b2d31;
		flex-shrink: 0;
	}

	.channel-list-content {
		flex: 1;
		overflow-y: auto;
	}

	.uncategorized-channels {
		padding-top: 8px;
	}

	/* Drag-drop styles */
	.drag-target {
		transition: border-color 0.1s ease;
		border-top: 2px solid transparent;
	}

	.drag-target.drag-over {
		border-top-color: #5865f2;
	}

	.empty-category-drop {
		padding: 4px 8px;
		min-height: 24px;
		border-top: 2px solid transparent;
	}

	.empty-category-drop.drag-over {
		border-top-color: #5865f2;
		background: rgba(88, 101, 242, 0.1);
	}

	/* Add Category button */
	.add-category-row {
		padding: 8px;
		display: flex;
		flex-direction: column;
		gap: 4px;
	}

	.add-category-btn {
		display: flex;
		align-items: center;
		gap: 6px;
		padding: 6px 8px;
		background: none;
		border: none;
		color: #949ba4;
		font-size: 13px;
		cursor: pointer;
		border-radius: 4px;
		width: 100%;
		text-align: left;
		transition: background-color 0.15s ease, color 0.15s ease;
	}

	.add-category-btn:hover {
		background: #35373c;
		color: #dbdee1;
	}

	/* DM styles */
	.dm-header {
		padding: 10px 8px;
	}

	.dm-search {
		width: 100%;
		padding: 6px 8px;
		background: #1e1f22;
		border: none;
		border-radius: 4px;
		color: #949ba4;
		font-size: 14px;
		cursor: pointer;
		text-align: left;
		transition: background-color 0.15s ease;
	}

	.dm-search:hover {
		background: #404249;
	}

	.dm-section {
		padding: 16px 8px 4px 16px;
		font-size: 12px;
		font-weight: 600;
		color: #949ba4;
		letter-spacing: 0.02em;
	}

	.dm-list {
		list-style: none;
		margin: 0;
		padding: 0;
	}

	.dm-list li {
		list-style: none;
		margin: 0;
		padding: 0;
	}

	.dm-item {
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 6px 8px;
		margin: 1px 8px;
		border-radius: 4px;
		background: none;
		border: none;
		cursor: pointer;
		width: calc(100% - 16px);
		position: relative;
		transition: background-color 0.15s ease;
	}

	.dm-item:hover {
		background: #35373c;
	}

	.dm-item.active {
		background: #404249;
	}

	.dm-avatar {
		width: 32px;
		height: 32px;
		border-radius: 50%;
		overflow: hidden;
		background: #5865f2;
		flex-shrink: 0;
	}

	.dm-avatar img {
		width: 100%;
		height: 100%;
		object-fit: cover;
	}

	.avatar-placeholder {
		width: 100%;
		height: 100%;
		display: flex;
		align-items: center;
		justify-content: center;
		color: white;
		font-weight: 500;
		font-size: 14px;
	}

	.dm-name {
		flex: 1;
		color: #949ba4;
		font-size: 16px;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
		text-align: left;
		transition: color 0.15s ease;
	}

	.dm-item.active .dm-name,
	.dm-item:hover .dm-name {
		color: #dbdee1;
	}

	.e2ee-indicator {
		font-size: 12px;
		opacity: 0.6;
	}

	.dm-item .mention-badge {
		display: flex;
		align-items: center;
		justify-content: center;
		min-width: 18px;
		height: 18px;
		padding: 0 4px;
		border-radius: 9px;
		background: #faa61a;
		color: #ffffff;
		font-size: 11px;
		font-weight: 600;
		line-height: 1;
		flex-shrink: 0;
	}

	.dm-item .unread-dot {
		width: 8px;
		height: 8px;
		border-radius: 50%;
		background: #f2f3f5;
		flex-shrink: 0;
	}

	.dm-item.unread .dm-name {
		color: #f2f3f5;
	}

	.no-channels-hint {
		padding: 8px 8px 8px 48px;
		font-size: 13px;
		color: var(--text-muted);
		font-style: italic;
	}
</style>
