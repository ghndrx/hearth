<script lang="ts">
	import { channels, currentChannel, type Channel, type ChannelTypeString } from '$lib/stores/channels';
	import { currentServer, leaveServer } from '$lib/stores/servers';
	import { user } from '$lib/stores/auth';
	import { settings } from '$lib/stores/settings';
	import { createEventDispatcher } from 'svelte';
	import { goto } from '$app/navigation';
	import UserPanel from './UserPanel.svelte';
	import ServerHeader from './ServerHeader.svelte';
	import ChannelCategory from './ChannelCategory.svelte';
	import ChannelItem from './ChannelItem.svelte';
	import InviteModal from './InviteModal.svelte';
	import CreateChannelModal from './CreateChannelModal.svelte';

	const dispatch = createEventDispatcher();

	let textCategoryCollapsed = false;
	let voiceCategoryCollapsed = false;
	let showInviteModal = false;
	let showCreateChannelModal = false;
	let createChannelType: ChannelTypeString = 'text';

	// Mock connected users for voice channels (would come from voice state in real app)
	let voiceConnectedUsers: Record<string, Array<{
		id: string;
		username: string;
		avatar: string | null;
		speaking?: boolean;
	}>> = {};

	$: serverChannels = $channels.filter((c) => c.server_id === $currentServer?.id);
	$: textChannels = serverChannels.filter((c) => c.type === 0);
	$: voiceChannels = serverChannels.filter((c) => c.type === 2);
	$: isOwner = $currentServer?.owner_id === $user?.id;

	function selectChannel(channel: Channel) {
		currentChannel.set(channel);
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
		// In production, this would send the invite to the API
		console.log('Invite created:', event.detail);
	}

	function handleAddTextChannel() {
		createChannelType = 'text';
		showCreateChannelModal = true;
	}

	function handleAddVoiceChannel() {
		createChannelType = 'voice';
		showCreateChannelModal = true;
	}

	function handleCreateChannelClose() {
		showCreateChannelModal = false;
	}

	function handleChannelCreated(event: CustomEvent<Channel>) {
		const channel = event.detail;
		// Navigate to the newly created channel
		if (channel.server_id) {
			goto(`/channels/${channel.server_id}/${channel.id}`);
		}
	}

	function handleChannelSettings(event: CustomEvent<Channel>) {
		// TODO: Open channel settings
		console.log('Open settings for channel:', event.detail.name);
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

			<!-- Text Channels -->
			{#if textChannels.length > 0}
				<ChannelCategory
					name="Text Channels"
					collapsed={textCategoryCollapsed}
					on:toggle={(e) => textCategoryCollapsed = e.detail.collapsed}
					on:addChannel={handleAddTextChannel}
				>
					{#each textChannels as channel (channel.id)}
						<ChannelItem
							{channel}
							active={$currentChannel?.id === channel.id}
							on:select={(e) => selectChannel(e.detail)}
							on:openSettings={handleChannelSettings}
						/>
					{/each}
				</ChannelCategory>
			{/if}

			<!-- Voice Channels -->
			{#if voiceChannels.length > 0}
				<ChannelCategory
					name="Voice Channels"
					collapsed={voiceCategoryCollapsed}
					on:toggle={(e) => voiceCategoryCollapsed = e.detail.collapsed}
					on:addChannel={handleAddVoiceChannel}
				>
					{#each voiceChannels as channel (channel.id)}
						<ChannelItem
							{channel}
							active={$currentChannel?.id === channel.id}
							connectedUsers={voiceConnectedUsers[channel.id] || []}
							on:select={(e) => selectChannel(e.detail)}
							on:openSettings={handleChannelSettings}
						/>
					{/each}
				</ChannelCategory>
			{/if}
		{:else}
			<!-- DM List -->
			<div class="dm-header">
				<button class="dm-search" aria-label="Find or start a conversation" type="button">Find or start a conversation</button>
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
							on:click={() => selectChannel(dm)}
							aria-current={$currentChannel?.id === dm.id ? 'page' : undefined}
							aria-label="Direct message with {dm.name || dm.recipients?.map((r) => r.display_name || r.username).join(', ') || 'Unknown'}{dm.e2ee_enabled ? ', encrypted' : ''}"
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
						</button>
					</li>
				{/each}
			</ul>
		{/if}
	</div>

	<UserPanel />
</nav>

<!-- Invite Modal -->
{#if $currentServer}
	<InviteModal
		open={showInviteModal}
		serverName={$currentServer.name}
		serverId={$currentServer.id}
		channelName={$currentChannel?.name ?? ''}
		channelId={$currentChannel?.id ?? ''}
		on:close={handleInviteClose}
		on:invite={handleInviteCreated}
	/>

	<!-- Create Channel Modal -->
	<CreateChannelModal
		open={showCreateChannelModal}
		defaultType={createChannelType}
		on:close={handleCreateChannelClose}
		on:created={handleChannelCreated}
	/>
{/if}

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
	}

	.dm-item.active .dm-name,
	.dm-item:hover .dm-name {
		color: #dbdee1;
	}

	.e2ee-indicator {
		font-size: 12px;
		opacity: 0.6;
	}
</style>
