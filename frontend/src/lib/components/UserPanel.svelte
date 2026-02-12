<script lang="ts">
	import { user, auth } from '$lib/stores/auth';
	import { gatewayState } from '$lib/gateway';
	import { settings } from '$lib/stores/settings';
	
	let isMuted = false;
	let isDeafened = false;
	
	function toggleMute() {
		isMuted = !isMuted;
	}
	
	function toggleDeafen() {
		isDeafened = !isDeafened;
		if (isDeafened) isMuted = true;
	}
	
	function openSettings() {
		settings.open('account');
	}
	
	function getConnectionStatus(state: string) {
		switch (state) {
			case 'connected': return { color: 'var(--status-online)', text: 'Connected' };
			case 'connecting': return { color: 'var(--status-idle)', text: 'Connecting...' };
			case 'reconnecting': return { color: 'var(--status-idle)', text: 'Reconnecting...' };
			default: return { color: 'var(--status-offline)', text: 'Disconnected' };
		}
	}
	
	$: connectionStatus = getConnectionStatus($gatewayState);
</script>

<div class="user-panel">
	<div class="user-info">
		<div class="avatar">
			{#if $user?.avatar}
				<img src={$user.avatar} alt="" />
			{:else}
				<div class="avatar-placeholder">
					{($user?.username || '?')[0].toUpperCase()}
				</div>
			{/if}
			<div class="status-indicator online"></div>
		</div>
		
		<div class="user-details">
			<span class="username">{$user?.display_name || $user?.username}</span>
			<span class="tag">
				<span 
					class="connection-dot"
					style="background: {connectionStatus.color}"
				></span>
				{connectionStatus.text}
			</span>
		</div>
	</div>
	
	<div class="controls">
		<button 
			class="control-btn"
			class:active={isMuted}
			on:click={toggleMute}
			title={isMuted ? 'Unmute' : 'Mute'}
		>
			{#if isMuted}
				<svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor">
					<path d="M12 14c1.66 0 3-1.34 3-3V5c0-1.66-1.34-3-3-3S9 3.34 9 5v6c0 1.66 1.34 3 3 3zm-1-9c0-.55.45-1 1-1s1 .45 1 1v6c0 .55-.45 1-1 1s-1-.45-1-1V5z"/>
					<path d="M17 11c0 2.76-2.24 5-5 5s-5-2.24-5-5H5c0 3.53 2.61 6.43 6 6.92V21h2v-3.08c3.39-.49 6-3.39 6-6.92h-2z"/>
					<line x1="3" y1="3" x2="21" y2="21" stroke="currentColor" stroke-width="2"/>
				</svg>
			{:else}
				<svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor">
					<path d="M12 14c1.66 0 3-1.34 3-3V5c0-1.66-1.34-3-3-3S9 3.34 9 5v6c0 1.66 1.34 3 3 3zm-1-9c0-.55.45-1 1-1s1 .45 1 1v6c0 .55-.45 1-1 1s-1-.45-1-1V5zm6 6c0 2.76-2.24 5-5 5s-5-2.24-5-5H5c0 3.53 2.61 6.43 6 6.92V21h2v-3.08c3.39-.49 6-3.39 6-6.92h-2z"/>
				</svg>
			{/if}
		</button>
		
		<button 
			class="control-btn"
			class:active={isDeafened}
			on:click={toggleDeafen}
			title={isDeafened ? 'Undeafen' : 'Deafen'}
		>
			{#if isDeafened}
				<svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor">
					<path d="M12 2C6.48 2 2 6.48 2 12v4c0 1.1.9 2 2 2h1v-6H4v-2c0-4.42 3.58-8 8-8s8 3.58 8 8v2h-1v6h1c1.1 0 2-.9 2-2v-4c0-5.52-4.48-10-10-10z"/>
					<line x1="3" y1="3" x2="21" y2="21" stroke="currentColor" stroke-width="2"/>
				</svg>
			{:else}
				<svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor">
					<path d="M12 2C6.48 2 2 6.48 2 12v4c0 1.1.9 2 2 2h1v-6H4v-2c0-4.42 3.58-8 8-8s8 3.58 8 8v2h-1v6h1c1.1 0 2-.9 2-2v-4c0-5.52-4.48-10-10-10z"/>
					<rect x="6" y="12" width="4" height="8" rx="1"/>
					<rect x="14" y="12" width="4" height="8" rx="1"/>
				</svg>
			{/if}
		</button>
		
		<button class="control-btn" on:click={openSettings} title="User Settings">
			<svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor">
				<path d="M19.14 12.94c.04-.31.06-.63.06-.94 0-.31-.02-.63-.06-.94l2.03-1.58c.18-.14.23-.41.12-.61l-1.92-3.32c-.12-.22-.37-.29-.59-.22l-2.39.96c-.5-.38-1.03-.7-1.62-.94l-.36-2.54c-.04-.24-.24-.41-.48-.41h-3.84c-.24 0-.43.17-.47.41l-.36 2.54c-.59.24-1.13.57-1.62.94l-2.39-.96c-.22-.08-.47 0-.59.22L2.74 8.87c-.12.21-.08.47.12.61l2.03 1.58c-.04.31-.06.63-.06.94s.02.63.06.94l-2.03 1.58c-.18.14-.23.41-.12.61l1.92 3.32c.12.22.37.29.59.22l2.39-.96c.5.38 1.03.7 1.62.94l.36 2.54c.05.24.24.41.48.41h3.84c.24 0 .44-.17.47-.41l.36-2.54c.59-.24 1.13-.56 1.62-.94l2.39.96c.22.08.47 0 .59-.22l1.92-3.32c.12-.22.07-.47-.12-.61l-2.01-1.58zM12 15.6c-1.98 0-3.6-1.62-3.6-3.6s1.62-3.6 3.6-3.6 3.6 1.62 3.6 3.6-1.62 3.6-3.6 3.6z"/>
			</svg>
		</button>
	</div>
</div>

<style>
	.user-panel {
		display: flex;
		align-items: center;
		padding: 0 8px;
		height: 52px;
		background: var(--bg-tertiary);
	}
	
	.user-info {
		display: flex;
		align-items: center;
		gap: 8px;
		flex: 1;
		min-width: 0;
		padding: 4px;
		border-radius: 4px;
		cursor: pointer;
	}
	
	.user-info:hover {
		background: var(--bg-modifier-hover);
	}
	
	.avatar {
		position: relative;
		width: 32px;
		height: 32px;
	}
	
	.avatar img {
		width: 100%;
		height: 100%;
		border-radius: 50%;
		object-fit: cover;
	}
	
	.avatar-placeholder {
		width: 100%;
		height: 100%;
		border-radius: 50%;
		background: var(--brand-primary);
		display: flex;
		align-items: center;
		justify-content: center;
		color: white;
		font-weight: 600;
		font-size: 14px;
	}
	
	.status-indicator {
		position: absolute;
		bottom: -2px;
		right: -2px;
		width: 12px;
		height: 12px;
		border-radius: 50%;
		border: 3px solid var(--bg-tertiary);
	}
	
	.status-indicator.online {
		background: var(--status-online);
	}
	
	.user-details {
		flex: 1;
		min-width: 0;
	}
	
	.username {
		display: block;
		font-size: 14px;
		font-weight: 600;
		color: var(--text-primary);
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	
	.tag {
		display: flex;
		align-items: center;
		gap: 4px;
		font-size: 12px;
		color: var(--text-muted);
	}
	
	.connection-dot {
		width: 8px;
		height: 8px;
		border-radius: 50%;
	}
	
	.controls {
		display: flex;
		gap: 4px;
	}
	
	.control-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 32px;
		height: 32px;
		background: none;
		border: none;
		border-radius: 4px;
		color: var(--text-muted);
		cursor: pointer;
	}
	
	.control-btn:hover {
		background: var(--bg-modifier-hover);
		color: var(--text-primary);
	}
	
	.control-btn.active {
		color: var(--status-danger);
	}
</style>
