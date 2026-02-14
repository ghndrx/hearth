<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { isAuthenticated } from '$lib/stores/auth';
	import { loadServers } from '$lib/stores/servers';
	import { loadDMChannels } from '$lib/stores/channels';
	import { isSettingsOpen, isServerSettingsOpen, settings } from '$lib/stores/settings';
	import { currentServer } from '$lib/stores/servers';
	import ServerList from '$lib/components/ServerList.svelte';
	import ChannelList from '$lib/components/ChannelList.svelte';
	import MemberList from '$lib/components/MemberList.svelte';
	import UserSettings from '$lib/components/UserSettings.svelte';
	import ServerSettings from '$lib/components/ServerSettings.svelte';

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
</div>

<UserSettings open={$isSettingsOpen} on:close={() => settings.close()} />

<ServerSettings open={$isServerSettingsOpen} on:close={() => settings.closeServerSettings()} />
