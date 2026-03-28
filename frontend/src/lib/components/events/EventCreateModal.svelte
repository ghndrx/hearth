<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import { 
		createEvent, 
		EVENT_TYPE_LABELS, 
		type CreateEventRequest,
		type EventType 
	} from '$lib/stores/events';
	import type { Channel } from '$lib/stores/channels';
	import { channels } from '$lib/stores/channels';

	export let serverId: string;

	const dispatch = createEventDispatcher<{
		close: void;
		created: { event: unknown };
	}>();

	let name = '';
	let description = '';
	let scheduledDate = '';
	let scheduledTime = '';
	let scheduledEndDate = '';
	let scheduledEndTime = '';
	let entityType: EventType = 1;
	let channelId: string | null = null;
	let location = '';
	let loading = false;
	let error: string | null = null;

	// Get voice/stage channels for entity types 1 and 2
	$: voiceChannels = $channels.filter(c => 
		c.type === 2 || c.type === 5 // voice or stage channel
	);

	function handleClose() {
		dispatch('close');
	}

	async function handleSubmit() {
		if (!name.trim()) {
			error = 'Event name is required';
			return;
		}

		if (!scheduledDate || !scheduledTime) {
			error = 'Start date and time are required';
			return;
		}

		// For stage/voice events, channel is required
		if (entityType !== 3 && !channelId) {
			error = 'Channel is required for stage and voice events';
			return;
		}

		loading = true;
		error = null;

		try {
			const scheduledStart = new Date(`${scheduledDate}T${scheduledTime}`).toISOString();
			
			let scheduledEnd: string | null = null;
			if (scheduledEndDate && scheduledEndTime) {
				scheduledEnd = new Date(`${scheduledEndDate}T${scheduledEndTime}`).toISOString();
			}

			const request: CreateEventRequest = {
				name: name.trim(),
				description: description.trim() || undefined,
				scheduled_start: scheduledStart,
				scheduled_end: scheduledEnd || undefined,
				entity_type: entityType,
				channel_id: channelId,
				location: location.trim() || undefined,
			};

			const event = await createEvent(serverId, request);
			dispatch('created', { event });
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to create event';
		} finally {
			loading = false;
		}
	}

	function handleOverlayClick(e: MouseEvent) {
		if (e.target === e.currentTarget) {
			handleClose();
		}
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') {
			handleClose();
		}
	}
</script>

<svelte:window on:keydown={handleKeydown} />

