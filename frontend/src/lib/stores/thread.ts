import { writable, get } from 'svelte/store';
import { api } from '$lib/api';
import type { Message } from './messages';

export interface Thread {
	id: string;
	channel_id: string;
	parent_message_id: string;
	parent_message?: Message;
	name?: string;
	message_count: number;
	member_count?: number;
	created_at: string;
	last_message_at?: string;
}

export interface ThreadMessage extends Message {
	thread_id: string;
}

interface ThreadState {
	currentThread: Thread | null;
	messages: ThreadMessage[];
	loading: boolean;
	sending: boolean;
}

function createThreadStore() {
	const { subscribe, set, update } = writable<ThreadState>({
		currentThread: null,
		messages: [],
		loading: false,
		sending: false
	});

	return {
		subscribe,

		// Open a thread panel for a specific message
		async open(parentMessage: Message, channelId: string) {
			update(state => ({ ...state, loading: true, messages: [] }));

			// Create a thread reference from the parent message
			const thread: Thread = {
				id: parentMessage.id, // Use message ID as thread ID for now
				channel_id: channelId,
				parent_message_id: parentMessage.id,
				parent_message: parentMessage,
				name: `Thread - ${parentMessage.content.slice(0, 30)}${parentMessage.content.length > 30 ? '...' : ''}`,
				message_count: 0,
				created_at: parentMessage.created_at
			};

			update(state => ({ ...state, currentThread: thread }));

			try {
				// Try to load thread messages from API
				// The API might use /channels/{channel_id}/messages/{message_id}/threads
				// or /threads/{thread_id}/messages
				const data = await api.get<ThreadMessage[]>(
					`/channels/${channelId}/messages/${parentMessage.id}/thread`
				).catch(() => []);

				update(state => ({
					...state,
					messages: Array.isArray(data) ? data : [],
					loading: false
				}));
			} catch (error) {
				console.error('Failed to load thread messages:', error);
				update(state => ({ ...state, loading: false }));
			}
		},

		// Close the thread panel
		close() {
			set({
				currentThread: null,
				messages: [],
				loading: false,
				sending: false
			});
		},

		// Send a reply in the thread
		async sendReply(content: string) {
			const state = get({ subscribe });
			if (!state.currentThread || !content.trim()) return;

			update(s => ({ ...s, sending: true }));

			try {
				const response = await api.post<ThreadMessage>(
					`/channels/${state.currentThread.channel_id}/messages`,
					{
						content: content.trim(),
						reply_to: state.currentThread.parent_message_id
					}
				);

				// Add the new message to the thread
				update(s => ({
					...s,
					messages: [...s.messages, response],
					sending: false
				}));

				return response;
			} catch (error) {
				console.error('Failed to send thread reply:', error);
				update(s => ({ ...s, sending: false }));
				throw error;
			}
		},

		// Add a message to the current thread (from WebSocket)
		addMessage(message: ThreadMessage) {
			update(state => {
				if (!state.currentThread) return state;
				if (message.reply_to !== state.currentThread.parent_message_id) return state;
				
				// Avoid duplicates
				if (state.messages.find(m => m.id === message.id)) return state;

				return {
					...state,
					messages: [...state.messages, message]
				};
			});
		},

		// Update a message in the thread
		updateMessage(message: Partial<ThreadMessage> & { id: string }) {
			update(state => ({
				...state,
				messages: state.messages.map(m =>
					m.id === message.id ? { ...m, ...message } : m
				)
			}));
		},

		// Remove a message from the thread
		removeMessage(messageId: string) {
			update(state => ({
				...state,
				messages: state.messages.filter(m => m.id !== messageId)
			}));
		},

		// Check if a thread is currently open
		isOpen(): boolean {
			return get({ subscribe }).currentThread !== null;
		}
	};
}

export const threadStore = createThreadStore();

// Derived stores for convenience
export const currentThread = {
	subscribe: (fn: (value: Thread | null) => void) => {
		return threadStore.subscribe(state => fn(state.currentThread));
	}
};

export const threadMessages = {
	subscribe: (fn: (value: ThreadMessage[]) => void) => {
		return threadStore.subscribe(state => fn(state.messages));
	}
};

export const threadLoading = {
	subscribe: (fn: (value: boolean) => void) => {
		return threadStore.subscribe(state => fn(state.loading));
	}
};
