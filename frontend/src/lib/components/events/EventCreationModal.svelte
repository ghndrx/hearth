<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import { api } from '$lib/api';
	import Modal from '../Modal.svelte';

	export let serverId: string;

	const dispatch = createEventDispatcher<{
		close: void;
		created: {
			id: string;
			server_id: string;
			channel_id: string | null;
			creator_id: string;
			name: string;
			description: string;
			image_url: string | null;
			scheduled_start: string;
			scheduled_end: string | null;
			entity_type: number;
			location: string;
			status: number;
			user_count: number;
			created_at: string;
		};
	}>();

	// Entity types: 1=stage, 2=voice, 3=external
	let entityType: number = 3; // Default to external
	let name = '';
	let description = '';
	let location = '';
	let scheduledDate = '';
	let scheduledTime = '';
	let scheduledEndDate = '';
	let scheduledEndTime = '';
	let channelId: string | null = null;
	let imageUrl = '';

	let channels: Array<{ id: string; name: string; type: number }> = [];
	let loadingChannels = false;
	let submitting = false;
	let error = '';

	const entityTypes = [
		{ value: 1, label: 'Stage', icon: '🎤', desc: 'Host in a stage channel' },
		{ value: 2, label: 'Voice', icon: '🔊', desc: 'Host in a voice channel' },
		{ value: 3, label: 'External', icon: '📍', desc: 'Meet somewhere else' }
	];

	async function loadChannels() {
		loadingChannels = true;
		try {
			const serverChannels = await api.get<Array<{ id: string; name: string; type: number }>>(
				`/servers/${serverId}/channels`
			);
			// Filter to voice and stage channels (type 2 = voice, type 5 = stage in hearth)
			channels = serverChannels.filter(c => c.type === 2 || c.type === 5);
		} catch (err) {
			console.error('Failed to load channels:', err);
		} finally {
			loadingChannels = false;
		}
	}

	function validateForm(): boolean {
		if (!name.trim()) {
			error = 'Event name is required';
			return false;
		}
		if (name.length > 100) {
			error = 'Event name must be 100 characters or less';
			return false;
		}
		if (!scheduledDate || !scheduledTime) {
			error = 'Start date and time are required';
			return false;
		}
		if (entityType !== 3 && !channelId) {
			error = 'Please select a channel for stage/voice events';
			return false;
		}
		if (entityType === 3 && !location.trim()) {
			error = 'Location is required for external events';
			return false;
		}

		const startDateTime = new Date(`${scheduledDate}T${scheduledTime}`);
		if (startDateTime <= new Date()) {
			error = 'Event must be scheduled in the future';
			return false;
		}

		if (scheduledEndDate && scheduledEndTime) {
			const endDateTime = new Date(`${scheduledEndDate}T${scheduledEndTime}`);
			if (endDateTime <= startDateTime) {
				error = 'End time must be after start time';
				return false;
			}
		}

		return true;
	}

	async function handleSubmit() {
		if (!validateForm()) return;

		submitting = true;
		error = '';

		try {
			const startDateTime = new Date(`${scheduledDate}T${scheduledTime}`);
			const endDateTime = scheduledEndDate && scheduledEndTime
				? new Date(`${scheduledEndDate}T${scheduledEndTime}`)
				: null;

			const payload: Record<string, unknown> = {
				name: name.trim(),
				description: description.trim(),
				entity_type: entityType,
				scheduled_start: startDateTime.toISOString()
			};

			if (entityType !== 3 && channelId) {
				payload.channel_id = channelId;
			}

			if (entityType === 3) {
				payload.location = location.trim();
			}

			if (endDateTime) {
				payload.scheduled_end = endDateTime.toISOString();
			}

			if (imageUrl.trim()) {
				payload.image_url = imageUrl.trim();
			}

			const event = await api.post<{
				id: string;
				server_id: string;
				channel_id: string | null;
				creator_id: string;
				name: string;
				description: string;
				image_url: string | null;
				scheduled_start: string;
				scheduled_end: string | null;
				entity_type: number;
				location: string;
				status: number;
				user_count: number;
				created_at: string;
			}>(`/servers/${serverId}/events`, payload);

			dispatch('created', event);
		} catch (err: unknown) {
			error = err instanceof Error ? err.message : 'Failed to create event';
		} finally {
			submitting = false;
		}
	}

	function handleClose() {
		dispatch('close');
	}

	function setQuickDate(daysFromNow: number) {
		const date = new Date();
		date.setDate(date.getDate() + daysFromNow);
		scheduledDate = date.toISOString().split('T')[0];
		scheduledTime = '18:00';
	}

	$: if (serverId) {
		loadChannels();
	}

	$: requiresChannel = entityType === 1 || entityType === 2;
	$: requiresLocation = entityType === 3;
