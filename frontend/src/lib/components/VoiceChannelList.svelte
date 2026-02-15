<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import type { Channel } from '$lib/stores/channels';
	import {
		voiceState,
		channelVoiceStates,
		joinVoiceChannel,
		leaveVoiceChannel,
		toggleMute,
		toggleDeafen,
		type VoiceUser
	} from '$lib/stores/voice';
	import Avatar from './Avatar.svelte';
	import Tooltip from './Tooltip.svelte';

	export let channels: Channel[] = [];
	export let serverId: string;

	const dispatch = createEventDispatcher<{
		channelSelect: Channel;
		userClick: { user: VoiceUser; channel: Channel };
	}>();

	// Filter to only voice channels (type 2)
	$: voiceChannels = channels.filter(c => c.type === 2);

	function handleChannelClick(channel: Channel) {
		const isConnected = $voiceState.channelId === channel.id;
		
		if (isConnected) {
			// If already connected to this channel, dispatch select event
			dispatch('channelSelect', channel);
		} else {
			// Join the voice channel
			joinVoiceChannel(channel.id, serverId);
		}
	}

	function handleDisconnect(e: MouseEvent) {
		e.stopPropagation();
		leaveVoiceChannel();
	}

	function handleUserClick(e: MouseEvent, user: VoiceUser, channel: Channel) {
		e.stopPropagation();
		dispatch('userClick', { user, channel });
	}

	function getChannelUsers(channelId: string): VoiceUser[] {
		return $channelVoiceStates[channelId] || [];
	}
</script>

