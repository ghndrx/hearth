import { writable } from 'svelte/store';

/**
 * Store for cross-component message input suggestions.
 * Used by components like MemberList to insert mentions into the active message input.
 */
export interface MessageInputSuggestion {
	type: 'mention';
	value: string; // e.g. `<@user-id>`
}

function createMessageInputSuggestionStore() {
	const { subscribe, set } = writable<MessageInputSuggestion | null>(null);

	return {
		subscribe,
		/**
		 * Request insertion of a mention string into the message input.
		 */
		insertMention(mention: string) {
			set({ type: 'mention', value: mention });
		},
		/**
		 * Clear any pending suggestion.
		 */
		clear() {
			set(null);
		}
	};
}

export const messageInputSuggestion = createMessageInputSuggestionStore();
