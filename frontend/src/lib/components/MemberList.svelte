<script lang="ts">
	import { currentServer } from '$lib/stores/servers';
	import { presenceStore, type PresenceStatus, type Activity, getActivityLabel } from '$lib/stores/presence';
	import { popoutStore } from '$lib/stores/popout';
	import { writable } from 'svelte/store';
	import { handleListKeyboard } from '$lib/utils/keyboard';
	import Avatar from './Avatar.svelte';

	let memberListElement: HTMLElement;

	interface Member {
		id: string;
		user: {
			id: string;
			username: string;
			display_name: string | null;
			avatar: string | null;
			banner?: string | null;
			bio?: string | null;
			pronouns?: string | null;
			bot?: boolean;
			created_at?: string;
		};
		nickname: string | null;
		roles: string[];
		joined_at?: string;
	}

	interface Role {
		id: string;
		name: string;
		color: string;
		position: number;
		hoist: boolean;
	}

	// TODO: Load from API
	const members = writable<Member[]>([]);
	const roles = writable<Role[]>([]);

	// Get presence status for a member
	function getMemberStatus(userId: string): PresenceStatus {
		return presenceStore.getPresence(userId).status;
	}

	// Get activity for a member
	function getMemberActivity(userId: string): Activity | null {
		const presence = presenceStore.getPresence(userId);
		return presence.activities?.[0] || null;
	}

	// Get the highest role color for a member
	function getMemberColor(member: Member, rolesList: Role[]): string {
		const memberRoles = rolesList
			.filter(r => member.roles.includes(r.id) && r.color && r.color !== '#000000')
			.sort((a, b) => b.position - a.position);

		return memberRoles[0]?.color || 'var(--text-normal)';
	}

	// Get roles with full info for a member
	function getMemberRoles(member: Member, rolesList: Role[]): { id: string; name: string; color: string }[] {
		return rolesList
			.filter(r => member.roles.includes(r.id) && r.name !== '@everyone')
			.sort((a, b) => b.position - a.position)
			.map(r => ({
				id: r.id,
				name: r.name,
				color: r.color || '#99aab5'
			}));
	}

	// Group members by role (hoisted roles) and status
	$: groupedMembers = groupMembersByRole($members, $roles);

	function groupMembersByRole(membersList: Member[], rolesList: Role[]) {
		const groups: {
			role: Role | null;
			members: Member[];
			label: string;
			isOffline?: boolean;
		}[] = [];

		const usedMembers = new Set<string>();

		// Get status for all members
		const memberStatuses = new Map<string, PresenceStatus>();
		for (const member of membersList) {
			memberStatuses.set(member.id, getMemberStatus(member.user.id));
		}

		// Only show hoisted roles (roles that should be displayed separately)
		const hoistedRoles = rolesList
			.filter(r => r.hoist && r.name !== '@everyone')
			.sort((a, b) => b.position - a.position);

		// Group members by their highest hoisted role
		for (const role of hoistedRoles) {
			const roleMembers = membersList.filter(m => {
				if (usedMembers.has(m.id)) return false;
				if (!m.roles.includes(role.id)) return false;
				const status = memberStatuses.get(m.id);
				return status && status !== 'offline' && status !== 'invisible';
			});

			if (roleMembers.length > 0) {
				// Sort members alphabetically within role
				roleMembers.sort((a, b) => {
					const nameA = a.nickname || a.user.display_name || a.user.username;
					const nameB = b.nickname || b.user.display_name || b.user.username;
					return nameA.localeCompare(nameB);
				});

				groups.push({
					role,
					members: roleMembers,
					label: role.name.toUpperCase()
				});
				roleMembers.forEach(m => usedMembers.add(m.id));
			}
		}

		// Online members without hoisted roles
		const onlineMembers = membersList.filter(m => {
			if (usedMembers.has(m.id)) return false;
			const status = memberStatuses.get(m.id);
			return status && status !== 'offline' && status !== 'invisible';
		});

		if (onlineMembers.length > 0) {
			onlineMembers.sort((a, b) => {
				const nameA = a.nickname || a.user.display_name || a.user.username;
				const nameB = b.nickname || b.user.display_name || b.user.username;
				return nameA.localeCompare(nameB);
			});

			groups.push({
				role: null,
				members: onlineMembers,
				label: 'ONLINE'
			});
			onlineMembers.forEach(m => usedMembers.add(m.id));
		}

		// Offline members
		const offlineMembers = membersList.filter(m => {
			if (usedMembers.has(m.id)) return false;
			const status = memberStatuses.get(m.id);
			return !status || status === 'offline' || status === 'invisible';
		});

		if (offlineMembers.length > 0) {
			offlineMembers.sort((a, b) => {
				const nameA = a.nickname || a.user.display_name || a.user.username;
				const nameB = b.nickname || b.user.display_name || b.user.username;
				return nameA.localeCompare(nameB);
			});

			groups.push({
				role: null,
				members: offlineMembers,
				label: 'OFFLINE',
				isOffline: true
			});
		}

		return groups;
	}

	// Format activity display
	function formatActivity(activity: Activity): string {
		const prefix = getActivityLabel(activity.type);
		if (prefix) {
			return `${prefix} ${activity.name}`;
		}
		return activity.state || activity.name;
	}

	// Handle member click - show popout
	function handleMemberClick(event: MouseEvent, member: Member) {
		const target = event.currentTarget as HTMLElement;
		const rect = target.getBoundingClientRect();
		
		// Position the popout to the left of the member list
		const position = {
			x: rect.left,
			y: rect.top
		};

		// Build user data for popout
		const user = {
			id: member.user.id,
			username: member.user.username,
			display_name: member.user.display_name,
			avatar: member.user.avatar,
			banner: member.user.banner || null,
			bio: member.user.bio || null,
			pronouns: member.user.pronouns || null,
			bot: member.user.bot || false,
			created_at: member.user.created_at || new Date().toISOString()
		};

		// Build member data with roles
		const memberData = {
			nickname: member.nickname,
			joined_at: member.joined_at || new Date().toISOString(),
			roles: getMemberRoles(member, $roles)
		};

		popoutStore.open({
			user,
			member: memberData,
			position,
			anchor: 'left',
			// TODO: Fetch mutual servers/friends from API
			mutualServers: [],
			mutualFriends: []
		});
	}

	// Handle context menu
	function handleContextMenu(event: MouseEvent, member: Member) {
		event.preventDefault();
		// TODO: Show context menu with options like:
		// - Profile, Mention, Message, Mute, Kick, Ban, etc.
	}

	// Keyboard navigation for member list
	function getMemberButtons(): HTMLElement[] {
		if (!memberListElement) return [];
		return Array.from(memberListElement.querySelectorAll<HTMLElement>('.member-item'));
	}

	function handleKeydown(event: KeyboardEvent) {
		const buttons = getMemberButtons();
		if (buttons.length === 0) return;

		const currentButton = document.activeElement as HTMLElement;
		const currentIndex = buttons.indexOf(currentButton);
		if (currentIndex === -1) return;

		const { handled, newIndex } = handleListKeyboard(event, currentIndex, buttons.length, {
			wrap: true
		});

		if (handled && newIndex !== currentIndex) {
			buttons[newIndex]?.focus();
		}
	}
