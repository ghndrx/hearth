<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import Modal from './Modal.svelte';

	export let open = false;
	export let title = 'Are you sure?';
	export let message = '';
	export let confirmText = 'Confirm';
	export let cancelText = 'Cancel';
	export let danger = false;
	export let loading = false;

	const dispatch = createEventDispatcher();

	function handleConfirm() {
		if (loading) return;
		dispatch('confirm');
	}

	function handleCancel() {
		if (loading) return;
		open = false;
		dispatch('cancel');
	}

	function handleClose() {
		if (loading) return;
		dispatch('cancel');
	}
</script>

<Modal {open} {title} size="small" on:close={handleClose}>
	<div class="confirm-dialog">
		{#if message}
			<p class="message">{message}</p>
		{/if}
		<slot />
	</div>

	<svelte:fragment slot="footer">
		<button class="btn secondary" on:click={handleCancel} disabled={loading}>
			{cancelText}
		</button>
		<button class="btn {danger ? 'danger' : 'primary'}" on:click={handleConfirm} disabled={loading}>
			{loading ? 'Please wait...' : confirmText}
		</button>
	</svelte:fragment>
</Modal>

<style>
	.confirm-dialog {
		color: var(--text-secondary);
	}

	.message {
		margin: 0;
		font-size: 16px;
		line-height: 1.375;
		color: var(--text-secondary);
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

	/* Primary Button: background var(--blurple), color white */
	.btn.primary {
		background: #5865f2;
		color: white;
	}

	.btn.primary:hover:not(:disabled) {
		background: #4752c4;
	}

	/* Secondary Button: background transparent, color var(--text-normal) */
	.btn.secondary {
		background: transparent;
		color: var(--text-primary);
	}

	.btn.secondary:hover:not(:disabled) {
		text-decoration: underline;
	}

	/* Danger Button: background var(--red), color white */
	.btn.danger {
		background: #da373c;
		color: white;
	}

	.btn.danger:hover:not(:disabled) {
		background: #a12828;
	}

	.btn:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}
</style>