</script>

<Modal open={true} title="Create Event" size="large" on:close={handleClose}>
	<div class="create-form">
		{#if error}
			<div class="error-banner">{error}</div>
		{/if}

		<div class="form-section">
			<label class="section-label">EVENT TYPE</label>
			<div class="entity-type-grid">
				{#each entityTypes as type}
					<button
						class="entity-type-btn"
						class:selected={entityType === type.value}
						on:click={() => (entityType = type.value)}
						type="button"
					>
						<span class="type-icon">{type.icon}</span>
						<span class="type-label">{type.label}</span>
						<span class="type-desc">{type.desc}</span>
					</button>
				{/each}
			</div>
		</div>

		<div class="form-group">
			<label for="event-name">EVENT NAME</label>
			<input
				id="event-name"
				type="text"
				bind:value={name}
				placeholder="Event title"
				maxlength="100"
			/>
			<span class="char-count">{name.length}/100</span>
		</div>

		<div class="form-group">
			<label for="event-desc">DESCRIPTION <span class="optional">(optional)</span></label>
			<textarea
				id="event-desc"
				bind:value={description}
				placeholder="What's this event about?"
				rows="3"
				maxlength="1000"
			></textarea>
		</div>

		<div class="form-row">
			<div class="form-group">
				<label for="start-date">START DATE</label>
				<input
					id="start-date"
					type="date"
					bind:value={scheduledDate}
					min={new Date().toISOString().split('T')[0]}
				/>
				<div class="quick-dates">
					<button type="button" on:click={() => setQuickDate(1)}>Tomorrow</button>
					<button type="button" on:click={() => setQuickDate(3)}>In 3 days</button>
					<button type="button" on:click={() => setQuickDate(7)}>In a week</button>
				</div>
			</div>
			<div class="form-group">
				<label for="start-time">START TIME</label>
				<input
					id="start-time"
					type="time"
					bind:value={scheduledTime}
				/>
			</div>
		</div>

		<div class="form-row">
			<div class="form-group">
				<label for="end-date">END DATE <span class="optional">(optional)</span></label>
				<input
					id="end-date"
					type="date"
					bind:value={scheduledEndDate}
					min={scheduledDate || new Date().toISOString().split('T')[0]}
				/>
			</div>
			<div class="form-group">
				<label for="end-time">END TIME <span class="optional">(optional)</span></label>
				<input
					id="end-time"
					type="time"
					bind:value={scheduledEndTime}
				/>
			</div>
		</div>

		{#if requiresChannel}
			<div class="form-group">
				<label for="channel">
					{entityType === 1 ? 'STAGE CHANNEL' : 'VOICE CHANNEL'}
				</label>
				{#if loadingChannels}
					<div class="loading-text">Loading channels...</div>
				{:else if channels.length === 0}
					<div class="no-channels">
						No {entityType === 1 ? 'stage' : 'voice'} channels available.
						Create one first.
					</div>
				{:else}
					<select id="channel" bind:value={channelId}>
						<option value={null}>Select a channel...</option>
						{#each channels as channel}
							<option value={channel.id}>{channel.name}</option>
						{/each}
					</select>
				{/if}
			</div>
		{/if}

		{#if requiresLocation}
			<div class="form-group">
				<label for="location">LOCATION</label>
				<input
					id="location"
					type="text"
					bind:value={location}
					placeholder="Where will this event take place?"
					maxlength="100"
				/>
			</div>
		{/if}

		<div class="form-group">
			<label for="image-url">COVER IMAGE URL <span class="optional">(optional)</span></label>
			<input
				id="image-url"
				type="url"
				bind:value={imageUrl}
				placeholder="https://example.com/image.jpg"
			/>
			{#if imageUrl}
				<div class="image-preview">
					<img src={imageUrl} alt="Cover preview" on:error={() => (imageUrl = '')} />
				</div>
			{/if}
		</div>

		<div class="form-actions">
			<button
				class="btn secondary"
				on:click={handleClose}
				disabled={submitting}
				type="button"
			>
				Cancel
			</button>
			<button
				class="btn primary"
				on:click={handleSubmit}
				disabled={submitting || !name.trim()}
				type="button"
			>
				{submitting ? 'Creating...' : 'Create Event'}
			</button>
		</div>
	</div>
</Modal>

<style>
	.create-form {
		display: flex;
		flex-direction: column;
		gap: 16px;
	}

	.error-banner {
		padding: 10px 12px;
		background: rgba(218, 55, 60, 0.1);
		border: 1px solid rgba(218, 55, 60, 0.3);
		border-radius: 4px;
		color: var(--status-danger, #da373c);
		font-size: 13px;
	}

	.form-section {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	.section-label {
		font-size: 12px;
		font-weight: 700;
		color: var(--text-muted, #b5bac1);
		letter-spacing: 0.02em;
		text-transform: uppercase;
	}

	.entity-type-grid {
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		gap: 8px;
	}

	.entity-type-btn {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 4px;
		padding: 12px 8px;
		background: var(--bg-secondary, #2b2d31);
		border: 2px solid transparent;
		border-radius: 8px;
		cursor: pointer;
		transition: border-color 0.15s, background-color 0.15s;
	}

	.entity-type-btn:hover {
		background: var(--bg-modifier-hover, #35373c);
	}

	.entity-type-btn.selected {
		border-color: var(--brand-primary, #5865f2);
		background: rgba(88, 101, 242, 0.1);
	}

	.type-icon {
		font-size: 24px;
	}

	.type-label {
		font-size: 13px;
		font-weight: 600;
		color: var(--text-normal, #f2f3f5);
	}

	.type-desc {
		font-size: 11px;
		color: var(--text-faint, #6d6f78);
		text-align: center;
	}

	.form-group {
		display: flex;
		flex-direction: column;
		gap: 8px;
		position: relative;
	}

	.form-group label {
		font-size: 12px;
		font-weight: 700;
		color: var(--text-muted, #b5bac1);
		letter-spacing: 0.02em;
		text-transform: uppercase;
	}

	.optional {
		font-weight: 400;
		text-transform: none;
		letter-spacing: normal;
		color: var(--text-faint, #6d6f78);
	}

	.char-count {
		position: absolute;
		right: 0;
		top: 0;
		font-size: 11px;
		color: var(--text-faint, #6d6f78);
	}

	input[type="text"],
	input[type="date"],
	input[type="time"],
	input[type="url"],
	select,
	textarea {
		width: 100%;
		padding: 10px;
		background: var(--bg-tertiary, #1e1f22);
		border: none;
		border-radius: 4px;
		color: var(--text-normal, #f2f3f5);
		font-size: 14px;
		font-family: inherit;
		box-sizing: border-box;
	}

	input::placeholder,
	textarea::placeholder {
		color: var(--text-faint, #6d6f78);
	}

	input:focus,
	select:focus,
	textarea:focus {
		outline: none;
		box-shadow: 0 0 0 2px var(--brand-primary, #5865f2);
	}

	textarea {
		resize: vertical;
		min-height: 60px;
	}

	select {
		cursor: pointer;
		appearance: none;
		background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 12 12'%3E%3Cpath fill='%23b5bac1' d='M6 8L1 3h10z'/%3E%3C/svg%3E");
		background-repeat: no-repeat;
		background-position: right 12px center;
		padding-right: 32px;
	}

	.quick-dates {
		display: flex;
		gap: 8px;
		margin-top: 4px;
	}

	.quick-dates button {
		padding: 4px 8px;
		background: var(--bg-secondary, #2b2d31);
		border: none;
		border-radius: 4px;
		color: var(--text-muted, #b5bac1);
		font-size: 11px;
		cursor: pointer;
		transition: background-color 0.15s;
	}

	.quick-dates button:hover {
		background: var(--bg-modifier-hover, #35373c);
		color: var(--text-normal, #f2f3f5);
	}

	.loading-text {
		font-size: 13px;
		color: var(--text-muted, #b5bac1);
		padding: 8px 0;
	}

	.no-channels {
		font-size: 13px;
		color: var(--text-faint, #6d6f78);
		padding: 8px 0;
	}

	.image-preview {
		margin-top: 8px;
		max-height: 120px;
		border-radius: 8px;
		overflow: hidden;
	}

	.image-preview img {
		width: 100%;
		height: auto;
		object-fit: cover;
	}

	.form-row {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 12px;
	}

	.form-actions {
		display: flex;
		justify-content: flex-end;
		gap: 8px;
		margin-top: 8px;
	}

	.btn {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		gap: 6px;
		min-width: 96px;
		min-height: 38px;
		padding: 8px 16px;
		border-radius: 3px;
		font-size: 14px;
		font-weight: 500;
		cursor: pointer;
		border: none;
		transition: background-color 0.1s ease;
	}

	.btn.primary {
		background: var(--blurple, #5865f2);
		color: white;
	}

	.btn.primary:hover:not(:disabled) {
		background: var(--blurple-hover, #4752c4);
	}

	.btn.secondary {
		background: var(--bg-secondary, #2b2d31);
		color: var(--text-normal, #f2f3f5);
	}

	.btn.secondary:hover:not(:disabled) {
		background: var(--bg-modifier-hover, #35373c);
	}

	.btn:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}
</style>
