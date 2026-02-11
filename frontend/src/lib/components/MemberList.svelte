<script lang="ts">
	import { currentServer } from '$lib/stores/servers';
	import { writable } from 'svelte/store';
	
	interface Member {
		id: string;
		user: {
			id: string;
			username: string;
			display_name: string | null;
			avatar: string | null;
		};
		nickname: string | null;
		roles: string[];
		status: 'online' | 'idle' | 'dnd' | 'offline';
	}
	
	interface Role {
		id: string;
		name: string;
		color: string;
		position: number;
	}
	
	// TODO: Load from API
	const members = writable<Member[]>([]);
	const roles = writable<Role[]>([]);
	
	$: groupedMembers = groupMembersByRole($members, $roles);
	
	function groupMembersByRole(members: Member[], roles: Role[]) {
		const groups: { role: Role | null; members: Member[] }[] = [];
		const roleMap = new Map(roles.map(r => [r.id, r]));
		const usedMembers = new Set<string>();
		
		// Sort roles by position
		const sortedRoles = [...roles].sort((a, b) => b.position - a.position);
		
		// Group members by their highest role
		for (const role of sortedRoles) {
			if (role.name === '@everyone') continue;
			
			const roleMembers = members.filter(m => 
				m.roles.includes(role.id) && !usedMembers.has(m.id)
			);
			
			if (roleMembers.length > 0) {
				groups.push({ role, members: roleMembers });
				roleMembers.forEach(m => usedMembers.add(m.id));
			}
		}
		
		// Online members without special roles
		const onlineMembers = members.filter(m => 
			!usedMembers.has(m.id) && m.status !== 'offline'
		);
		if (onlineMembers.length > 0) {
			groups.push({ role: null, members: onlineMembers });
		}
		
		// Offline members
		const offlineMembers = members.filter(m => 
			!usedMembers.has(m.id) && m.status === 'offline'
		);
		if (offlineMembers.length > 0) {
			groups.push({ 
				role: { id: 'offline', name: 'Offline', color: '', position: -1 }, 
				members: offlineMembers 
			});
		}
		
		return groups;
	}
	
	function getStatusColor(status: string) {
		switch (status) {
			case 'online': return 'var(--status-online)';
			case 'idle': return 'var(--status-idle)';
			case 'dnd': return 'var(--status-dnd)';
			default: return 'var(--status-offline)';
		}
	}
</script>

{#if $currentServer}
	<div class="member-list">
		{#each groupedMembers as group}
			<div class="member-group">
				<div class="group-header">
					{group.role?.name || 'Online'} — {group.members.length}
				</div>
				
				{#each group.members as member}
					<button class="member">
						<div class="avatar">
							{#if member.user.avatar}
								<img src={member.user.avatar} alt="" />
							{:else}
								<div class="avatar-placeholder">
									{(member.user.username)[0].toUpperCase()}
								</div>
							{/if}
							<div 
								class="status-indicator"
								style="background: {getStatusColor(member.status)}"
							></div>
						</div>
						
						<div class="member-info">
							<span 
								class="name"
								style="color: {group.role?.color || 'inherit'}"
							>
								{member.nickname || member.user.display_name || member.user.username}
							</span>
						</div>
					</button>
				{/each}
			</div>
		{/each}
	</div>
{/if}

<style>
	.member-list {
		width: 240px;
		background: var(--bg-secondary);
		overflow-y: auto;
		padding: 8px 8px 8px 0;
	}
	
	.member-group {
		margin-bottom: 8px;
	}
	
	.group-header {
		padding: 16px 8px 4px 16px;
		font-size: 12px;
		font-weight: 600;
		color: var(--text-muted);
		text-transform: uppercase;
		letter-spacing: 0.02em;
	}
	
	.member {
		display: flex;
		align-items: center;
		gap: 12px;
		width: 100%;
		padding: 6px 8px;
		margin-left: 8px;
		border-radius: 4px;
		background: none;
		border: none;
		cursor: pointer;
		text-align: left;
	}
	
	.member:hover {
		background: var(--bg-modifier-hover);
	}
	
	.avatar {
		position: relative;
		width: 32px;
		height: 32px;
		border-radius: 50%;
		overflow: visible;
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
		border: 3px solid var(--bg-secondary);
	}
	
	.member-info {
		flex: 1;
		min-width: 0;
	}
	
	.name {
		display: block;
		font-size: 14px;
		font-weight: 500;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
</style>
