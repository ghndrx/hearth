import { writable, derived, get } from 'svelte/store';
import { browser } from '$app/environment';
import { API_BASE, fetchApi } from '$lib/api';
import { onGatewayEvent, gateway } from '$lib/stores/gateway';

// Stream types
export type StreamType = 1 | 2 | 3; // 1=Screen, 2=Application, 3=Camera
export type StreamQuality = 1 | 2 | 3; // 1=480p, 2=720p, 3=1080p

export interface Stream {
	id: string;
	channel_id: string;
	server_id: string;
	streamer_id: string;
	streamer?: {
		id: string;
		username: string;
		display_name?: string;
		avatar?: string;
	};
	type: StreamType;
	quality: StreamQuality;
	status: 1 | 2; // 1=Active, 2=Ended
	viewer_count: number;
	viewers: string[];
	started_at: string;
	ended_at?: string;
}

export interface StreamState {
	isStreaming: boolean;
	isViewing: boolean;
	currentStream: Stream | null;
	activeStreamInChannel: Stream | null;
	viewerCount: number;
	loading: boolean;
	error: string | null;
}

const initialState: StreamState = {
	isStreaming: false,
	isViewing: false,
	currentStream: null,
	activeStreamInChannel: null,
	viewerCount: 0,
	loading: false,
	error: null
};

function createStreamStore() {
	const { subscribe, set, update } = writable<StreamState>(initialState);

	return {
		subscribe,
		set,
		update,

		// Set the active stream for a channel (when user is not streaming)
		setActiveStream(channelId: string, stream: Stream | null) {
			update(state => ({
				...state,
				activeStreamInChannel: stream
			}));
		},

		// Set current user's stream (when they are streaming)
		setCurrentStream(stream: Stream | null) {
			update(state => ({
				...state,
				currentStream: stream,
				isStreaming: stream !== null
			}));
		},

		// Set viewing state
		setViewing(isViewing: boolean) {
			update(state => ({
				...state,
				isViewing
			}));
		},

		// Update viewer count
		setViewerCount(count: number) {
			update(state => ({
				...state,
				viewerCount: count
			}));
		},

		// Set loading state
		setLoading(loading: boolean) {
			update(state => ({ ...state, loading }));
		},

		// Set error
		setError(error: string | null) {
			update(state => ({ ...state, error }));
		},

		// Reset state
		reset() {
			set(initialState);
		}
	};
}

export const streamState = createStreamStore();

// Derived stores
export const isStreaming = derived(streamState, $state => $state.isStreaming);
export const isViewing = derived(streamState, $state => $state.isViewing);
export const currentStream = derived(streamState, $state => $state.currentStream);
export const activeStreamInChannel = derived(streamState, $state => $state.activeStreamInChannel);