</script>

{#if $currentServer}
	<aside 
		bind:this={memberListElement}
		class="member-list"
		aria-label="Server members"
		on:keydown={handleKeydown}
	>
		{#each groupedMembers as group}
			<div class="member-group">
				<!-- Group Header -->
				<h3 class="group-header">
					{group.label} — {group.members.length}
				</h3>

				<!-- Members -->
				{#each group.members as member}
					{@const activity = getMemberActivity(member.user.id)}
					{@const memberColor = getMemberColor(member, $roles)}
					<button
						class="member-item"
						class:offline={group.isOffline}
						on:click={(e) => handleMemberClick(e, member)}
						on:contextmenu={(e) => handleContextMenu(e, member)}
						aria-label="View {member.nickname || member.user.display_name || member.user.username}'s profile"
					>
						<!-- Avatar with status indicator -->
						<Avatar
							src={member.user.avatar}
							username={member.user.username}
							size="sm"
							userId={member.user.id}
							showPresence={true}
						/>

						<!-- Member info -->
						<div class="member-info">
							<span
								class="member-name"
								style="color: {memberColor}"
							>
								{member.nickname || member.user.display_name || member.user.username}
							</span>

							{#if activity && !group.isOffline}
								<span class="member-activity">
									{formatActivity(activity)}
								</span>
							{/if}
						</div>
					</button>
				{/each}
			</div>
		{/each}
	</aside>
{/if}

<style>
	.member-list {
		width: 240px;
		height: 100%;
		background-color: var(--bg-secondary, #2b2d31);
		overflow-y: auto;
		overflow-x: hidden;
		padding: 8px 0;
		flex-shrink: 0;
	}

	.member-group {
		margin-bottom: 8px;
	}

	.group-header {
		padding: 24px 8px 4px 16px;
		font-size: 12px;
		font-weight: 600;
		color: var(--text-muted, #949ba4);
		text-transform: uppercase;
		letter-spacing: 0.02em;
		line-height: 1.3;
	}

	.member-group:first-child .group-header {
		padding-top: 8px;
	}

	.member-item {
		display: flex;
		align-items: center;
		gap: 12px;
		width: calc(100% - 16px);
		margin: 0 8px;
		padding: 4px 8px;
		border-radius: 4px;
		background: transparent;
		border: none;
		cursor: pointer;
		transition: background-color 0.1s ease;
		text-align: left;
	}

	.member-item:hover {
		background-color: var(--bg-modifier-hover, #35373c);
	}

	.member-item:active {
		background-color: var(--bg-modifier-active, #404249);
	}

	.member-item:focus-visible {
		outline: 2px solid var(--brand-primary, #5865f2);
		outline-offset: -2px;
	}

	.member-item.offline {
		opacity: 0.3;
	}

	.member-item.offline:hover {
		opacity: 0.6;
	}

	.member-info {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
		gap: 1px;
	}

	.member-name {
		font-size: 14px;
		font-weight: 500;
		line-height: 1.3;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.member-activity {
		font-size: 12px;
		color: var(--text-muted, #b5bac1);
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
		line-height: 1.3;
	}
</style>
