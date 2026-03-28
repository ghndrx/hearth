import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import VideoGrid from './VideoGrid.svelte';

// Mock Avatar component for tests
vi.mock('./Avatar.svelte', () => {
	return {
		default: vi.fn().mockImplementation((props) => {
			const div = document.createElement('div');
			div.className = 'avatar-mock';
			div.setAttribute('data-username', props?.username || '');
			return div;
		})
	};
});

describe('VideoGrid', () => {
	const mockParticipants = [
		{
			id: 'user-1',
			username: 'testuser1',
			display_name: 'Test User 1',
			avatar: 'https://example.com/avatar1.png',
			isVideoEnabled: true,
			isScreenSharing: false,
			isSpeaking: false,
			isMuted: false
		},
		{
			id: 'user-2',
			username: 'testuser2',
			display_name: 'Test User 2',
			avatar: 'https://example.com/avatar2.png',
			isVideoEnabled: false,
			isScreenSharing: true,
			isSpeaking: true,
			isMuted: true
		}
	];

	describe('rendering', () => {
		it('renders empty state when no participants and no local tracks', () => {
			const { container } = render(VideoGrid, {
				props: {
					participants: [],
					localVideoTrack: null,
					localScreenShareTrack: null
				}
			});

			const emptyState = container.querySelector('.empty-state');
			expect(emptyState).toBeInTheDocument();
		});

		it('renders participant tiles when participants provided', () => {
			const { container } = render(VideoGrid, {
				props: {
					participants: mockParticipants,
					localVideoTrack: null,
					localScreenShareTrack: null
				}
			});

			const videoGrid = container.querySelector('.video-grid');
			expect(videoGrid).toBeInTheDocument();
		});

		it('renders avatar-only tile for participant without video', () => {
			const { container } = render(VideoGrid, {
				props: {
					participants: [mockParticipants[1]], // user-2 with isVideoEnabled: false
					localVideoTrack: null,
					localScreenShareTrack: null
				}
			});

			const avatarOnlyTile = container.querySelector('.video-tile.avatar-only');
			expect(avatarOnlyTile).toBeInTheDocument();
		});

		it('renders screen share tile for participant with screen sharing', () => {
			const { container } = render(VideoGrid, {
				props: {
					participants: [mockParticipants[1]], // user-2 with isScreenSharing: true
					localVideoTrack: null,
					localScreenShareTrack: null
				}
			});

			const screenShareTile = container.querySelector('.video-tile.screen-share');
			expect(screenShareTile).toBeInTheDocument();
		});
	});

	describe('local video tracks', () => {
		it('renders local video track when provided', () => {
			const localVideo = document.createElement('video');
			const { container } = render(VideoGrid, {
				props: {
					participants: [],
					localVideoTrack: localVideo,
					localScreenShareTrack: null
				}
			});

			const localVideoTile = container.querySelector('.video-tile:not(.screen-share)');
			expect(localVideoTile).toBeInTheDocument();
		});

		it('renders local screen share track when provided', () => {
			const localScreen = document.createElement('video');
			const { container } = render(VideoGrid, {
				props: {
					participants: [],
					localVideoTrack: null,
					localScreenShareTrack: localScreen
				}
			});

			const screenShareTile = container.querySelector('.video-tile.screen-share');
			expect(screenShareTile).toBeInTheDocument();
		});
	});

	describe('participant name display', () => {
		it('displays display_name when available', () => {
			const { container } = render(VideoGrid, {
				props: {
					participants: [mockParticipants[0]],
					localVideoTrack: null,
					localScreenShareTrack: null
				}
			});

			const nameElement = container.querySelector('.participant-name');
			expect(nameElement).toHaveTextContent('Test User 1');
		});

		it('falls back to username when display_name not available', () => {
			const participantWithoutDisplayName = {
				...mockParticipants[0],
				display_name: null
			};
			const { container } = render(VideoGrid, {
				props: {
					participants: [participantWithoutDisplayName],
					localVideoTrack: null,
					localScreenShareTrack: null
				}
			});

			const nameElement = container.querySelector('.participant-name');
			expect(nameElement).toHaveTextContent('testuser1');
		});

		it('shows "You (Camera)" for local camera', () => {
			const localVideo = document.createElement('video');
			const { container } = render(VideoGrid, {
				props: {
					participants: [],
					localVideoTrack: localVideo,
					localScreenShareTrack: null
				}
			});

			const nameElement = container.querySelector('.participant-name.local');
			expect(nameElement).toHaveTextContent('You (Camera)');
		});

		it('shows "You (Screen)" for local screen share', () => {
			const localScreen = document.createElement('video');
			const { container } = render(VideoGrid, {
				props: {
					participants: [],
					localVideoTrack: null,
					localScreenShareTrack: localScreen
				}
			});

			const nameElement = container.querySelector('.participant-name.local');
			expect(nameElement).toHaveTextContent('You (Screen)');
		});
	});

	describe('speaking indicator', () => {
		it('adds speaking class when participant is speaking', () => {
			const speakingParticipant = {
				...mockParticipants[0],
				isSpeaking: true
			};
			const { container } = render(VideoGrid, {
				props: {
					participants: [speakingParticipant],
					localVideoTrack: null,
					localScreenShareTrack: null
				}
			});

			const speakingTile = container.querySelector('.video-tile.speaking');
			expect(speakingTile).toBeInTheDocument();
		});

		it('shows speaking ring for avatar-only speaking participant', () => {
			const speakingParticipant = {
				...mockParticipants[1],
				isSpeaking: true,
				isVideoEnabled: false
			};
			const { container } = render(VideoGrid, {
				props: {
					participants: [speakingParticipant],
					localVideoTrack: null,
					localScreenShareTrack: null
				}
			});

			const speakingRing = container.querySelector('.speaking-ring');
			expect(speakingRing).toBeInTheDocument();
		});
	});

	describe('mute indicator', () => {
		it('shows mute overlay for muted participant without video', () => {
			const mutedParticipant = {
				...mockParticipants[1],
				isVideoEnabled: false,
				isMuted: true
			};
			const { container } = render(VideoGrid, {
				props: {
					participants: [mutedParticipant],
					localVideoTrack: null,
					localScreenShareTrack: null
				}
			});

			const muteOverlay = container.querySelector('.mute-overlay');
			expect(muteOverlay).toBeInTheDocument();
		});
	});

	describe('attachTrack method', () => {
		it('returns null when video element not found', () => {
			const { component } = render(VideoGrid, {
				props: {
					participants: [],
					localVideoTrack: null,
					localScreenShareTrack: null
				}
			});

			const mockTrack = {
				id: 'track-1',
				clone: () => mockTrack
			} as unknown as MediaStreamTrack;

			// When participant doesn't exist, attachTrack should return null
			const result = component.attachTrack('nonexistent-user', mockTrack, false);
			expect(result).toBeNull();
		});
	});

	describe('getVideoElement method', () => {
		it('returns null for nonexistent participant', () => {
			const { component } = render(VideoGrid, {
				props: {
					participants: [],
					localVideoTrack: null,
					localScreenShareTrack: null
				}
			});

			const result = component.getVideoElement('nonexistent-user', false);
			expect(result).toBeNull();
		});
	});

	describe('detachTrack method', () => {
		it('does not throw for nonexistent participant', () => {
			const { component } = render(VideoGrid, {
				props: {
					participants: [],
					localVideoTrack: null,
					localScreenShareTrack: null
				}
			});

			expect(() => component.detachTrack('nonexistent-user', false)).not.toThrow();
		});
	});
});
