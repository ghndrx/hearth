<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import { createServer } from '$lib/stores/servers';
	import Modal from './Modal.svelte';

	export let open = false;

	const dispatch = createEventDispatcher();

	let step: 'choose' | 'create' | 'join' = 'choose';
	let name = '';
	let inviteCode = '';
	let loading = false;
	let error = '';

	function reset() {
		step = 'choose';
		name = '';
		inviteCode = '';
		error = '';
	}

	function close() {
		open = false;
		reset();
	}

	async function handleCreate() {
		if (!name.trim()) return;

		loading = true;
		error = '';

		try {
			const server = await createServer(name.trim());
			dispatch('created', server);
			close();
		} catch (e: any) {
			error = e.message || 'Failed to create server';
		} finally {
			loading = false;
		}
	}

	async function handleJoin() {
		if (!inviteCode.trim()) return;

		loading = true;
		error = '';

		try {
			// Extract code from full URL if pasted
			let code = inviteCode.trim();
			const match = code.match(/(?:hearth\.chat\/|discord\.gg\/)?([a-zA-Z0-9]+)$/);
			if (match) code = match[1];

			const response = await fetch(`/api/v1/invites/${code}`, {
				method: 'POST',
				headers: {
					Authorization: `Bearer ${localStorage.getItem('hearth_token')}`
				}
			});

			if (!response.ok) throw new Error('Invalid invite');

			const server = await response.json();
			dispatch('joined', server);
			close();
		} catch (e: any) {
			error = e.message || 'Failed to join server';
		} finally {
			loading = false;
		}
	}
</script>

<Modal
	{open}
	title={step === 'choose'
		? 'Add a Server'
		: step === 'create'
			? 'Create a Server'
			: 'Join a Server'}
	on:close={close}
>
	{#if step === 'choose'}
		<div class="options">
			<button class="option" on:click={() => (step = 'create')}>
				<div class="option-icon">🏠</div>
				<div class="option-info">
					<span class="option-title">Create My Own</span>
					<span class="option-desc">Start a new community</span>
				</div>
				<span class="arrow">→</span>
			</button>

			<div class="divider">
				<span>or</span>
			</div>

			<button class="option" on:click={() => (step = 'join')}>
				<div class="option-icon">🔗</div>
				<div class="option-info">
					<span class="option-title">Join a Server</span>
					<span class="option-desc">Enter an invite link</span>
				</div>
				<span class="arrow">→</span>
			</button>
		</div>
	{:else if step === 'create'}
		<form on:submit|preventDefault={handleCreate}>
			{#if error}
				<div class="error">{error}</div>
			{/if}

			<div class="form-group">
				<label for="server-name">SERVER NAME</label>
				<input
					type="text"
					id="server-name"
					bind:value={name}
					placeholder="My Awesome Server"
					required
					disabled={loading}
				/>
			</div>

			<p class="hint">By creating a server, you agree to Hearth's Community Guidelines.</p>
		</form>
	{:else}
		<form on:submit|preventDefault={handleJoin}>
			{#if error}
				<div class="error">{error}</div>
			{/if}

			<div class="form-group">
				<label for="invite-link">INVITE LINK</label>
				<input
					type="text"
					id="invite-link"
					bind:value={inviteCode}
					placeholder="https://hearth.chat/AbCdEf or AbCdEf"
					required
					disabled={loading}
				/>
			</div>

			<p class="hint">Invites look like: https://hearth.chat/AbCdEf or AbCdEf</p>
		</form>
	{/if}

	<svelte:fragment slot="footer">
		{#if step !== 'choose'}
			<button class="btn secondary" on:click={() => (step = 'choose')} disabled={loading}>
				Back
			</button>
		{/if}

		{#if step === 'create'}
			<button class="btn primary" on:click={handleCreate} disabled={loading || !name.trim()}>
				{loading ? 'Creating...' : 'Create'}
			</button>
		{:else if step === 'join'}
			<button class="btn primary" on:click={handleJoin} disabled={loading || !inviteCode.trim()}>
				{loading ? 'Joining...' : 'Join Server'}
			</button>
		{/if}
	</svelte:fragment>
</Modal>

<style>
	.options {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	.option {
		display: flex;
		align-items: center;
		gap: 16px;
		width: 100%;
		padding: 16px;
		background: var(--bg-secondary);
		border: 1px solid var(--bg-modifier-accent);
		border-radius: 8px;
		cursor: pointer;
		text-align: left;
		transition:
			background-color 0.15s ease,
			border-color 0.15s ease;
	}

	.option:hover {
		background: var(--bg-modifier-hover);
		border-color: var(--text-muted);
	}

	.option-icon {
		font-size: 32px;
	}

	.option-info {
		flex: 1;
	}

	.option-title {
		display: block;
		font-size: 16px;
		font-weight: 600;
		color: var(--text-primary);
	}

	.option-desc {
		display: block;
		font-size: 14px;
		color: var(--text-muted);
	}

	.arrow {
		font-size: 20px;
		color: var(--text-muted);
	}

	.divider {
		display: flex;
		align-items: center;
		gap: 16px;
		margin: 16px 0;
	}

	.divider::before,
	.divider::after {
		content: '';
		flex: 1;
		height: 1px;
		background: var(--bg-modifier-accent);
	}

	.divider span {
		font-size: 12px;
		color: var(--text-muted);
		text-transform: uppercase;
	}

	.error {
		background: rgba(218, 55, 60, 0.1);
		border: 1px solid #da373c;
		color: #da373c;
		padding: 10px;
		border-radius: 3px;
		margin-bottom: 16px;
		font-size: 14px;
	}

	.form-group {
		margin-bottom: 16px;
	}

	/* PRD Section 4.2: Text Input styling */
	label {
		display: block;
		margin-bottom: 8px;
		font-size: 12px;
		font-weight: 700;
		color: var(--text-muted);
		letter-spacing: 0.02em;
	}

	/* PRD Section 4.2: Text Input - background var(--bg-tertiary), border none, border-radius 3px, padding 10px */
	input {
		width: 100%;
		padding: 10px;
		background: var(--bg-tertiary);
		border: none;
		border-radius: 3px;
		color: var(--text-primary);
		font-size: 16px;
	}

	/* PRD Section 4.2: Focus State - outline none, box-shadow 0 0 0 2px var(--blurple) */
	input:focus {
		outline: none;
		box-shadow: 0 0 0 2px #5865f2;
	}

	.hint {
		font-size: 12px;
		color: var(--text-muted);
		margin: 0;
	}

	/* PRD Section 4.1 Button Styles */
	.btn {
		padding: 8px 16px;
		border-radius: 3px;
		font-size: 14px;
		font-weight: 500;
		cursor: pointer;
		border: none;
		transition: background-color 0.1s ease;
	}

	/* PRD Section 4.1: Primary Button - background var(--blurple), color white */
	.btn.primary {
		background: #5865f2;
		color: white;
	}

	.btn.primary:hover:not(:disabled) {
		background: #4752c4;
	}

	/* PRD Section 4.1: Secondary Button - background transparent, color var(--text-normal) */
	.btn.secondary {
		background: transparent;
		color: var(--text-primary);
	}

	.btn.secondary:hover:not(:disabled) {
		text-decoration: underline;
	}

	.btn:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}
</style>