// Stream actions
export const streamActions = {
	// Start a live stream
	async startStream(channelId: string, streamType: StreamType, quality: StreamQuality = 2) {
		if (!browser) return;

		streamState.setLoading(true);
		streamState.setError(null);

		try {
			const response = await fetchApi(`/channels/${channelId}/stream/start`, {
				method: 'POST',
				body: JSON.stringify({
					stream_type: streamType,
					quality: quality
				})
			});

			if (!response.ok) {
				const error = await response.json().catch(() => ({ error: 'Failed to start stream' }));
				throw new Error(error.error || 'Failed to start stream');
			}

			const stream: Stream = await response.json();
			streamState.setCurrentStream(stream);
			return stream;
		} catch (error) {
			const message = error instanceof Error ? error.message : 'Failed to start stream';
			streamState.setError(message);
			throw error;
		} finally {
			streamState.setLoading(false);
		}
	},

	// Stop the current stream
	async stopStream(channelId: string) {
		if (!browser) return;

		const state = get(streamState);
		if (!state.currentStream) return;

		streamState.setLoading(true);
		streamState.setError(null);

		try {
			const response = await fetchApi(`/channels/${channelId}/stream/stop`, {
				method: 'POST'
			});

			if (!response.ok) {
				const error = await response.json().catch(() => ({ error: 'Failed to stop stream' }));
				throw new Error(error.error || 'Failed to stop stream');
			}

			streamState.setCurrentStream(null);
		} catch (error) {
			const message = error instanceof Error ? error.message : 'Failed to stop stream';
			streamState.setError(message);
			throw error;
		} finally {
			streamState.setLoading(false);
		}
	},

	// Get active stream for a channel
	async getActiveStream(channelId: string) {
		if (!browser) return;

		try {
			const response = await fetchApi(`/channels/${channelId}/stream`);

			if (response.status === 204) {
				streamState.setActiveStream(channelId, null);
				return null;
			}

			if (!response.ok) {
				throw new Error('Failed to get stream');
			}

			const stream: Stream = await response.json();
			streamState.setActiveStream(channelId, stream);
			return stream;
		} catch (error) {
			console.error('Failed to get active stream:', error);
			return null;
		}
	},

	// Join a stream as viewer
	async joinStream(streamId: string) {
		if (!browser) return;

		const state = get(streamState);
		if (state.isViewing) return;

		streamState.setLoading(true);
		streamState.setError(null);

		try {
			const response = await fetchApi(`/streams/${streamId}/join`, {
				method: 'POST'
			});

			if (!response.ok) {
				const error = await response.json().catch(() => ({ error: 'Failed to join stream' }));
				throw new Error(error.error || 'Failed to join stream');
			}

			streamState.setViewing(true);
		} catch (error) {
			const message = error instanceof Error ? error.message : 'Failed to join stream';
			streamState.setError(message);
			throw error;
		} finally {
			streamState.setLoading(false);
		}
	},

	// Leave a stream as viewer
	async leaveStream(streamId: string) {
		if (!browser) return;

		const state = get(streamState);
		if (!state.isViewing) return;

		streamState.setLoading(true);
		streamState.setError(null);

		try {
			const response = await fetchApi(`/streams/${streamId}/leave`, {
				method: 'POST'
			});

			if (!response.ok) {
				const error = await response.json().catch(() => ({ error: 'Failed to leave stream' }));
				throw new Error(error.error || 'Failed to leave stream');
			}

			streamState.setViewing(false);
		} catch (error) {
			const message = error instanceof Error ? error.message : 'Failed to leave stream';
			streamState.setError(message);
			throw error;
		} finally {
			streamState.setLoading(false);
		}
	},

	// Update stream quality
	async updateStream(streamId: string, quality: StreamQuality) {
		if (!browser) return;

		try {
			const response = await fetchApi(`/streams/${streamId}`, {
				method: 'PATCH',
				body: JSON.stringify({ quality })
			});

			if (!response.ok) {
				throw new Error('Failed to update stream');
			}

			const updatedStream: Stream = await response.json();
			streamState.setCurrentStream(updatedStream);
			return updatedStream;
		} catch (error) {
			console.error('Failed to update stream:', error);
			throw error;
		}
	},

	// Get stream info
	async getStream(streamId: string) {
		if (!browser) return null;

		try {
			const response = await fetchApi(`/streams/${streamId}`);

			if (!response.ok) {
				throw new Error('Failed to get stream');
			}

			return await response.json() as Stream;
		} catch (error) {
			console.error('Failed to get stream:', error);
			return null;
		}
	}
};

// Listen to gateway events for stream updates
if (browser) {
	// Stream started event
	onGatewayEvent('STREAM_START', (data: unknown) => {
		const event = data as {
			stream: Stream;
			channel_id: string;
			server_id: string;
		};

		streamState.setActiveStream(event.channel_id, event.stream);
	});

	// Stream ended event
	onGatewayEvent('STREAM_END', (data: unknown) => {
		const event = data as {
			stream_id: string;
			channel_id: string;
			server_id: string;
			user_id: string;
		};

		const state = get(streamState);

		// If we were viewing this stream, update state
		if (state.isViewing && state.activeStreamInChannel?.id === event.stream_id) {
			streamState.setViewing(false);
		}

		// If this is our stream, clear it
		if (state.currentStream?.id === event.stream_id) {
			streamState.setCurrentStream(null);
		}

		// Clear the active stream for this channel
		if (state.activeStreamInChannel?.channel_id === event.channel_id) {
			streamState.setActiveStream(event.channel_id, null);
		}
	});

	// Viewer joined event
	onGatewayEvent('STREAM_VIEWER_JOIN', (data: unknown) => {
		const event = data as {
			stream_id: string;
			user_id: string;
			viewer_count: number;
			viewers: string[];
			channel_id: string;
		};

		const state = get(streamState);

		// Update viewer count if this is our stream
		if (state.currentStream?.id === event.stream_id) {
			streamState.setViewerCount(event.viewer_count);
		}

		// Update active stream viewer count if it matches
		if (state.activeStreamInChannel?.id === event.stream_id) {
			streamState.update(s => ({
				...s,
				activeStreamInChannel: s.activeStreamInChannel ? {
					...s.activeStreamInChannel,
					viewer_count: event.viewer_count,
					viewers: event.viewers
				} : null
			}));
		}
	});

	// Viewer left event
	onGatewayEvent('STREAM_VIEWER_LEAVE', (data: unknown) => {
		const event = data as {
			stream_id: string;
			user_id: string;
			viewer_count: number;
			viewers: string[];
			channel_id: string;
		};

		const state = get(streamState);

		// Update viewer count if this is our stream
		if (state.currentStream?.id === event.stream_id) {
			streamState.setViewerCount(event.viewer_count);
		}

		// Update active stream viewer count if it matches
		if (state.activeStreamInChannel?.id === event.stream_id) {
			streamState.update(s => ({
				...s,
				activeStreamInChannel: s.activeStreamInChannel ? {
					...s.activeStreamInChannel,
					viewer_count: event.viewer_count,
					viewers: event.viewers
				} : null
			}));
		}
	});
}
