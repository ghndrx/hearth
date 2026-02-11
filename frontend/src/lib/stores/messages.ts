import { writable, get } from 'svelte/store';
import type { Message } from '$lib/types';
import { auth } from './auth';
import { servers } from './servers';

function createMessagesStore() {
	const { subscribe, update } = writable<{
		messages: Map<string, Message[]>;
		loading: boolean;
		hasMore: Map<string, boolean>;
	}>({
		messages: new Map(),
		loading: false,
		hasMore: new Map()
	});

	async function fetchWithAuth(url: string, options: RequestInit = {}) {
		const token = get(auth).token;
		return fetch(url, {
			...options,
			headers: {
				...options.headers,
				Authorization: `Bearer ${token}`,
				'Content-Type': 'application/json'
			}
		});
	}

	return {
		subscribe,

		loadMessages: async (channelId: string, before?: string) => {
			update((s) => ({ ...s, loading: true }));
			try {
				const url = before
					? `/api/channels/${channelId}/messages?before=${before}&limit=50`
					: `/api/channels/${channelId}/messages?limit=50`;

				const res = await fetchWithAuth(url);
				if (res.ok) {
					const newMessages: Message[] = await res.json();
					update((s) => {
						const allMessages = new Map(s.messages);
						const existing = allMessages.get(channelId) || [];

						if (before) {
							// Prepend older messages
							allMessages.set(channelId, [...newMessages, ...existing]);
						} else {
							// Initial load
							allMessages.set(channelId, newMessages);
						}

						const hasMore = new Map(s.hasMore);
						hasMore.set(channelId, newMessages.length === 50);

						return { ...s, messages: allMessages, hasMore, loading: false };
					});
				}
			} catch (e) {
				console.error('Failed to load messages:', e);
				update((s) => ({ ...s, loading: false }));
			}
		},

		sendMessage: async (channelId: string, content: string) => {
			try {
				const res = await fetchWithAuth(`/api/channels/${channelId}/messages`, {
					method: 'POST',
					body: JSON.stringify({ content })
				});
				if (res.ok) {
					const message: Message = await res.json();
					// Message will be added via WebSocket event
					return message;
				}
			} catch (e) {
				console.error('Failed to send message:', e);
			}
			return null;
		},

		addMessage: (message: Message) => {
			update((s) => {
				const allMessages = new Map(s.messages);
				const channelMessages = allMessages.get(message.channelId) || [];
				
				// Check for duplicate
				if (channelMessages.some((m) => m.id === message.id)) {
					return s;
				}
				
				allMessages.set(message.channelId, [...channelMessages, message]);
				return { ...s, messages: allMessages };
			});
		},

		updateMessage: (message: Message) => {
			update((s) => {
				const allMessages = new Map(s.messages);
				const channelMessages = allMessages.get(message.channelId) || [];
				allMessages.set(
					message.channelId,
					channelMessages.map((m) => (m.id === message.id ? message : m))
				);
				return { ...s, messages: allMessages };
			});
		},

		deleteMessage: (messageId: string, channelId: string) => {
			update((s) => {
				const allMessages = new Map(s.messages);
				const channelMessages = allMessages.get(channelId) || [];
				allMessages.set(
					channelId,
					channelMessages.filter((m) => m.id !== messageId)
				);
				return { ...s, messages: allMessages };
			});
		},

		getChannelMessages: (channelId: string): Message[] => {
			let result: Message[] = [];
			const unsubscribe = subscribe((s) => {
				result = s.messages.get(channelId) || [];
			});
			unsubscribe();
			return result;
		},

		reset: () => {
			update(() => ({
				messages: new Map(),
				loading: false,
				hasMore: new Map()
			}));
		}
	};
}

export const messages = createMessagesStore();
