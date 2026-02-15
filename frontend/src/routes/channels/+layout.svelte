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
	import ServerList from '$lib/components/ServerList.svelte';
	import ChannelList from '$lib/components/ChannelList.svelte';
	import MemberList from '$lib/components/MemberList.svelte';
	import ThreadView from '$lib/components/ThreadView.svelte';
	import UserSettings from '$lib/components/UserSettings.svelte';
	import ServerSettings from '$lib/components/ServerSettings.svelte';
	import UserPopout from '$lib/components/UserPopout.svelte';

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
</script>

<div class="flex h-screen overflow-hidden bg-[#313338]">
	<!-- Server List - Leftmost sidebar -->
	<ServerList />

	<!-- Channel List - Second sidebar -->
	<ChannelList />

	<!-- Main Content -->
	<main class="flex-1 flex flex-col min-w-0 bg-[#313338]">
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