<div class="voice-channel-list">
	{#each voiceChannels as channel (channel.id)}
		{@const users = getChannelUsers(channel.id)}
		{@const isConnected = $voiceState.channelId === channel.id}
		
		<div class="voice-channel-container">
			<!-- Channel Header -->
			<button
				class="voice-channel"
				class:active={isConnected}
				on:click={() => handleChannelClick(channel)}
				aria-label="Voice channel {channel.name}, {users.length} users connected"
			>
				<div class="channel-info">
					<!-- Voice icon -->
					<div class="channel-icon">
						<svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor" aria-hidden="true">
							<path d="M11.383 3.07904C11.009 2.92504 10.579 3.01004 10.293 3.29604L6 8.00204H3C2.45 8.00204 2 8.45304 2 9.00204V15.002C2 15.552 2.45 16.002 3 16.002H6L10.293 20.71C10.579 20.996 11.009 21.082 11.383 20.927C11.757 20.772 12 20.407 12 20.002V4.00204C12 3.59904 11.757 3.23204 11.383 3.07904Z"/>
							<path d="M14 9C14 9 16 10.5 16 12C16 13.5 14 15 14 15" stroke="currentColor" stroke-width="2" stroke-linecap="round" fill="none"/>
							<path d="M17.7 6.3C17.7 6.3 21 9.5 21 12C21 14.5 17.7 17.7 17.7 17.7" stroke="currentColor" stroke-width="2" stroke-linecap="round" fill="none"/>
						</svg>
					</div>
					
					<span class="channel-name">{channel.name}</span>
				</div>

				<!-- User count badge -->
				{#if users.length > 0}
					<span class="user-count">{users.length}</span>
				{/if}
			</button>

			<!-- Connected Users -->
			{#if users.length > 0}
				<div class="connected-users" role="list" aria-label="Connected users">
					{#each users as voiceUser (voiceUser.id)}
						<button
							class="voice-user"
							class:speaking={voiceUser.speaking}
							class:muted={voiceUser.muted}
							class:deafened={voiceUser.deafened}
							on:click={(e) => handleUserClick(e, voiceUser, channel)}
							role="listitem"
							aria-label="{voiceUser.display_name || voiceUser.username}{voiceUser.speaking ? ', speaking' : ''}{voiceUser.muted ? ', muted' : ''}{voiceUser.deafened ? ', deafened' : ''}"
						>
							<div class="user-avatar" class:speaking-ring={voiceUser.speaking}>
								<Avatar
									src={voiceUser.avatar}
									username={voiceUser.username}
									size="xs"
									alt=""
								/>
							</div>

							<span class="user-name">
								{voiceUser.display_name || voiceUser.username}
							</span>

							<!-- Status Icons -->
							<div class="status-icons">
								{#if voiceUser.streaming}
									<Tooltip text="Streaming">
										<div class="status-icon streaming">
											<svg viewBox="0 0 24 24" width="14" height="14" fill="currentColor">
												<path d="M2 4.5C2 3.67 2.67 3 3.5 3H20.5C21.33 3 22 3.67 22 4.5V15.5C22 16.33 21.33 17 20.5 17H3.5C2.67 17 2 16.33 2 15.5V4.5ZM4 15H20V5H4V15ZM7 21C6.45 21 6 20.55 6 20C6 19.45 6.45 19 7 19H17C17.55 19 18 19.45 18 20C18 20.55 17.55 21 17 21H7Z"/>
											</svg>
										</div>
									</Tooltip>
								{/if}

								{#if voiceUser.video}
									<Tooltip text="Video">
										<div class="status-icon video">
											<svg viewBox="0 0 24 24" width="14" height="14" fill="currentColor">
												<path d="M21.526 8.149C21.231 7.966 20.862 7.951 20.553 8.105L17 9.882V8C17 6.897 16.103 6 15 6H5C3.897 6 3 6.897 3 8V16C3 17.103 3.897 18 5 18H15C16.103 18 17 17.103 17 16V14.118L20.553 15.895C20.694 15.965 20.847 16 21 16C21.183 16 21.365 15.949 21.526 15.851C21.82 15.668 22 15.347 22 15V9C22 8.653 21.82 8.332 21.526 8.149Z"/>
											</svg>
										</div>
									</Tooltip>
								{/if}

								{#if voiceUser.deafened}
									<Tooltip text="Deafened">
										<div class="status-icon deafened">
											<svg viewBox="0 0 24 24" width="14" height="14" fill="currentColor">
												<path d="M6.16204 15.0065C6.10859 15.0022 6.05455 15 6 15H4V12C4 7.588 7.589 4 12 4C13.4809 4 14.8691 4.40439 16.0599 5.10859L17.5102 3.65835C15.9292 2.61064 14.0346 2 12 2C6.486 2 2 6.485 2 12V19.1685L6.16204 15.0065Z"/>
												<path d="M19.725 9.91686C19.9043 10.5813 20 11.2796 20 12V15H18C16.896 15 16 15.896 16 17V20C16 21.104 16.896 22 18 22H20C21.105 22 22 21.104 22 20V12C22 10.7075 21.7536 9.47149 21.3053 8.33658L19.725 9.91686Z"/>
												<path d="M3.20101 23.6243L1.7868 22.2101L21.5858 2.41113L23 3.82535L3.20101 23.6243Z"/>
											</svg>
										</div>
									</Tooltip>
								{:else if voiceUser.muted}
									<Tooltip text="Muted">
										<div class="status-icon muted">
											<svg viewBox="0 0 24 24" width="14" height="14" fill="currentColor">
												<path d="M6.7 11H5C5 12.19 5.34 13.3 5.9 14.28L7.13 13.05C6.86 12.43 6.7 11.74 6.7 11ZM9.01 11.085C9.015 11.1125 9.02 11.14 9.02 11.17L15 5.18V5C15 3.34 13.66 2 12 2C10.34 2 9 3.34 9 5V11C9 11.03 9.005 11.0575 9.01 11.085ZM11.7 16.61C11.8 16.63 11.9 16.64 12 16.64C14.76 16.64 17 14.4 17 11.64H15.3C15.3 13.44 13.8 14.94 12 14.94C11.9 14.94 11.8 14.93 11.7 14.92L11.7 16.61ZM21 2.1L3.1 20L4.5 21.4L8.63 17.27C9.68 18.02 10.95 18.51 12.3 18.63V21H13.7V18.63C17.14 18.26 19.78 15.35 19.78 11.77H18.08C18.08 14.53 15.87 16.8 13.15 16.9C12.75 16.94 12.37 16.89 12 16.82L14.1 14.72C14.38 14.49 14.61 14.22 14.79 13.91L21 7.7V21H22.4V0.7L21 2.1Z"/>
											</svg>
										</div>
									</Tooltip>
								{/if}
							</div>
						</button>
					{/each}
				</div>
			{/if}
		</div>
	{/each}

	<!-- Voice Connected Panel (shows when connected) -->
	{#if $voiceState.channelId}
		{@const currentChannel = voiceChannels.find(c => c.id === $voiceState.channelId)}
		<div class="voice-connected-panel">
			<div class="connected-info">
				<div class="connected-status">
					<span class="status-indicator"></span>
					<span class="status-text">Voice Connected</span>
				</div>
				{#if currentChannel}
					<span class="connected-channel">{currentChannel.name}</span>
				{/if}
			</div>

			<div class="voice-controls">
				<Tooltip text={$voiceState.muted ? 'Unmute' : 'Mute'}>
					<button
						class="control-btn"
						class:active={$voiceState.muted}
						on:click={toggleMute}
						aria-label={$voiceState.muted ? 'Unmute' : 'Mute'}
					>
						{#if $voiceState.muted}
							<svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor">
								<path d="M6.7 11H5C5 12.19 5.34 13.3 5.9 14.28L7.13 13.05C6.86 12.43 6.7 11.74 6.7 11ZM9.01 11.085C9.015 11.1125 9.02 11.14 9.02 11.17L15 5.18V5C15 3.34 13.66 2 12 2C10.34 2 9 3.34 9 5V11C9 11.03 9.005 11.0575 9.01 11.085ZM11.7 16.61C11.8 16.63 11.9 16.64 12 16.64C14.76 16.64 17 14.4 17 11.64H15.3C15.3 13.44 13.8 14.94 12 14.94C11.9 14.94 11.8 14.93 11.7 14.92L11.7 16.61ZM21 2.1L3.1 20L4.5 21.4L8.63 17.27C9.68 18.02 10.95 18.51 12.3 18.63V21H13.7V18.63C17.14 18.26 19.78 15.35 19.78 11.77H18.08C18.08 14.53 15.87 16.8 13.15 16.9C12.75 16.94 12.37 16.89 12 16.82L14.1 14.72C14.38 14.49 14.61 14.22 14.79 13.91L21 7.7V21H22.4V0.7L21 2.1Z"/>
							</svg>
						{:else}
							<svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor">
								<path d="M12 14C13.66 14 14.99 12.66 14.99 11L15 5C15 3.34 13.66 2 12 2C10.34 2 9 3.34 9 5V11C9 12.66 10.34 14 12 14ZM17.3 11C17.3 14 14.76 16.1 12 16.1C9.24 16.1 6.7 14 6.7 11H5C5 14.41 7.72 17.23 11 17.72V21H13V17.72C16.28 17.23 19 14.41 19 11H17.3Z"/>
							</svg>
						{/if}
					</button>
				</Tooltip>

				<Tooltip text={$voiceState.deafened ? 'Undeafen' : 'Deafen'}>
					<button
						class="control-btn"
						class:active={$voiceState.deafened}
						on:click={toggleDeafen}
						aria-label={$voiceState.deafened ? 'Undeafen' : 'Deafen'}
					>
						{#if $voiceState.deafened}
							<svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor">
								<path d="M6.16204 15.0065C6.10859 15.0022 6.05455 15 6 15H4V12C4 7.588 7.589 4 12 4C13.4809 4 14.8691 4.40439 16.0599 5.10859L17.5102 3.65835C15.9292 2.61064 14.0346 2 12 2C6.486 2 2 6.485 2 12V19.1685L6.16204 15.0065Z"/>
								<path d="M19.725 9.91686C19.9043 10.5813 20 11.2796 20 12V15H18C16.896 15 16 15.896 16 17V20C16 21.104 16.896 22 18 22H20C21.105 22 22 21.104 22 20V12C22 10.7075 21.7536 9.47149 21.3053 8.33658L19.725 9.91686Z"/>
								<path d="M3.20101 23.6243L1.7868 22.2101L21.5858 2.41113L23 3.82535L3.20101 23.6243Z"/>
							</svg>
						{:else}
							<svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor">
								<path d="M12 2.00305C6.486 2.00305 2 6.48805 2 12.0031V20.0031C2 21.1071 2.895 22.0031 4 22.0031H6C7.104 22.0031 8 21.1071 8 20.0031V17.0031C8 15.8991 7.104 15.0031 6 15.0031H4V12.0031C4 7.59105 7.589 4.00305 12 4.00305C16.411 4.00305 20 7.59105 20 12.0031V15.0031H18C16.896 15.0031 16 15.8991 16 17.0031V20.0031C16 21.1071 16.896 22.0031 18 22.0031H20C21.104 22.0031 22 21.1071 22 20.0031V12.0031C22 6.48805 17.514 2.00305 12 2.00305Z"/>
							</svg>
						{/if}
					</button>
				</Tooltip>

				<Tooltip text="Disconnect">
					<button
						class="control-btn disconnect"
						on:click={handleDisconnect}
						aria-label="Disconnect from voice"
					>
						<svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor">
							<path d="M21.1169 1.11603L22.8839 2.88403L19.7679 6.00003L22.8839 9.11603L21.1169 10.884L17.9999 7.76803L14.8839 10.884L13.1169 9.11603L16.2329 6.00003L13.1169 2.88403L14.8839 1.11603L17.9999 4.23203L21.1169 1.11603ZM18 22H13C6.925 22 2 17.075 2 11V6C2 5.447 2.448 5 3 5H7C7.553 5 8 5.447 8 6V10C8 10.553 7.553 11 7 11H6C6.063 14.938 9 18 13 18V17C13 16.447 13.447 16 14 16H18C18.553 16 19 16.447 19 17V21C19 21.553 18.553 22 18 22Z"/>
						</svg>
					</button>
				</Tooltip>
			</div>
		</div>
	{/if}
</div>

<style>
	.voice-channel-list {
		display: flex;
		flex-direction: column;
		gap: 2px;
	}

	.voice-channel-container {
		display: flex;
		flex-direction: column;
	}

	.voice-channel {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 6px 8px;
		margin: 1px 8px;
		border-radius: 4px;
		background: none;
		border: none;
		color: var(--text-muted, #949ba4);
		font-size: 16px;
		cursor: pointer;
		text-align: left;
		width: calc(100% - 16px);
		transition: background-color 0.1s ease, color 0.1s ease;
	}

	.voice-channel:hover {
		background: var(--bg-modifier-hover, #35373c);
		color: var(--text-normal, #dbdee1);
	}

	.voice-channel.active {
		background: var(--bg-modifier-selected, #404249);
		color: var(--text-normal, #ffffff);
	}

	.channel-info {
		display: flex;
		align-items: center;
		gap: 6px;
		flex: 1;
		min-width: 0;
	}

	.channel-icon {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 20px;
		height: 20px;
		flex-shrink: 0;
		color: inherit;
	}

	.channel-icon svg {
		color: inherit;
	}

	.channel-name {
		flex: 1;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
		font-weight: 500;
	}

	.user-count {
		font-size: 12px;
		color: var(--text-muted, #949ba4);
		background: var(--bg-tertiary, #1e1f22);
		padding: 2px 6px;
		border-radius: 8px;
		flex-shrink: 0;
	}

	/* Connected Users */
	.connected-users {
		display: flex;
		flex-direction: column;
		margin-left: 32px;
		padding: 2px 8px;
	}

	.voice-user {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 4px 8px;
		border-radius: 4px;
		background: none;
		border: none;
		cursor: pointer;
		width: 100%;
		text-align: left;
		transition: background-color 0.1s ease;
	}

	.voice-user:hover {
		background: rgba(79, 84, 92, 0.16);
	}

	.user-avatar {
		position: relative;
		border-radius: 50%;
	}

	.user-avatar.speaking-ring {
		box-shadow: 0 0 0 2px var(--status-online, #23a559);
	}

	.user-name {
		font-size: 13px;
		color: var(--text-muted, #949ba4);
		flex: 1;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.voice-user:hover .user-name {
		color: var(--text-normal, #dbdee1);
	}

	.voice-user.speaking .user-name {
		color: var(--status-online, #23a559);
	}

	.status-icons {
		display: flex;
		align-items: center;
		gap: 4px;
	}

	.status-icon {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 14px;
		height: 14px;
		color: var(--text-muted, #949ba4);
	}

	.status-icon.muted,
	.status-icon.deafened {
		color: var(--status-dnd, #f23f43);
	}

	.status-icon.streaming {
		color: var(--fuchsia, #eb459e);
	}

	.status-icon.video {
		color: var(--status-online, #23a559);
	}

	/* Voice Connected Panel */
	.voice-connected-panel {
		display: flex;
		flex-direction: column;
		gap: 8px;
		padding: 8px;
		margin: 8px;
		background: var(--bg-secondary-alt, #232428);
		border-radius: 4px;
		border: 1px solid var(--bg-modifier-accent, #3f4147);
	}

	.connected-info {
		display: flex;
		flex-direction: column;
		gap: 2px;
	}

	.connected-status {
		display: flex;
		align-items: center;
		gap: 8px;
	}

	.status-indicator {
		width: 8px;
		height: 8px;
		border-radius: 50%;
		background: var(--status-online, #23a559);
		animation: pulse 2s ease-in-out infinite;
	}

	@keyframes pulse {
		0%, 100% {
			opacity: 1;
		}
		50% {
			opacity: 0.5;
		}
	}

	.status-text {
		font-size: 12px;
		font-weight: 600;
		color: var(--status-online, #23a559);
		text-transform: uppercase;
	}

	.connected-channel {
		font-size: 13px;
		color: var(--text-normal, #dbdee1);
		margin-left: 16px;
	}

	.voice-controls {
		display: flex;
		gap: 8px;
		justify-content: center;
	}

	.control-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 32px;
		height: 32px;
		border-radius: 4px;
		background: var(--bg-tertiary, #1e1f22);
		border: none;
		color: var(--text-muted, #b5bac1);
		cursor: pointer;
		transition: background-color 0.15s, color 0.15s;
	}

	.control-btn:hover {
		background: var(--bg-modifier-hover, #35373c);
		color: var(--text-normal, #f2f3f5);
	}

	.control-btn.active {
		background: var(--status-dnd, #f23f43);
		color: white;
	}

	.control-btn.active:hover {
		background: #d83c3e;
	}

	.control-btn.disconnect {
		background: var(--status-dnd, #f23f43);
		color: white;
	}

	.control-btn.disconnect:hover {
		background: #d83c3e;
	}

	/* Mobile Responsive */
	@media (max-width: 640px) {
		.voice-channel {
			padding: 8px;
		}

		.channel-name {
			font-size: 15px;
		}

		.voice-connected-panel {
			margin: 4px;
			padding: 6px;
		}
	}
</style>
