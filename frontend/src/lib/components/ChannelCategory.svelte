<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import { slide } from 'svelte/transition';
	import { cubicOut } from 'svelte/easing';
	import { prefersReducedMotion } from '$lib/utils/transitions';

	export let name: string;
	export let categoryId: string = '';
	export let collapsed = false;
	export let showAddButton = true;
	export let editable = false;

	let editing = false;
	let editName = name;
	let editInput: HTMLInputElement;
	let showContextMenu = false;
	let contextMenuX = 0;
	let contextMenuY = 0;

	const dispatch = createEventDispatcher<{
		toggle: { collapsed: boolean };
		addChannel: void;
		rename: { id: string; name: string };
		deleteCategory: { id: string };
	}>();

	function handleToggle() {
		collapsed = !collapsed;
		dispatch('toggle', { collapsed });
	}

	function handleAddChannel(e: MouseEvent) {
		e.stopPropagation();
		dispatch('addChannel');
	}

	function handleContextMenu(e: MouseEvent) {
		if (!editable) return;
		e.preventDefault();
		e.stopPropagation();
		contextMenuX = e.clientX;
		contextMenuY = e.clientY;
		showContextMenu = true;
	}

	function startRename() {
		showContextMenu = false;
		editName = name;
		editing = true;
		requestAnimationFrame(() => {
			if (editInput) {
				editInput.focus();
				editInput.select();
			}
		});
	}

	function finishRename() {
		editing = false;
		const trimmed = editName.trim();
		if (trimmed && trimmed !== name && categoryId) {
			dispatch('rename', { id: categoryId, name: trimmed });
		}
	}

	function handleRenameKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter') {
			e.preventDefault();
			finishRename();
		} else if (e.key === 'Escape') {
			editing = false;
		}
	}

	function handleDelete() {
		showContextMenu = false;
		if (categoryId) {
			dispatch('deleteCategory', { id: categoryId });
		}
	}

	function handleClickOutside() {
		showContextMenu = false;
	}
</script>

<svelte:window on:click={handleClickOutside} />

<div class="channel-category" class:collapsed on:contextmenu={handleContextMenu} role="group">
	{#if editing}
		<input
			bind:this={editInput}
			bind:value={editName}
			on:blur={finishRename}
			on:keydown={handleRenameKeydown}
			class="category-edit-input"
			maxlength="100"
		/>
	{:else}
		<button
			class="category-header"
			on:click={handleToggle}
			aria-expanded={!collapsed}
			aria-label="{name} category, {collapsed ? 'collapsed' : 'expanded'}"
			type="button"
		>
			<svg
				viewBox="0 0 24 24"
				width="12"
				height="12"
				fill="currentColor"
				class="collapse-icon"
				class:rotated={!collapsed}
				aria-hidden="true"
			>
				<path d="M9.29 15.88L13.17 12 9.29 8.12a1 1 0 0 1 1.42-1.42l4.59 4.59a1 1 0 0 1 0 1.42l-4.59 4.59a1 1 0 0 1-1.42 0 1 1 0 0 1 0-1.42z"/>
			</svg>
			<span class="category-name">{name.toUpperCase()}</span>
		</button>
	{/if}

	{#if showAddButton}
		<button
			class="add-channel"
			title="Create Channel"
			aria-label="Create channel in {name}"
			on:click={handleAddChannel}
			type="button"
		>
			<svg viewBox="0 0 24 24" width="18" height="18" fill="currentColor" aria-hidden="true">
				<path d="M19 13h-6v6h-2v-6H5v-2h6V5h2v6h6v2z"/>
			</svg>
		</button>
	{/if}
</div>

{#if !collapsed}
	<div
		class="category-channels"
		role="group"
		aria-label="{name} channels"
		transition:slide={{ duration: prefersReducedMotion() ? 0 : 150, easing: cubicOut }}
	>
		<slot />
	</div>
{/if}

<!-- Context Menu -->
{#if showContextMenu && editable}
	<div
		class="context-menu"
		style="left: {contextMenuX}px; top: {contextMenuY}px;"
		on:click|stopPropagation
		role="menu"
		tabindex="-1"
	>
		<button class="context-menu-item" on:click={startRename} type="button" role="menuitem">
			Rename Category
		</button>
		<button class="context-menu-item" on:click={() => dispatch('addChannel')} type="button" role="menuitem">
			Create Channel
		</button>
		<button class="context-menu-item danger" on:click={handleDelete} type="button" role="menuitem">
			Delete Category
		</button>
	</div>
{/if}

<style>
	.channel-category {
		display: flex;
		align-items: center;
		padding: 16px 8px 4px 2px;
		user-select: none;
	}

	.category-header {
		display: flex;
		align-items: center;
		gap: 2px;
		background: none;
		border: none;
		color: #949ba4;
		font-size: 12px;
		font-weight: 600;
		letter-spacing: 0.02em;
		cursor: pointer;
		flex: 1;
		text-align: left;
		padding: 0;
		text-transform: uppercase;
		transition: color 0.15s ease;
	}

	.category-header:hover {
		color: #dbdee1;
	}

	.category-header:focus-visible {
		outline: 2px solid var(--brand-primary, #5865f2);
		outline-offset: 2px;
		color: #dbdee1;
	}

	.collapse-icon {
		transition: transform 0.1s ease;
		flex-shrink: 0;
	}

	.collapse-icon.rotated {
		transform: rotate(90deg);
	}

	.category-name {
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.category-edit-input {
		flex: 1;
		background: #1e1f22;
		border: 1px solid #5865f2;
		border-radius: 3px;
		color: #f2f3f5;
		font-size: 12px;
		font-weight: 600;
		letter-spacing: 0.02em;
		padding: 2px 6px;
		text-transform: uppercase;
		outline: none;
	}

	.add-channel {
		background: none;
		border: none;
		color: #949ba4;
		cursor: pointer;
		padding: 0;
		opacity: 0;
		display: flex;
		align-items: center;
		justify-content: center;
		width: 18px;
		height: 18px;
		transition: opacity 0.15s ease, color 0.15s ease;
	}

	.add-channel:hover {
		color: #dbdee1;
	}

	.add-channel:focus-visible {
		outline: 2px solid var(--brand-primary, #5865f2);
		outline-offset: 2px;
		opacity: 1;
	}

	.channel-category:hover .add-channel {
		opacity: 1;
	}

	.category-channels {
		display: flex;
		flex-direction: column;
	}

	/* Context Menu */
	.context-menu {
		position: fixed;
		z-index: 1000;
		background: #111214;
		border-radius: 4px;
		padding: 6px 0;
		min-width: 160px;
		box-shadow: 0 8px 16px rgba(0, 0, 0, 0.24);
	}

	.context-menu-item {
		display: block;
		width: 100%;
		padding: 6px 12px;
		background: none;
		border: none;
		color: #b5bac1;
		font-size: 14px;
		cursor: pointer;
		text-align: left;
		transition: background-color 0.1s ease, color 0.1s ease;
	}

	.context-menu-item:hover {
		background: #5865f2;
		color: #ffffff;
	}

	.context-menu-item.danger:hover {
		background: #da373c;
		color: #ffffff;
	}
</style>
