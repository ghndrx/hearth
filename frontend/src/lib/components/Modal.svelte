<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import { fade, scale } from 'svelte/transition';
	
	export let open = false;
	export let title = '';
	export let size: 'small' | 'medium' | 'large' = 'medium';
	
	const dispatch = createEventDispatcher();
	
	function close() {
		open = false;
		dispatch('close');
	}
	
	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') close();
	}
	
	function handleBackdropClick(e: MouseEvent) {
		if (e.target === e.currentTarget) close();
	}
</script>

<svelte:window on:keydown={handleKeydown} />

{#if open}
	<div 
		class="modal-backdrop"
		transition:fade={{ duration: 150 }}
		on:click={handleBackdropClick}
	>
		<div 
			class="modal {size}"
			transition:scale={{ duration: 150, start: 0.95 }}
		>
			{#if title}
				<div class="modal-header">
					<h2>{title}</h2>
					<button class="close-btn" on:click={close}>
						<svg viewBox="0 0 24 24" width="24" height="24" fill="currentColor">
							<path d="M18.3 5.71a.996.996 0 0 0-1.41 0L12 10.59 7.11 5.7A.996.996 0 1 0 5.7 7.11L10.59 12 5.7 16.89a.996.996 0 1 0 1.41 1.41L12 13.41l4.89 4.89a.996.996 0 1 0 1.41-1.41L13.41 12l4.89-4.89c.38-.38.38-1.02 0-1.4z"/>
						</svg>
					</button>
				</div>
			{/if}
			
			<div class="modal-content">
				<slot />
			</div>
			
			{#if $$slots.footer}
				<div class="modal-footer">
					<slot name="footer" />
				</div>
			{/if}
		</div>
	</div>
{/if}

<style>
	.modal-backdrop {
		position: fixed;
		inset: 0;
		background: rgba(0, 0, 0, 0.85);
		display: flex;
		align-items: center;
		justify-content: center;
		z-index: 1000;
		padding: 40px;
	}
	
	.modal {
		background: var(--bg-primary);
		border-radius: 8px;
		max-height: 100%;
		overflow: hidden;
		display: flex;
		flex-direction: column;
		box-shadow: var(--shadow-elevation-high);
	}
	
	.modal.small {
		width: 400px;
	}
	
	.modal.medium {
		width: 520px;
	}
	
	.modal.large {
		width: 720px;
	}
	
	.modal-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 16px;
		border-bottom: 1px solid var(--bg-modifier-accent);
	}
	
	.modal-header h2 {
		font-size: 20px;
		font-weight: 600;
		color: var(--text-primary);
		margin: 0;
	}
	
	.close-btn {
		background: none;
		border: none;
		color: var(--text-muted);
		cursor: pointer;
		padding: 4px;
		border-radius: 4px;
	}
	
	.close-btn:hover {
		background: var(--bg-modifier-hover);
		color: var(--text-primary);
	}
	
	.modal-content {
		padding: 16px;
		overflow-y: auto;
		flex: 1;
	}
	
	.modal-footer {
		display: flex;
		justify-content: flex-end;
		gap: 8px;
		padding: 16px;
		background: var(--bg-secondary);
	}
</style>
