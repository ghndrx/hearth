import { describe, it, expect, beforeEach, vi } from 'vitest';
import { get } from 'svelte/store';

// Use vi.hoisted so mock variables are available in vi.mock factories (which are hoisted)
const { mockApi } = vi.hoisted(() => {
	const api = {
		get: vi.fn(),
		post: vi.fn()
	};
	return { mockApi: api };
});

vi.mock('$lib/api', () => ({
	api: mockApi
}));

import {
	unreadStore,
	unreadChannels,
	fetchUnreadState,
	markChannelRead,
	updateChannelUnread,
	type UnreadState,
	type UnreadChannel
} from './unread';

// ---------- Factory ----------

function makeUnreadChannel(overrides: Partial<UnreadChannel> = {}): UnreadChannel {
	return {
		channel_id: 'ch-1',
		has_unread: false,
		unread_count: 0,
		mention_count: 0,
		...overrides
	};
}

function makeUnreadState(overrides: Partial<UnreadState> = {}): UnreadState {
	return {
		total_unread: 0,
		total_mentions: 0,
		channels: [],
		...overrides
	};
}

// ---------- Tests ----------

describe('Unread Store', () => {
	beforeEach(() => {
		unreadStore.set(makeUnreadState());
		vi.clearAllMocks();
	});

	// ---------- fetchUnreadState ----------

	describe('fetchUnreadState', () => {
		it('should fetch unread state from API and update store', async () => {
			const apiState = makeUnreadState({
				total_unread: 10,
				total_mentions: 3,
				channels: [
					makeUnreadChannel({ channel_id: 'ch-1', has_unread: true, unread_count: 5, mention_count: 2 }),
					makeUnreadChannel({ channel_id: 'ch-2', has_unread: true, unread_count: 5, mention_count: 1 })
				]
			});
			mockApi.get.mockResolvedValueOnce(apiState);

			await fetchUnreadState();

			const state = get(unreadStore);
			expect(state.total_unread).toBe(10);
			expect(state.total_mentions).toBe(3);
			expect(state.channels).toHaveLength(2);
			expect(mockApi.get).toHaveBeenCalledWith('/users/@me/unread');
		});

		it('should update unreadChannels derived store when fetch succeeds', async () => {
			const apiState = makeUnreadState({
				total_unread: 1,
				channels: [
					makeUnreadChannel({ channel_id: 'ch-1', has_unread: true })
				]
			});
			mockApi.get.mockResolvedValueOnce(apiState);

			await fetchUnreadState();

			const unread = get(unreadChannels);
			expect(unread.has('ch-1')).toBe(true);
		});

		it('should log error on API failure', async () => {
			mockApi.get.mockRejectedValueOnce(new Error('Network error'));
			const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});

			await fetchUnreadState();

			expect(consoleSpy).toHaveBeenCalledWith('Failed to fetch unread state:', expect.any(Error));
			consoleSpy.mockRestore();
		});
	});

	// ---------- markChannelRead ----------

	describe('markChannelRead', () => {
		it('should call API to acknowledge channel', async () => {
			mockApi.post.mockResolvedValueOnce(undefined);

			await markChannelRead('ch-1');

			expect(mockApi.post).toHaveBeenCalledWith('/channels/ch-1/ack');
		});

		it('should update unreadChannels when channel is marked as read', async () => {
			// Pre-populate via fetchUnreadState which properly syncs both stores
			const apiState = makeUnreadState({
				total_unread: 1,
				channels: [
					makeUnreadChannel({ channel_id: 'ch-1', has_unread: true, unread_count: 1 })
				]
			});
			mockApi.get.mockResolvedValueOnce(apiState);
			await fetchUnreadState();

			// Verify unread
			expect(get(unreadChannels).has('ch-1')).toBe(true);

			mockApi.post.mockResolvedValueOnce(undefined);
			await markChannelRead('ch-1');

			// The store should now not have ch-1 as unread
			const unread = get(unreadChannels);
			expect(unread.has('ch-1')).toBe(false);
		});

		it('should throw on API failure', async () => {
			mockApi.post.mockRejectedValueOnce(new Error('API error'));

			await expect(markChannelRead('ch-1')).rejects.toThrow('API error');
		});
	});

	// ---------- updateChannelUnread ----------

	describe('updateChannelUnread', () => {
		it('should not throw when updating channel state', () => {
			expect(() => updateChannelUnread('ch-1', true, 5, 2)).not.toThrow();
		});

		it('should handle updating same channel multiple times', () => {
			expect(() => {
				updateChannelUnread('ch-1', true, 5, 2);
				updateChannelUnread('ch-1', false, 0, 0);
				updateChannelUnread('ch-1', true, 3, 1);
			}).not.toThrow();
		});

		it('should update unreadChannels when a channel becomes unread', async () => {
			// Pre-populate via fetchUnreadState
			const apiState = makeUnreadState({ channels: [] });
			mockApi.get.mockResolvedValueOnce(apiState);
			await fetchUnreadState();

			expect(get(unreadChannels).has('ch-1')).toBe(false);

			// Update to make ch-1 unread
			updateChannelUnread('ch-1', true, 5, 2);

			expect(get(unreadChannels).has('ch-1')).toBe(true);
		});

		it('should update unreadChannels when a channel becomes read', async () => {
			// Pre-populate via fetchUnreadState
			const apiState = makeUnreadState({
				channels: [
					makeUnreadChannel({ channel_id: 'ch-1', has_unread: true, unread_count: 5 })
				]
			});
			mockApi.get.mockResolvedValueOnce(apiState);
			await fetchUnreadState();

			expect(get(unreadChannels).has('ch-1')).toBe(true);

			// Update to mark as read
			updateChannelUnread('ch-1', false, 0, 0);

			expect(get(unreadChannels).has('ch-1')).toBe(false);
		});
	});

	// ---------- unreadChannels derived store ----------

	describe('unreadChannels', () => {
		it('should return empty set when no unread channels', async () => {
			const apiState = makeUnreadState({
				channels: [
					makeUnreadChannel({ channel_id: 'ch-1', has_unread: false }),
					makeUnreadChannel({ channel_id: 'ch-2', has_unread: false })
				]
			});
			mockApi.get.mockResolvedValueOnce(apiState);

			await fetchUnreadState();

			const unread = get(unreadChannels);
			expect(unread.size).toBe(0);
		});

		it('should return set of channel IDs with unread messages', async () => {
			const apiState = makeUnreadState({
				total_unread: 3,
				channels: [
					makeUnreadChannel({ channel_id: 'ch-1', has_unread: true, unread_count: 2 }),
					makeUnreadChannel({ channel_id: 'ch-2', has_unread: false }),
					makeUnreadChannel({ channel_id: 'ch-3', has_unread: true, unread_count: 1 })
				]
			});
			mockApi.get.mockResolvedValueOnce(apiState);

			await fetchUnreadState();

			const unread = get(unreadChannels);
			expect(unread.size).toBe(2);
			expect(unread.has('ch-1')).toBe(true);
			expect(unread.has('ch-2')).toBe(false);
			expect(unread.has('ch-3')).toBe(true);
		});
	});
});
