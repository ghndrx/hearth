import { writable, derived } from 'svelte/store';
import { api } from '$lib/api';

export interface ForumTag {
	id: string;
	server_id: string;
	channel_id: string;
	name: string;
	color: string | null;
	emoji_name: string | null;
	moderated: boolean;
	created_at: string;
}

export interface ForumTagsState {
	tagsByChannel: Record<string, ForumTag[]>;
	threadTags: Record<string, ForumTag[]>; // threadId -> tags
	loading: boolean;
	error: string | null;
}

function createForumTagsStore() {
	const { subscribe, set, update } = writable<ForumTagsState>({
		tagsByChannel: {},
		threadTags: {},
		loading: false,
		error: null
	});

	return {
		subscribe,

		/**
		 * Load tags for a channel
		 */
		async loadChannelTags(channelId: string): Promise<ForumTag[]> {
			update(s => ({ ...s, loading: true, error: null }));
			try {
				const response = await api.get<{ tags: ForumTag[] }>(`/channels/${channelId}/tags`);
				const tags = response.tags || [];
				update(s => ({
					...s,
					tagsByChannel: { ...s.tagsByChannel, [channelId]: tags },
					loading: false
				}));
				return tags;
			} catch (err) {
				const error = err instanceof Error ? err.message : 'Failed to load tags';
				update(s => ({ ...s, error, loading: false }));
				return [];
			}
		},

		/**
		 * Get tags for a channel from cache
		 */
		getChannelTags(state: ForumTagsState, channelId: string): ForumTag[] {
			return state.tagsByChannel[channelId] || [];
		},

		/**
		 * Create a new tag for a channel
		 */
		async createTag(
			channelId: string,
			data: { name: string; color?: string; emoji_name?: string | null; moderated?: boolean }
		): Promise<ForumTag | null> {
			update(s => ({ ...s, loading: true, error: null }));
			try {
				const tag = await api.post<ForumTag>(`/channels/${channelId}/tags`, data);
				update(s => ({
					...s,
					tagsByChannel: {
						...s.tagsByChannel,
						[channelId]: [...(s.tagsByChannel[channelId] || []), tag]
					},
					loading: false
				}));
				return tag;
			} catch (err) {
				const error = err instanceof Error ? err.message : 'Failed to create tag';
				update(s => ({ ...s, error, loading: false }));
				return null;
			}
		},

		/**
		 * Update a tag
		 */
		async updateTag(
			tagId: string,
			channelId: string,
			data: { name?: string; color?: string; emoji_name?: string | null; moderated?: boolean }
		): Promise<ForumTag | null> {
			update(s => ({ ...s, loading: true, error: null }));
			try {
				const tag = await api.patch<ForumTag>(`/forum-tags/${tagId}`, data);
				update(s => ({
					...s,
					tagsByChannel: {
						...s.tagsByChannel,
						[channelId]: (s.tagsByChannel[channelId] || []).map(t =>
							t.id === tagId ? tag : t
						)
					},
					loading: false
				}));
				return tag;
			} catch (err) {
				const error = err instanceof Error ? err.message : 'Failed to update tag';
				update(s => ({ ...s, error, loading: false }));
				return null;
			}
		},

		/**
		 * Delete a tag
		 */
		async deleteTag(tagId: string, channelId: string): Promise<boolean> {
			update(s => ({ ...s, loading: true, error: null }));
			try {
				await api.delete(`/forum-tags/${tagId}`);
				update(s => ({
					...s,
					tagsByChannel: {
						...s.tagsByChannel,
						[channelId]: (s.tagsByChannel[channelId] || []).filter(t => t.id !== tagId)
					},
					loading: false
				}));
				return true;
			} catch (err) {
				const error = err instanceof Error ? err.message : 'Failed to delete tag';
				update(s => ({ ...s, error, loading: false }));
				return false;
			}
		},

		/**
		 * Apply tags to a thread/post
		 */
		async applyTagsToThread(
			threadId: string,
			channelId: string,
			tagIds: string[]
		): Promise<boolean> {
			update(s => ({ ...s, loading: true, error: null }));
			try {
				await api.put(`/threads/${threadId}/tags`, { tag_ids: tagIds });
				// After applying tags, reload the thread's tags
				const tags = await api.get<ForumTag[]>(`/threads/${threadId}/tags`);
				update(s => ({
					...s,
					threadTags: { ...s.threadTags, [threadId]: tags },
					loading: false
				}));
				return true;
			} catch (err) {
				const error = err instanceof Error ? err.message : 'Failed to apply tags';
				update(s => ({ ...s, error, loading: false }));
				return false;
			}
		},

		/**
		 * Load tags for a thread
		 */
		async loadThreadTags(threadId: string): Promise<ForumTag[]> {
			update(s => ({ ...s, loading: true, error: null }));
			try {
				const response = await api.get<{ tags: ForumTag[] }>(`/threads/${threadId}/tags`);
				const tags = response.tags || [];
				update(s => ({
					...s,
					threadTags: { ...s.threadTags, [threadId]: tags },
					loading: false
				}));
				return tags;
			} catch (err) {
				const error = err instanceof Error ? err.message : 'Failed to load thread tags';
				update(s => ({ ...s, error, loading: false }));
				return [];
			}
		},

		/**
		 * Get thread tags from cache
		 */
		getThreadTags(state: ForumTagsState, threadId: string): ForumTag[] {
			return state.threadTags[threadId] || [];
		},

		/**
		 * Clear all cached data
		 */
		clear() {
			set({
				tagsByChannel: {},
				threadTags: {},
				loading: false,
				error: null
			});
		}
	};
}

export const forumTags = createForumTagsStore();

// Functional API re-exports (used by ForumTagEditor)
export const forumTagsByChannel = derived(
	forumTags,
	$s => $s.tagsByChannel
);
export const forumTagsLoading = derived(
	forumTags,
	$s => $s.loading
);

export const loadChannelTags = (channelId: string) =>
	forumTags.loadChannelTags(channelId);
export const createForumTag = (
	channelId: string,
	data: { name: string; color?: string; emoji_name?: string | null; moderated?: boolean }
) => forumTags.createTag(channelId, data);
export const updateForumTag = (
	tagId: string,
	channelId: string,
	data: { name?: string; color?: string; emoji_name?: string | null; moderated?: boolean }
) => forumTags.updateTag(tagId, channelId, data);
export const deleteForumTag = (tagId: string, channelId: string) =>
	forumTags.deleteTag(tagId, channelId);
