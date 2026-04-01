import { writable, get } from 'svelte/store';
import { api, ApiError } from '$lib/api';
import { gateway, onGatewayEvent } from './gateway';

// Component interaction types
export interface ComponentInteraction {
	id: string;
	user_id: string;
	channel_id: string;
	message_id: string;
	component_id: string;
	custom_id: string;
	type: string;
	values?: string[];
	created_at: string;
}

export interface ModalSubmitEvent {
	id: string;
	user_id: string;
	channel_id: string;
	message_id: string;
	modal_id: string;
	custom_id: string;
	component_id: string;
	values: Record<string, string>;
	submitted_at: string;
}

export interface ModalState {
	open: boolean;
	customId: string;
	title: string;
	modalType: 'primary' | 'danger';
	rows: Array<{
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
	}>;
}

// Modal state store
export const modalState = writable<ModalState>({
	open: false,
	customId: '',
	title: '',
	modalType: 'primary',
	rows: [],
});

// Active modals tracking (for multiple modals)
const activeModals = writable<Set<string>>(new Set());

// Subscribe to gateway events for component interactions
let unsubscribeComponentInteraction: (() => void) | null = null;
let unsubscribeModalSubmit: (() => void) | null = null;
let unsubscribeModalClose: (() => void) | null = null;

function initializeEventListeners() {
	// Listen for component interaction events
	unsubscribeComponentInteraction = onGatewayEvent('COMPONENT_INTERACTION', (data) => {
		console.log('[Components] Component interaction event:', data);
		handleComponentInteraction(data as ComponentInteraction);
	});

	// Listen for modal submit events
	unsubscribeModalSubmit = onGatewayEvent('MODAL_SUBMIT', (data) => {
		console.log('[Components] Modal submit event:', data);
		handleModalSubmit(data as ModalSubmitEvent);
	});

	// Listen for modal close events
	unsubscribeModalClose = onGatewayEvent('MODAL_CLOSE', (data) => {
		console.log('[Components] Modal close event:', data);
		const eventData = data as { custom_id: string };
		closeModal(eventData.custom_id);
	});
}

function handleComponentInteraction(data: ComponentInteraction) {
	// Handle specific interaction types
	switch (data.type) {
		case 'show_modal':
			// Modal will be shown via the API response
			break;
		case 'acknowledge':
			// Just acknowledge, no UI change needed
			break;
		case 'update_message':
			// Message was updated, trigger a refresh
			// This would typically be handled by the message store
			break;
	}
}

function handleModalSubmit(data: ModalSubmitEvent) {
	// Remove from active modals
	activeModals.update((modals) => {
		modals.delete(data.custom_id);
		return modals;
	});

	// Close modal if it's the current one
	const currentModal = get(modalState);
	if (currentModal.customId === data.custom_id) {
		modalState.set({
			open: false,
			customId: '',
			title: '',
			modalType: 'primary',
			rows: [],
		});
	}
}

// Open a modal
export function openModal(modal: ModalState) {
	modalState.set({
		open: true,
		customId: modal.customId,
		title: modal.title,
		modalType: modal.modalType,
		rows: modal.rows,
	});

	activeModals.update((modals) => {
		modals.add(modal.customId);
		return modals;
	});
}

// Close a modal
export function closeModal(customId?: string) {
	if (customId) {
		activeModals.update((modals) => {
			modals.delete(customId);
			return modals;
		});
	}

	const currentModal = get(modalState);
	if (!customId || currentModal.customId === customId) {
		modalState.set({
			open: false,
			customId: '',
			title: '',
			modalType: 'primary',
			rows: [],
		});
	}
}

// Submit a modal
export async function submitModal(customId: string, values: Record<string, string>): Promise<void> {
	try {
		await api.post('/api/v1/interactions/modals/submit', {
			custom_id: customId,
			values,
		});
		closeModal(customId);
	} catch (error) {
		console.error('[Components] Failed to submit modal:', error);
		throw error;
	}
}

// Send component interaction
export async function sendComponentInteraction(
	channelId: string,
	messageId: string,
	componentId: string,
	customId: string,
	values: string[] = []
): Promise<{ type: string; data?: unknown }> {
	try {
		const response = await api.post<{ type: string; data?: unknown }>('/api/v1/interactions/components', {
			channel_id: channelId,
			message_id: messageId,
			component_id: componentId,
			custom_id: customId,
			values,
		});

		// Check if we need to show a modal
		if (response.type === 'show_modal' && response.data) {
			const modalData = response.data as {
				custom_id: string;
				title: string;
				type: 'primary' | 'danger';
				rows: ModalState['rows'];
			};
			openModal({
				open: true,
				customId: modalData.custom_id,
				title: modalData.title,
				modalType: modalData.type,
				rows: modalData.rows,
			});
		}

		return response;
	} catch (error) {
		console.error('[Components] Failed to send component interaction:', error);
		throw error;
	}
}

// Initialize event listeners
export function initializeComponents() {
	initializeEventListeners();
}

// Cleanup function
export function cleanupComponents() {
	if (unsubscribeComponentInteraction) {
		unsubscribeComponentInteraction();
		unsubscribeComponentInteraction = null;
	}
	if (unsubscribeModalSubmit) {
		unsubscribeModalSubmit();
		unsubscribeModalSubmit = null;
	}
	if (unsubscribeModalClose) {
		unsubscribeModalClose();
		unsubscribeModalClose = null;
	}
}