<div class="modal-overlay" on:click={handleOverlayClick}>
	<div class="modal-content">
		<div class="modal-header">
			<h2>Create Event</h2>
			<button class="close-btn" on:click={handleClose}>
				<svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor">
					<path d="M19 6.41L17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z"/>
				</svg>
			</button>
		</div>

		<form class="modal-body" on:submit|preventDefault={handleSubmit}>
			{#if error}
				<div class="error-message">{error}</div>
			{/if}

			<div class="form-group">
				<label for="event-name">Event Name *</label>
				<input
					id="event-name"
					type="text"
					bind:value={name}
					placeholder="Give your event a name"
					maxlength="100"
					required
				/>
			</div>

			<div class="form-group">
				<label for="event-description">Description</label>
				<textarea
					id="event-description"
					bind:value={description}
					placeholder="What's the event about?"
					rows="3"
					maxlength="1000"
				></textarea>
			</div>

			<div class="form-group">
				<label>Event Type *</label>
				<div class="entity-type-selector">
					{#each ([1, 2, 3] as EventType[]) as type}
						<label class="radio-label">
							<input
								type="radio"
								name="entityType"
								value={type}
								bind:group={entityType}
							/>
							<span class="radio-text">{EVENT_TYPE_LABELS[type]}</span>
						</label>
					{/each}
				</div>
			</div>

			{#if entityType !== 3}
				<div class="form-group">
					<label for="event-channel">Channel *</label>
					<select id="event-channel" bind:value={channelId}>
						<option value={null}>Select a channel</option>
						{#each voiceChannels as channel}
							<option value={channel.id}>{channel.name}</option>
						{/each}
					</select>
				</div>
			{:else}
				<div class="form-group">
					<label for="event-location">Location</label>
					<input
						id="event-location"
						type="text"
						bind:value={location}
						placeholder="Where is this event taking place?"
						maxlength="100"
					/>
				</div>
			{/if}

			<div class="form-row">
				<div class="form-group">
					<label for="event-start-date">Start Date *</label>
					<input
						id="event-start-date"
						type="date"
						bind:value={scheduledDate}
						required
					/>
				</div>
				<div class="form-group">
					<label for="event-start-time">Start Time *</label>
					<input
						id="event-start-time"
						type="time"
						bind:value={scheduledTime}
						required
					/>
				</div>
			</div>

			<div class="form-row">
				<div class="form-group">
					<label for="event-end-date">End Date</label>
					<input
						id="event-end-date"
						type="date"
						bind:value={scheduledEndDate}
					/>
				</div>
				<div class="form-group">
					<label for="event-end-time">End Time</label>
					<input
						id="event-end-time"
						type="time"
						bind:value={scheduledEndTime}
					/>
				</div>
			</div>
		</form>

		<div class="modal-footer">
			<button type="button" class="btn-secondary" on:click={handleClose}>
				Cancel
			</button>
			<button 
				type="submit" 
				class="btn-primary" 
				on:click={handleSubmit}
				disabled={loading}
			>
				{loading ? 'Creating...' : 'Create Event'}
			</button>
		</div>
	</div>
</div>

<style>
	.modal-overlay {
		position: fixed;
		top: 0;
		left: 0;
		right: 0;
		bottom: 0;
		background: rgba(0, 0, 0, 0.7);
		display: flex;
		align-items: center;
		justify-content: center;
		z-index: 1000;
	}

	.modal-content {
		background: var(--bg-secondary);
		border-radius: 8px;
		width: 100%;
		max-width: 520px;
		max-height: 90vh;
		overflow-y: auto;
		box-shadow: 0 8px 24px rgba(0, 0, 0, 0.4);
	}

	.modal-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 16px 20px;
		border-bottom: 1px solid rgba(0, 0, 0, 0.24);
	}

	.modal-header h2 {
		margin: 0;
		font-size: 18px;
		font-weight: 600;
		color: var(--text-primary);
	}

	.close-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 4px;
		background: none;
		border: none;
		color: var(--text-muted);
		cursor: pointer;
		border-radius: 4px;
		transition: background 0.15s ease;
	}

	.close-btn:hover {
		background: rgba(79, 84, 92, 0.32);
		color: var(--text-primary);
	}

	.modal-body {
		padding: 20px;
	}

	.error-message {
		background: rgba(242, 63, 67, 0.15);
		border: 1px solid rgba(242, 63, 67, 0.3);
		color: #f23f43;
		padding: 10px 12px;
		border-radius: 4px;
		font-size: 13px;
		margin-bottom: 16px;
	}

	.form-group {
		margin-bottom: 16px;
	}

	.form-group label {
		display: block;
		margin-bottom: 6px;
		font-size: 13px;
		font-weight: 500;
		color: var(--text-primary);
	}

	.form-group input,
	.form-group select,
	.form-group textarea {
		width: 100%;
		padding: 10px 12px;
		background: var(--bg-primary);
		border: 1px solid rgba(0, 0, 0, 0.3);
		border-radius: 4px;
		color: var(--text-primary);
		font-size: 14px;
		font-family: inherit;
		box-sizing: border-box;
	}

	.form-group input:focus,
	.form-group select:focus,
	.form-group textarea:focus {
		outline: none;
		border-color: var(--brand-primary);
	}

	.form-group textarea {
		resize: vertical;
		min-height: 80px;
	}

	.form-row {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 12px;
	}

	.entity-type-selector {
		display: flex;
		gap: 12px;
	}

	.radio-label {
		display: flex;
		align-items: center;
		gap: 6px;
		cursor: pointer;
	}

	.radio-label input {
		width: auto;
	}

	.radio-text {
		font-size: 14px;
		color: var(--text-secondary);
	}

	.modal-footer {
		display: flex;
		justify-content: flex-end;
		gap: 12px;
		padding: 16px 20px;
		border-top: 1px solid rgba(0, 0, 0, 0.24);
	}

	.btn-secondary,
	.btn-primary {
		padding: 10px 20px;
		border-radius: 4px;
		font-size: 14px;
		font-weight: 500;
		cursor: pointer;
		transition: background 0.15s ease;
	}

	.btn-secondary {
		background: transparent;
		border: 1px solid rgba(79, 84, 92, 0.48);
		color: var(--text-primary);
	}

	.btn-secondary:hover {
		background: rgba(79, 84, 92, 0.24);
	}

	.btn-primary {
		background: var(--brand-primary);
		border: none;
		color: white;
	}

	.btn-primary:hover:not(:disabled) {
		background: var(--brand-hover);
	}

	.btn-primary:disabled {
		opacity: 0.6;
		cursor: not-allowed;
	}
</style>
