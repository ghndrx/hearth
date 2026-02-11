<script lang="ts">
	import { activeServer, activeMembers } from '$stores';
	import type { Member } from '$lib/types';

	function getStatusColor(status: string): string {
		switch (status) {
			case 'online':
				return 'bg-green-500';
			case 'idle':
				return 'bg-yellow-500';
			case 'dnd':
				return 'bg-red-500';
			default:
				return 'bg-gray-500';
		}
	}

	// Group members by their primary role (simplified)
	$: onlineMembers = $activeMembers.filter((m) => m.user?.status !== 'offline');
	$: offlineMembers = $activeMembers.filter((m) => m.user?.status === 'offline');
</script>

{#if $activeServer}
	<aside class="w-60 bg-dark-800 flex flex-col shrink-0 hidden lg:flex">
		<div class="flex-1 overflow-y-auto px-2 py-4">
			<!-- Online members -->
			{#if onlineMembers.length > 0}
				<div class="mb-4">
					<h3 class="px-2 mb-2 text-xs font-semibold uppercase text-gray-400">
						Online — {onlineMembers.length}
					</h3>
					{#each onlineMembers as member (member.userId)}
						<button class="sidebar-item w-full text-left group">
							<div class="relative">
								<div class="w-8 h-8 rounded-full bg-hearth-500 flex items-center justify-center">
									{#if member.user?.avatarUrl}
										<img
											src={member.user.avatarUrl}
											alt={member.user.username}
											class="w-full h-full rounded-full object-cover"
										/>
									{:else}
										<span class="text-sm font-medium text-white">
											{member.user?.username?.charAt(0).toUpperCase() || 'U'}
										</span>
									{/if}
								</div>
								<div
									class="absolute -bottom-0.5 -right-0.5 w-3.5 h-3.5 rounded-full border-2 border-dark-800
									       {getStatusColor(member.user?.status || 'offline')}"
								></div>
							</div>
							<div class="flex-1 min-w-0">
								<span class="text-gray-300 group-hover:text-white truncate block">
									{member.nickname || member.user?.username || 'Unknown'}
								</span>
							</div>
						</button>
					{/each}
				</div>
			{/if}

			<!-- Offline members -->
			{#if offlineMembers.length > 0}
				<div>
					<h3 class="px-2 mb-2 text-xs font-semibold uppercase text-gray-400">
						Offline — {offlineMembers.length}
					</h3>
					{#each offlineMembers as member (member.userId)}
						<button class="sidebar-item w-full text-left group opacity-60">
							<div class="relative">
								<div class="w-8 h-8 rounded-full bg-dark-600 flex items-center justify-center">
									{#if member.user?.avatarUrl}
										<img
											src={member.user.avatarUrl}
											alt={member.user.username}
											class="w-full h-full rounded-full object-cover grayscale"
										/>
									{:else}
										<span class="text-sm font-medium text-gray-400">
											{member.user?.username?.charAt(0).toUpperCase() || 'U'}
										</span>
									{/if}
								</div>
							</div>
							<div class="flex-1 min-w-0">
								<span class="text-gray-500 group-hover:text-gray-400 truncate block">
									{member.nickname || member.user?.username || 'Unknown'}
								</span>
							</div>
						</button>
					{/each}
				</div>
			{/if}

			{#if $activeMembers.length === 0}
				<p class="text-center text-gray-500 text-sm py-4">
					No members to display
				</p>
			{/if}
		</div>
	</aside>
{/if}
