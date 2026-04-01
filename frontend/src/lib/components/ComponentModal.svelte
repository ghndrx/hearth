<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import { fade, scale } from 'svelte/transition';

	export let open = false;
	export let title: string = '';
	export let modalType: 'primary' | 'danger' = 'primary';
	export let customId: string = '';
	export let rows: Array<{
		components: Array<{
			id: string;
			type: string;
			custom_id?: string;
			label?: string;
			style?: string;
			placeholder?: string;
			required?: boolean;
			value?: string;
			min_length?: number;
			max_length?: number;
			options?: Array<{
				label: string;
				value: string;
				description?: string;
				emoji?: string;
			}>;
		}>;
	}> = [];

	const dispatch = createEventDispatcher<{
		close: void;
		submit: { customId: string; values: Record<string, string> };
	}>();

	interface TextInputValue {
		[customId: string]: string;
	}

	let textInputs: TextInputValue = {};
	let selectValues: TextInputValue = {};

	function close() {
		open = false;
		dispatch('close');
	}

	function handleKeydown(e: KeyboardEvent) {
		if (!open) return;
		if (e.key === 'Escape') {
			e.preventDefault();
			close();
		}
	}

	function handleBackdropClick(e: MouseEvent) {
		if (e.target === e.currentTarget) {
			close();
		}
	}

	function handleTextInput(customId: string, value: string) {
		textInputs[customId] = value;
	}

	function handleSelect(customId: string, values: string[]) {
		selectValues[customId] = values[0] || '';
	}

	function handleSubmit() {
		// Collect all values
		const values: Record<string, string> = { ...textInputs, ...selectValues };
		dispatch('submit', { customId, values });
	}

	function getTextInputStyle(style?: string): string {
		switch (style) {
			case 'paragraph':
				return 'textarea';
			default:
				return 'input';
		}
	}

	$: allValues = { ...textInputs, ...selectValues };
</script>

<svelte:window on:keydown={handleKeydown} />

{#if open}
	<!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
	<div class="modal-backdrop" transition:fade={{ duration: 150 }} on:click={handleBackdropClick}>
		<div
			class="modal {modalType}"
			role="dialog"
			aria-modal="true"
			aria-labelledby="component-modal-title"
			transition:scale={{ duration: 200, start: 0.95 }}
		>
			<div class="modal-header">
				<h2 id="component-modal-title">{title}</h2>
				<button class="close-btn" on:click={close} aria-label="Close modal" type="button">
					<svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor">
						<path d="M18.4 4L12 10.4L5.6 4L4 5.6L10.4 12L4 18.4L5.6 20L12 13.6L18.4 20L20 18.4L13.6 12L20 5.6L18.4 4Z"/>
					</svg>
				</button>
			</div>

			<div class="modal-content">
				{#each rows as row}
					<div class="modal-row">
						{#each row.components as component}
							{#if component.type === 'text_input'}
								<div class="text-input-wrapper">
									{#if component.label}
										<label class="input-label" for={component.custom_id}>
											{component.label}
											{#if component.required}
												<span class="required">*</span>
											{/if}
										</label>
									{/if}
									{#if getTextInputStyle(component.style) === 'textarea'}
										<textarea
											id={component.custom_id}
											class="text-input"
											placeholder={component.placeholder}
											value={component.value || ''}
											minlength={component.min_length}
											maxlength={component.max_length}
											on:input={(e) => handleTextInput(component.custom_id || '', e.currentTarget.value)}
										></textarea>
									{:else}
										<input
											id={component.custom_id}
											type="text"
											class="text-input"
											placeholder={component.placeholder}
											value={component.value || ''}
											minlength={component.min_length}
											maxlength={component.max_length}
											on:input={(e) => handleTextInput(component.custom_id || '', e.currentTarget.value)}
										/>
									{/if}
								</div>
							{/if}
						{/each}
					</div>
				{/each}
			</div>

			<div class="modal-footer">
				<button type="button" class="cancel-btn" on:click={close}>
					Cancel
				</button>
				<button type="button" class="submit-btn {modalType}" on:click={handleSubmit}>
					Submit
				</button>
			</div>
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
		position: relative;
		background: var(--bg-primary, #313338);
		border-radius: 4px;
		max-height: calc(100vh - 80px);
		overflow: hidden;
		display: flex;
		flex-direction: column;
		box-shadow: 0 0 0 1px rgba(0, 0, 0, 0.15),
		            0 8px 16px rgba(0, 0, 0, 0.24);
		width: 100%;
		max-width: 500px;
	}

	.modal-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 16px 20px;
		border-bottom: 1px solid var(--bg-modifier-accent, #1e1f22);
	}

	.modal-header h2 {
		font-size: 20px;
		font-weight: 600;
		color: var(--text-normal, #f2f3f5);
		margin: 0;
	}

	.close-btn {
		background: none;
		border: none;
		color: var(--text-muted, #b5bac1);
		cursor: pointer;
		padding: 4px;
		border-radius: 4px;
		display: flex;
		align-items: center;
		justify-content: center;
		transition: color 0.15s ease;
	}

	.close-btn:hover {
		color: var(--text-normal, #f2f3f5);
	}

	.modal-content {
		padding: 20px;
		overflow-y: auto;
		flex: 1;
	}

	.modal-row {
		margin-bottom: 16px;
	}

	.modal-row:last-child {
		margin-bottom: 0;
	}

	.text-input-wrapper {
		display: flex;
		flex-direction: column;
		gap: 6px;
	}

	.input-label {
		font-size: 13px;
		font-weight: 500;
		color: var(--text-normal, #f2f3f5);
	}

	.required {
		color: var(--red, #f23f43);
	}

	.text-input {
		padding: 10px 12px;
		background: var(--bg-secondary, #2b2d31);
		border: 1px solid var(--bg-modifier-accent, #1e1f22);
		border-radius: 4px;
		color: var(--text-normal, #f2f3f5);
		font-size: 14px;
		font-family: inherit;
		transition: border-color 0.15s ease;
	}

	.text-input:focus {
		outline: none;
		border-color: var(--blurple, #5865f2);
	}

	.text-input::placeholder {
		color: var(--text-muted, #b5bac1);
	}

	textarea.text-input {
		min-height: 80px;
		resize: vertical;
	}

	.modal-footer {
		display: flex;
		justify-content: flex-end;
		gap: 8px;
		padding: 16px 20px;
		background: var(--bg-secondary, #2b2d31);
		border-top: 1px solid var(--bg-modifier-accent, #1e1f22);
	}

	.cancel-btn {
		padding: 10px 16px;
		background: transparent;
		border: none;
		border-radius: 4px;
		color: var(--text-normal, #f2f3f5);
		font-size: 14px;
		font-weight: 500;
		cursor: pointer;
		transition: background-color 0.15s ease;
	}

	.cancel-btn:hover {
		background: var(--bg-modifier-hover, #35373c);
	}

	.submit-btn {
		padding: 10px 16px;
		border: none;
		border-radius: 4px;
		font-size: 14px;
		font-weight: 500;
		cursor: pointer;
		transition: background-color 0.15s ease;
	}

	.submit-btn.primary {
		background: var(--blurple, #5865f2);
		color: white;
	}

	.submit-btn.primary:hover {
		background: var(--blurple-hover, #4752c4);
	}

	.submit-btn.danger {
		background: var(--red, #f23f43);
		color: white;
	}

	.submit-btn.danger:hover {
		background: #d63839;
	}
</style>
