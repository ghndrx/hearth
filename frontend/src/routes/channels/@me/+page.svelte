<script lang="ts">
	import { user } from '$lib/stores/auth';
	import { api } from '$lib/api';
	
	let activeTab = 'online';
	let showAddFriend = false;
	let friendUsername = '';
	let addFriendError = '';
	let addFriendSuccess = '';
	let isSubmitting = false;
	
	function setTab(tab: string) {
		if (tab === 'add') {
			showAddFriend = true;
		} else {
			activeTab = tab;
			showAddFriend = false;
		}
		addFriendError = '';
		addFriendSuccess = '';
	}
	
	async function handleAddFriend() {
		if (!friendUsername.trim()) {
			addFriendError = 'Please enter a username';
			return;
		}
		
		isSubmitting = true;
		addFriendError = '';
		addFriendSuccess = '';
		
		try {
			await api.post('/users/@me/relationships', {
				username: friendUsername.trim()
			});
			addFriendSuccess = `Friend request sent to ${friendUsername}!`;
			friendUsername = '';
		} catch (err: any) {
			addFriendError = err.message || 'Failed to send friend request';
		} finally {
			isSubmitting = false;
		}
	}
	
	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter' && !isSubmitting) {
			handleAddFriend();
		}
	}
</script>

<svelte:head>
	<title>Friends | Hearth</title>
</svelte:head>

<div class="friends-header">
	<div class="header-left">
		<span class="icon">👥</span>
		<span class="title">Friends</span>
	</div>
	
	<div class="header-tabs">
		<button 
			class="tab" 
			class:active={activeTab === 'online' && !showAddFriend}
			on:click={() => setTab('online')}
		>Online</button>
		<button 
			class="tab"
			class:active={activeTab === 'all' && !showAddFriend}
			on:click={() => setTab('all')}
		>All</button>
		<button 
			class="tab"
			class:active={activeTab === 'pending' && !showAddFriend}
			on:click={() => setTab('pending')}
		>Pending</button>
		<button 
			class="tab"
			class:active={activeTab === 'blocked' && !showAddFriend}
			on:click={() => setTab('blocked')}
		>Blocked</button>
		<button 
			class="tab add"
			class:active={showAddFriend}
			on:click={() => setTab('add')}
		>Add Friend</button>
	</div>
</div>

<div class="friends-content">
	{#if showAddFriend}
		<div class="add-friend-section">
			<h2>ADD FRIEND</h2>
			<p>You can add friends with their Hearth username.</p>
			
			<div class="add-friend-input-wrapper" class:error={addFriendError} class:success={addFriendSuccess}>
				<input 
					type="text" 
					placeholder="Enter a username#0000"
					bind:value={friendUsername}
					on:keydown={handleKeydown}
					disabled={isSubmitting}
				/>
				<button 
					class="send-request-btn"
					on:click={handleAddFriend}
					disabled={isSubmitting || !friendUsername.trim()}
				>
					{isSubmitting ? 'Sending...' : 'Send Friend Request'}
				</button>
			</div>
			
			{#if addFriendError}
				<p class="error-text">{addFriendError}</p>
			{/if}
			{#if addFriendSuccess}
				<p class="success-text">{addFriendSuccess}</p>
			{/if}
		</div>
	{:else}
		<div class="empty-state">
			<div class="illustration">🔥</div>
			<h2>Your hearth awaits friends</h2>
			<p>Add friends by their username to start chatting.</p>
		</div>
	{/if}
</div>

<style>
	.friends-header {
		display: flex;
		align-items: center;
		gap: 16px;
		padding: 0 16px;
		height: 48px;
		border-bottom: 1px solid var(--bg-modifier-accent);
		background: var(--bg-primary);
		flex-shrink: 0;
	}
	
	.header-left {
		display: flex;
		align-items: center;
		gap: 8px;
	}
	
	.icon {
		font-size: 24px;
	}
	
	.title {
		font-weight: 600;
		color: var(--text-primary);
	}
	
	.header-tabs {
		display: flex;
		gap: 16px;
		margin-left: 16px;
		padding-left: 16px;
		border-left: 1px solid var(--bg-modifier-accent);
	}
	
	.tab {
		background: none;
		border: none;
		color: var(--text-muted);
		font-size: 14px;
		font-weight: 500;
		cursor: pointer;
		padding: 4px 8px;
		border-radius: 4px;
	}
	
	.tab:hover {
		background: var(--bg-modifier-hover);
		color: var(--text-primary);
	}
	
	.tab.active {
		background: var(--bg-modifier-selected);
		color: var(--text-primary);
	}
	
	.tab.add {
		background: var(--text-positive);
		color: white;
	}
	
	.tab.add:hover {
		background: #1a8f4a;
	}
	
	.tab.add.active {
		background: #1a8f4a;
	}
	
	.friends-content {
		flex: 1;
		display: flex;
		align-items: flex-start;
		justify-content: center;
		padding: 32px;
	}
	
	.empty-state {
		text-align: center;
		max-width: 400px;
		margin-top: 100px;
	}
	
	.illustration {
		font-size: 100px;
		margin-bottom: 24px;
	}
	
	.empty-state h2 {
		font-size: 20px;
		color: var(--text-primary);
		margin-bottom: 8px;
	}
	
	.empty-state p {
		color: var(--text-muted);
	}
	
	/* Add Friend Section */
	.add-friend-section {
		width: 100%;
		max-width: 900px;
	}
	
	.add-friend-section h2 {
		font-size: 12px;
		font-weight: 700;
		text-transform: uppercase;
		color: var(--text-primary);
		margin-bottom: 8px;
	}
	
	.add-friend-section > p {
		color: var(--text-muted);
		font-size: 14px;
		margin-bottom: 16px;
	}
	
	.add-friend-input-wrapper {
		display: flex;
		align-items: center;
		background: var(--bg-tertiary);
		border-radius: 8px;
		padding: 4px 4px 4px 12px;
		border: 1px solid transparent;
	}
	
	.add-friend-input-wrapper:focus-within {
		border-color: var(--text-link);
	}
	
	.add-friend-input-wrapper.error {
		border-color: var(--text-danger);
	}
	
	.add-friend-input-wrapper.success {
		border-color: var(--text-positive);
	}
	
	.add-friend-input-wrapper input {
		flex: 1;
		background: none;
		border: none;
		color: var(--text-primary);
		font-size: 16px;
		padding: 12px 0;
		outline: none;
	}
	
	.add-friend-input-wrapper input::placeholder {
		color: var(--text-muted);
	}
	
	.send-request-btn {
		background: var(--text-link);
		color: white;
		border: none;
		border-radius: 4px;
		padding: 10px 16px;
		font-size: 14px;
		font-weight: 500;
		cursor: pointer;
		white-space: nowrap;
	}
	
	.send-request-btn:hover:not(:disabled) {
		background: #4752c4;
	}
	
	.send-request-btn:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}
	
	.error-text {
		color: var(--text-danger);
		font-size: 14px;
		margin-top: 8px;
	}
	
	.success-text {
		color: var(--text-positive);
		font-size: 14px;
		margin-top: 8px;
	}
</style>
