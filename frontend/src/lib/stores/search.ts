import { writable, derived, get } from 'svelte/store';
import { api } from '$lib/api';

export interface SearchResult {
	id: string;
	channel_id: string;
	guild_id?: string;
	author: {
		id: string;
		username: string;
		display_name?: string | null;
		avatar?: string | null;
	} | null;
	content: string;
	timestamp: string;
	edited_timestamp?: string | null;
	attachments?: { id: string; filename: string; url: string }[];
	pinned: boolean;
}

export interface SearchResponse {
	messages: SearchResult[];
	total_count: number;
	has_more: boolean;
}

export interface SearchFilters {
	query: string;
	guild_id?: string;
	channel_id?: string;
	author_id?: string;
	before?: string;
	after?: string;
	has_attachments?: boolean;
	pinned?: boolean;
}

interface SearchState {
	isOpen: boolean;
	filters: SearchFilters;
	results: SearchResult[];
	totalCount: number;
	hasMore: boolean;
	loading: boolean;
	error: string | null;
	offset: number;
}

const initialState: SearchState = {
	isOpen: false,
	filters: { query: '' },
	results: [],
	totalCount: 0,
	hasMore: false,
	loading: false,
	error: null,
	offset: 0,
};

function createSearchStore() {
	const { subscribe, set, update } = writable<SearchState>(initialState);

	return {
		subscribe,

		open(serverId?: string, channelId?: string) {
			update(state => ({
				...state,
				isOpen: true,
				filters: {
					...state.filters,
					guild_id: serverId,
					channel_id: channelId,
				},
			}));
		},

		close() {
			update(state => ({
				...state,
				isOpen: false,
			}));
		},

		setFilters(filters: Partial<SearchFilters>) {
			update(state => ({
				...state,
				filters: { ...state.filters, ...filters },
				results: [], // Clear results when filters change
				offset: 0,
				totalCount: 0,
				hasMore: false,
			}));
		},

		async search(append = false) {
			const state = get({ subscribe });
			
			if (!state.filters.query.trim()) {
				update(s => ({ ...s, results: [], totalCount: 0, hasMore: false, error: null }));
				return;
			}

			update(s => ({ ...s, loading: true, error: null }));

			try {
				const params = new URLSearchParams();
				params.set('q', state.filters.query);
				params.set('limit', '25');
				params.set('offset', append ? String(state.offset) : '0');

				if (state.filters.guild_id) {
					params.set('guild_id', state.filters.guild_id);
				}
				if (state.filters.channel_id) {
					params.set('channel_id', state.filters.channel_id);
				}
				if (state.filters.author_id) {
					params.set('author_id', state.filters.author_id);
				}
				if (state.filters.before) {
					params.set('before', state.filters.before);
				}
				if (state.filters.after) {
					params.set('after', state.filters.after);
				}
				if (state.filters.has_attachments !== undefined) {
					params.set('has_attachments', String(state.filters.has_attachments));
				}
				if (state.filters.pinned !== undefined) {
					params.set('pinned', String(state.filters.pinned));
				}

				const response = await api.get<SearchResponse>(`/search/messages?${params}`);

				update(s => ({
					...s,
					results: append ? [...s.results, ...response.messages] : response.messages,
					totalCount: response.total_count,
					hasMore: response.has_more,
					offset: append ? s.offset + response.messages.length : response.messages.length,
					loading: false,
				}));
			} catch (error) {
				console.error('Search failed:', error);
				update(s => ({
					...s,
					loading: false,
					error: error instanceof Error ? error.message : 'Search failed',
				}));
			}
		},

		loadMore() {
			return this.search(true);
		},

		clear() {
			set(initialState);
		},

		reset() {
			update(state => ({
				...initialState,
				isOpen: state.isOpen,
				filters: { query: '', guild_id: state.filters.guild_id },
			}));
		},
	};
}

export const searchStore = createSearchStore();

// Derived stores for convenience
export const isSearchOpen = derived(searchStore, $s => $s.isOpen);
export const searchResults = derived(searchStore, $s => $s.results);
export const searchLoading = derived(searchStore, $s => $s.loading);
export const searchError = derived(searchStore, $s => $s.error);
export const searchTotalCount = derived(searchStore, $s => $s.totalCount);
export const searchHasMore = derived(searchStore, $s => $s.hasMore);
