<script lang="ts">
	import { servers, activeServer, activeChannels, activeChannel } from '$stores';
	import type { Channel } from '$lib/types';

	function getChannelIcon(type: Channel['type']): string {
		switch (type) {
			case 'voice':
				return '🔊';
			case 'announcement':
				return '📢';
			default:
				return '#';
		}
	}

	function selectChannel(channel: Channel) {
		servers.setActiveChannel(channel.id);
	}
</script>

<aside class="w-60 bg-dark-800 flex flex-col shrink-0">
	<!-- Server header -->
	<div class="h-12 px-4 flex items-center border-b border-dark-900 shadow-sm">
		{#if $activeServer}
			<button class="flex-1 flex items-center justify-between hover:bg-dark-700 -mx-2 px-2 py-1 rounded">
				<h2 class="font-semibold text-white truncate">{$activeServer.name}</h2>
				<svg class="w-4 h-4 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
				</svg>
			</button>
		{:else}
			<h2 class="font-semibold text-white">Direct Messages</h2>
		{/if}
	</div>

	<!-- Channel list -->
	<nav class="flex-1 overflow-y-auto py-4">
		{#if $activeServer}
			<!-- Text channels section -->
			<div class="mb-4">
				<button
					class="w-full flex items-center gap-1 px-4 py-1 text-xs font-semibold uppercase
					       text-gray-400 hover:text-gray-300"
				>
					<svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
					</svg>
					Text Channels
				</button>

				<div class="mt-1">
					{#each $activeChannels.filter((c) => c.type === 'text') as channel (channel.id)}
						<button
							class="channel-item w-full text-left"
							class:channel-item-active={$activeChannel?.id === channel.id}
							on:click={() => selectChannel(channel)}
						>
							<span class="text-lg opacity-70">{getChannelIcon(channel.type)}</span>
							<span class="truncate">{channel.name}</span>
						</button>
					{/each}
				</div>
			</div>

			<!-- Voice channels section -->
			{#if $activeChannels.some((c) => c.type === 'voice')}
				<div class="mb-4">
					<button
						class="w-full flex items-center gap-1 px-4 py-1 text-xs font-semibold uppercase
						       text-gray-400 hover:text-gray-300"
					>
						<svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
						</svg>
						Voice Channels
					</button>

					<div class="mt-1">
						{#each $activeChannels.filter((c) => c.type === 'voice') as channel (channel.id)}
							<button
								class="channel-item w-full text-left"
								class:channel-item-active={$activeChannel?.id === channel.id}
								on:click={() => selectChannel(channel)}
							>
								<span class="text-lg opacity-70">{getChannelIcon(channel.type)}</span>
								<span class="truncate">{channel.name}</span>
							</button>
						{/each}
					</div>
				</div>
			{/if}
		{:else}
			<!-- DM list placeholder -->
			<div class="px-4 py-2">
				<input
					type="text"
					placeholder="Find or start a conversation"
					class="input text-sm py-1.5"
				/>
			</div>
			<p class="px-4 py-2 text-sm text-gray-500">
				Select a server or start a direct message.
			</p>
		{/if}
	</nav>

	<!-- User panel -->
	<div class="h-14 px-2 bg-dark-900 flex items-center gap-2">
		<div class="w-8 h-8 rounded-full bg-hearth-500 flex items-center justify-center">
			<span class="text-xs font-medium text-white">U</span>
		</div>
		<div class="flex-1 min-w-0">
			<div class="text-sm font-medium text-white truncate">Username</div>
			<div class="text-xs text-gray-400">Online</div>
		</div>
		<div class="flex gap-1">
			<button class="p-1.5 text-gray-400 hover:text-white hover:bg-dark-700 rounded">
				<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
						d="M19 11a7 7 0 01-7 7m0 0a7 7 0 01-7-7m7 7v4m0 0H8m4 0h4m-4-8a3 3 0 01-3-3V5a3 3 0 116 0v6a3 3 0 01-3 3z" />
				</svg>
			</button>
			<button class="p-1.5 text-gray-400 hover:text-white hover:bg-dark-700 rounded">
				<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
						d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
						d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
				</svg>
			</button>
		</div>
	</div>
</aside>
