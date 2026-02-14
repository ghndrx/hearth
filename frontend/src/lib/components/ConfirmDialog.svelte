<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import Modal from './Modal.svelte';

	export let open = false;
	export let title = '';
	export let message = '';
	export let confirmLabel = 'Confirm';
	export let cancelLabel = 'Cancel';
	export let confirmVariant: 'danger' | 'primary' | 'brand' = 'primary';
	export let loading = false;

	const dispatch = createEventDispatcher();

	function handleConfirm() {
		dispatch('confirm');
	}

	function handleCancel() {
		dispatch('cancel');
		open = false;
	}

	function handleClose() {
		dispatch('cancel');
	}
</script>

<Modal {open} {title} size="small" on:close={handleClose}>
	<p class="message">{message}</p>

	<svelte:fragment slot="footer">
		<button class="btn secondary" on:click={handleCancel} disabled={loading}>
			{cancelLabel}
		</button>
		<button class="btn {confirmVariant}" on:click={handleConfirm} disabled={loading}>
			{loading ? 'Loading...' : confirmLabel}
		</button>
	</svelte:fragment>
</Modal>

<style>
	.message {
		margin: 0;
		font-size: 16px;
		color: var(--text-secondary);
		line-height: 1.5;
	}

	.btn {
		padding: 10px 24px;
		border-radius: 4px;
		font-size: 14px;
		font-weight: 500;
		cursor: pointer;
		border: none;
		transition: background-color 0.15s ease;
	}

	.btn.primary {
		background: var(--brand-primary);
		color: white;
	}

	.btn.primary:hover:not(:disabled) {
		background: var(--brand-hover);
	}

	.btn.brand {
		background: var(--brand-primary);
		color: white;
	}

	.btn.brand:hover:not(:disabled) {
		background: var(--brand-hover);
	}

	.btn.danger {
		background: var(--status-danger);
		color: white;
	}

	.btn.danger:hover:not(:disabled) {
		background: #d9383f;
	}

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
