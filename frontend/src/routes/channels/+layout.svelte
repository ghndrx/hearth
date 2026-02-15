<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { isAuthenticated } from '$lib/stores/auth';
	import { loadServers } from '$lib/stores/servers';
	import { loadDMChannels } from '$lib/stores/channels';
	import { isSettingsOpen, isServerSettingsOpen, settings } from '$lib/stores/settings';
	import { currentServer } from '$lib/stores/servers';
	import { popoutStore } from '$lib/stores/popout';
	import { threadStore } from '$lib/stores/thread';
	import { pinnedMessagesStore, pinnedMessagesOpen } from '$lib/stores/pinnedMessages';
	import ServerList from '$lib/components/ServerList.svelte';
	import ChannelList from '$lib/components/ChannelList.svelte';
	import MemberList from '$lib/components/MemberList.svelte';
	import ThreadView from '$lib/components/ThreadView.svelte';
	import PinnedMessages from '$lib/components/PinnedMessages.svelte';
	import UserSettings from '$lib/components/UserSettings.svelte';
	import ServerSettings from '$lib/components/ServerSettings.svelte';
	import UserPopout from '$lib/components/UserPopout.svelte';
	import VoiceCallOverlay from '$lib/components/VoiceCallOverlay.svelte';

	onMount(() => {
		if (!$isAuthenticated) {
			goto('/login');
			return;
		}

		loadServers();
		loadDMChannels();
	});

	$: if ($isAuthenticated === false) {
		goto('/login');
	}

	// Handle popout events
	function handlePopoutClose() {
		popoutStore.close();
	}

	function handlePopoutMessage(event: CustomEvent<{ userId: string }>) {
		// TODO: Navigate to DM with user
		console.log('Message user:', event.detail.userId);
		popoutStore.close();
	}

	function handlePopoutCall(event: CustomEvent<{ userId: string; type: 'voice' | 'video' }>) {
		// TODO: Initiate call with user
		console.log('Call user:', event.detail.userId, event.detail.type);
		popoutStore.close();
	}

	function handlePopoutServerClick(event: CustomEvent<{ serverId: string }>) {
		// Navigate to server
		goto(`/channels/${event.detail.serverId}`);
		popoutStore.close();
	}

	function handleJumpToMessage(event: CustomEvent<{ channelId: string; messageId: string }>) {
		// Close the pinned messages panel
		pinnedMessagesStore.close();
		
		// TODO: Scroll to message in MessageList
		// For now, we just close the panel - scrolling to message would require
		// additional coordination with the MessageList component
		console.log('Jump to message:', event.detail.messageId);
	}
</script>

<div class="flex h-screen overflow-hidden bg-[#313338]">
	<!-- Skip link for keyboard users -->
	<a href="#main-content" class="skip-link">Skip to main content</a>
	
	<!-- Visually hidden app title for screen readers -->
	<h1 class="sr-only">Hearth Chat Application</h1>
	
	<!-- Server List - Leftmost sidebar -->
	<ServerList />

	<!-- Channel List - Second sidebar -->
	<ChannelList />

	<!-- Main Content -->
	<main id="main-content" class="flex-1 flex flex-col min-w-0 bg-[#313338]" aria-label="Main content area">
		<slot />
	</main>

	<!-- Member List - Right sidebar (only in servers) -->
	{#if $currentServer}
		<MemberList />
	{/if}

	<!-- Thread View - Right sidebar panel for viewing threads -->
	{#if $threadStore.currentThread}
		<ThreadView />
	{/if}

	<!-- Pinned Messages - Right sidebar panel for viewing pinned messages -->
	{#if $pinnedMessagesOpen}
		<PinnedMessages on:jumpToMessage={handleJumpToMessage} />
	{/if}
</div>

<!-- User Popout -->
{#if $popoutStore.isOpen && $popoutStore.user}
	<UserPopout
		user={$popoutStore.user}
		member={$popoutStore.member}
		position={$popoutStore.position}
		anchor={$popoutStore.anchor}
		mutualServers={$popoutStore.mutualServers}
		mutualFriends={$popoutStore.mutualFriends}
		on:close={handlePopoutClose}
		on:message={handlePopoutMessage}
		on:call={handlePopoutCall}
		on:serverClick={handlePopoutServerClick}
	/>
{/if}

<UserSettings open={$isSettingsOpen} on:close={() => settings.close()} />

<ServerSettings open={$isServerSettingsOpen} on:close={() => settings.closeServerSettings()} />

<!-- Voice Call Overlay - Floating mini-view during active calls -->
<VoiceCallOverlay />

<style>
	/* Skip link for keyboard navigation */
	.skip-link {
		position: absolute;
		top: -40px;
		left: 0;
		background: var(--brand-primary, #5865f2);
		color: white;
		padding: 8px 16px;
		z-index: 10000;
		text-decoration: none;
		font-weight: 600;
		border-radius: 0 0 4px 0;
		transition: top 0.2s;
	}

	.skip-link:focus {
		top: 0;
		outline: 2px solid white;
		outline-offset: 2px;
	}

	/* Screen reader only - visually hidden but accessible */
	.sr-only {
		position: absolute;
		width: 1px;
		height: 1px;
		padding: 0;
		margin: -1px;
		overflow: hidden;
		clip: rect(0, 0, 0, 0);
		white-space: nowrap;
		border: 0;
	}
</style>
