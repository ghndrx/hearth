import { browser } from '$app/environment';
import { get } from 'svelte/store';
import { gateway, Op, onGatewayEvent } from '$lib/stores/gateway';
import { videoCallStore, videoCallActions } from '$lib/stores/videoCall';
import { user as authUser } from '$lib/stores/auth';

// ICE servers for STUN/TURN
const ICE_SERVERS: RTCIceServer[] = [
	{ urls: 'stun:stun.l.google.com:19302' },
	{ urls: 'stun:stun1.l.google.com:19302' },
	{ urls: 'stun:stun2.l.google.com:19302' },
];

interface VideoPeerConnection {
	connection: RTCPeerConnection;
	userId: string;
	videoTrack: MediaStreamTrack | null;
	audioTrack: MediaStreamTrack | null;
	remoteStream: MediaStream;
	remoteVideoElement: HTMLVideoElement | null;
	remoteScreenShare: boolean;
}

interface LocalVideoStream {
	stream: MediaStream;
	videoTrack: MediaStreamTrack | null;
	audioTrack: MediaStreamTrack | null;
	screenShareStream: MediaStream | null;
	screenShareTrack: MediaStreamTrack | null;
}

class VideoCallManager {
	private localStream: LocalVideoStream | null = null;
	private peers: Map<string, VideoPeerConnection> = new Map();
	private cleanupFunctions: Array<() => void> = [];
	private currentCallId: string | null = null;
	private screenShareEnabled = false;

	constructor() {
		if (!browser) return;
		this.setupGatewayListeners();
	}

	private setupGatewayListeners() {
		// Listen for video call events

		// Ring accepted - start WebRTC connection
		this.cleanupFunctions.push(
			onGatewayEvent('VIDEO_RING_ACCEPT', (data: unknown) => {
				const accept = data as { call_id: string; from_user_id: string };
				this.handleCallAccepted(accept.call_id, accept.from_user_id);
			})
		);

		// Full call state received
		this.cleanupFunctions.push(
			onGatewayEvent('VIDEO_SERVER_UPDATE', (data: unknown) => {
				const update = data as {
					call_id: string;
					state: string;
					participants: Array<{
						user_id: string;
						username: string;
						is_camera_on: boolean;
						is_muted: boolean;
						is_screen_share: boolean;
					}>;
				};
				this.handleCallStateUpdate(update);
			})
		);

		// WebRTC signaling - offer
		this.cleanupFunctions.push(
			onGatewayEvent('VIDEO_OFFER', (data: unknown) => {
				const offer = data as { call_id: string; from_user_id: string; sdp: string };
				this.handleOffer(offer.call_id, offer.from_user_id, offer.sdp);
			})
		);

		// WebRTC signaling - answer
		this.cleanupFunctions.push(
			onGatewayEvent('VIDEO_ANSWER', (data: unknown) => {
				const answer = data as { call_id: string; from_user_id: string; sdp: string };
				this.handleAnswer(answer.from_user_id, answer.sdp);
			})
		);

		// WebRTC signaling - ICE candidate
		this.cleanupFunctions.push(
			onGatewayEvent('VIDEO_ICE_CANDIDATE', (data: unknown) => {
				const candidate = data as {
					call_id: string;
					from_user_id: string;
					candidate: string;
					sdpMid: string;
					sdpMLineIndex: number;
				};
				this.handleICECandidate(candidate.from_user_id, candidate);
			})
		);

		// Participant state update
		this.cleanupFunctions.push(
			onGatewayEvent('VIDEO_STATE_UPDATE', (data: unknown) => {
				const update = data as {
					call_id: string;
					user_id: string;
					is_camera_on?: boolean;
					is_muted?: boolean;
					is_screen_share?: boolean;
				};
				this.handleParticipantStateUpdate(update);
			})
		);
	}

	// Start a video call
	async startCall(channelId: string, toUserId?: string, serverId?: string): Promise<void> {
		if (!browser) return;

		try {
			// Get user media
			await this.initializeLocalStream(true); // Start with video

			// Set call ID
			this.currentCallId = crypto.randomUUID();

			// Notify store
			videoCallActions.startCall(channelId, toUserId, serverId);

			console.log('[VideoCall] Started call in channel', channelId);
		} catch (error) {
			console.error('[VideoCall] Failed to start call:', error);
			videoCallStore.setError(error instanceof Error ? error.message : 'Failed to access camera');
			throw error;
		}
	}

	// Accept incoming call
	async acceptCall(): Promise<void> {
		if (!browser) return;

		try {
			await this.initializeLocalStream(true);
			videoCallActions.acceptCall();
			console.log('[VideoCall] Accepted call');
		} catch (error) {
			console.error('[VideoCall] Failed to accept call:', error);
			videoCallStore.setError(error instanceof Error ? error.message : 'Failed to access camera');
			throw error;
		}
	}

	// End current call
	endCall(): void {
		this.cleanup();
		videoCallActions.endCall();
		console.log('[VideoCall] Ended call');
	}

	// Initialize local media stream
	private async initializeLocalStream(withVideo: boolean): Promise<void> {
		const constraints: MediaStreamConstraints = {
			audio: {
				echoCancellation: true,
				noiseSuppression: true,
				autoGainControl: true
			},
			video: withVideo ? {
				width: { ideal: 1280 },
				height: { ideal: 720 },
				frameRate: { ideal: 30 }
			} : false
		};

		const stream = await navigator.mediaDevices.getUserMedia(constraints);
		const videoTrack = stream.getVideoTracks()[0] || null;
		const audioTrack = stream.getAudioTracks()[0] || null;

		this.localStream = {
			stream,
			videoTrack,
			audioTrack,
			screenShareStream: null,
			screenShareTrack: null
		};
	}

	// Handle incoming offer
	private async handleOffer(callId: string, fromUserId: string, sdp: string): Promise<void> {
		console.log('[VideoCall] Received offer from', fromUserId);

		// Create peer connection if doesn't exist
		if (!this.peers.has(fromUserId)) {
			await this.createPeerConnection(fromUserId, false);
		}

		const peer = this.peers.get(fromUserId);
		if (!peer) return;

		try {
			await peer.connection.setRemoteDescription({ type: 'offer', sdp });
			const answer = await peer.connection.createAnswer();
			await peer.connection.setLocalDescription(answer);

			// Send answer
			gateway.send({
				op: Op.DISPATCH,
				d: {
					t: 'VIDEO_ANSWER',
					d: {
						call_id: callId,
						to_user_id: fromUserId,
						sdp: answer.sdp
					}
				}
			});
		} catch (error) {
			console.error('[VideoCall] Failed to handle offer:', error);
		}
	}

	// Handle incoming answer
	private async handleAnswer(fromUserId: string, sdp: string): Promise<void> {
		console.log('[VideoCall] Received answer from', fromUserId);

		const peer = this.peers.get(fromUserId);
		if (!peer) return;

		try {
			await peer.connection.setRemoteDescription({ type: 'answer', sdp });
		} catch (error) {
			console.error('[VideoCall] Failed to handle answer:', error);
		}
	}

	// Handle incoming ICE candidate
	private async handleICECandidate(
		fromUserId: string,
		candidate: { candidate: string; sdpMid: string; sdpMLineIndex: number }
	): Promise<void> {
		const peer = this.peers.get(fromUserId);
		if (!peer) return;

		try {
			await peer.connection.addIceCandidate(new RTCIceCandidate(candidate));
		} catch (error) {
			console.error('[VideoCall] Failed to add ICE candidate:', error);
		}
	}

	// Handle call accepted
	private async handleCallAccepted(callId: string, toUserId: string): Promise<void> {
		console.log('[VideoCall] Call accepted, creating offer for', toUserId);

		this.currentCallId = callId;
		await this.createPeerConnection(toUserId, true);
	}

	// Handle call state update
	private handleCallStateUpdate(update: {
		call_id: string;
		state: string;
		participants: Array<{
			user_id: string;
			username: string;
			is_camera_on: boolean;
			is_muted: boolean;
			is_screen_share: boolean;
		}>;
	}): void {
		const currentUser = get(authUser);
		if (!currentUser) return;

		// Create connections to existing participants
		for (const participant of update.participants) {
			if (participant.user_id !== currentUser.id && !this.peers.has(participant.user_id)) {
				this.createPeerConnection(participant.user_id, true);
			}
		}

		// Update connection state
		if (update.state === 'connected') {
			videoCallStore.setConnectionState('connected');
		}
	}

	// Handle participant state update
	private handleParticipantStateUpdate(update: {
		user_id: string;
		is_camera_on?: boolean;
		is_muted?: boolean;
		is_screen_share?: boolean;
	}): void {
		// Update the peer's state in the store
		videoCallStore.updateParticipant(update.user_id, {
			isCameraOn: update.is_camera_on,
			isMuted: update.is_muted,
			isScreenShare: update.is_screen_share
		});
	}

	// Create a peer connection
	private async createPeerConnection(userId: string, createOffer: boolean): Promise<void> {
		if (this.peers.has(userId)) return;

		console.log('[VideoCall] Creating peer connection to', userId, 'offer:', createOffer);

		const connection = new RTCPeerConnection({ iceServers: ICE_SERVERS });
		const remoteStream = new MediaStream();

		const peer: VideoPeerConnection = {
			connection,
			userId,
			videoTrack: null,
			audioTrack: null,
			remoteStream,
			remoteVideoElement: null,
			remoteScreenShare: false
		};

		this.peers.set(userId, peer);

		// Add local tracks
		if (this.localStream) {
			if (this.localStream.videoTrack) {
				connection.addTrack(this.localStream.videoTrack, this.localStream.stream);
			}
			if (this.localStream.audioTrack) {
				connection.addTrack(this.localStream.audioTrack, this.localStream.stream);
			}
		}

		// Handle incoming tracks
		connection.ontrack = (event) => {
			console.log('[VideoCall] Received track from', userId);
			event.streams[0].getTracks().forEach(track => {
				remoteStream.addTrack(track);
				
				if (track.kind === 'video') {
					peer.videoTrack = track;
				} else if (track.kind === 'audio') {
					peer.audioTrack = track;
				}
			});
			
			// Create video element for remote stream
			this.attachRemoteStream(userId, remoteStream);
		};

		// Handle ICE candidates
		connection.onicecandidate = (event) => {
			if (event.candidate && this.currentCallId) {
				gateway.send({
					op: Op.DISPATCH,
					d: {
						t: 'VIDEO_ICE_CANDIDATE',
						d: {
							call_id: this.currentCallId,
							to_user_id: userId,
							candidate: event.candidate.candidate,
							sdpMid: event.candidate.sdpMid,
							sdpMLineIndex: event.candidate.sdpMLineIndex
						}
					}
				});
			}
		};

		// Handle connection state
		connection.onconnectionstatechange = () => {
			console.log('[VideoCall] Connection state with', userId, ':', connection.connectionState);
			if (connection.connectionState === 'failed' || connection.connectionState === 'disconnected') {
				this.closePeerConnection(userId);
			}
		};

		// Create offer if we're initiating
		if (createOffer && this.currentCallId) {
			await this.createAndSendOffer(userId, connection);
		}
	}

	// Create and send offer
	private async createAndSendOffer(userId: string, connection: RTCPeerConnection): Promise<void> {
		if (!this.currentCallId) return;

		try {
			const offer = await connection.createOffer();
			await connection.setLocalDescription(offer);

			gateway.send({
				op: Op.DISPATCH,
				d: {
					t: 'VIDEO_OFFER',
					d: {
						call_id: this.currentCallId,
						to_user_id: userId,
						sdp: offer.sdp
					}
				}
			});
		} catch (error) {
			console.error('[VideoCall] Failed to create offer:', error);
		}
	}

	// Attach remote stream to video element
	private attachRemoteStream(userId: string, stream: MediaStream): void {
		const existingVideo = document.getElementById(`video-${userId}`) as HTMLVideoElement;
		
		if (existingVideo) {
			existingVideo.srcObject = stream;
			return;
		}

		const video = document.createElement('video');
		video.id = `video-${userId}`;
		video.srcObject = stream;
		video.autoplay = true;
		video.playsInline = true;
		video.style.display = 'none';
		document.body.appendChild(video);

		const peer = this.peers.get(userId);
		if (peer) {
			peer.remoteVideoElement = video;
		}
	}

	// Close peer connection
	private closePeerConnection(userId: string): void {
		const peer = this.peers.get(userId);
		if (!peer) return;

		console.log('[VideoCall] Closing peer connection to', userId);

		peer.connection.close();
		
		// Remove video element
		if (peer.remoteVideoElement) {
			peer.remoteVideoElement.remove();
		}

		this.peers.delete(userId);
	}

	// Toggle camera
	async toggleCamera(): Promise<void> {
		if (!this.localStream || !this.localStream.videoTrack) return;

		const enabled = !this.localStream.videoTrack.enabled;
		this.localStream.videoTrack.enabled = enabled;

		videoCallStore.toggleCamera();
	}

	// Toggle mute
	toggleMute(): void {
		if (!this.localStream || !this.localStream.audioTrack) return;

		const enabled = !this.localStream.audioTrack.enabled;
		this.localStream.audioTrack.enabled = enabled;

		videoCallStore.toggleMute();
	}

	// Start screen share
	async startScreenShare(): Promise<void> {
		if (!browser) return;

		try {
			const screenStream = await navigator.mediaDevices.getDisplayMedia({
				video: true,
				audio: false
			});

			const screenTrack = screenStream.getVideoTracks()[0];

			if (this.localStream) {
				// Replace video track in all peer connections
				for (const [, peer] of this.peers) {
					const sender = peer.connection.getSenders().find(s => s.track?.kind === 'video');
					if (sender) {
						await sender.replaceTrack(screenTrack);
					}
				}

				// Store original video track
				this.localStream.screenShareTrack = this.localStream.videoTrack;
				this.localStream.screenShareStream = screenStream;
				this.localStream.videoTrack = screenTrack;
				this.screenShareEnabled = true;

				// Notify other participants
				if (this.currentCallId) {
					gateway.send({
						op: Op.DISPATCH,
						d: {
							t: 'VIDEO_SCREEN_START',
							d: {
								call_id: this.currentCallId,
								is_screen_share: true
							}
						}
					});
				}
			}

			// Handle screen share stop (when user clicks "Stop sharing" in browser)
			screenTrack.onended = () => {
				this.stopScreenShare();
			};

			videoCallStore.toggleScreenShare();
			console.log('[VideoCall] Screen share started');
		} catch (error) {
			console.error('[VideoCall] Failed to start screen share:', error);
		}
	}

	// Stop screen share
	async stopScreenShare(): Promise<void> {
		if (!this.localStream || !this.screenShareEnabled) return;

		const originalTrack = this.localStream.screenShareTrack;

		if (originalTrack) {
			// Replace back to original video track
			for (const [, peer] of this.peers) {
				const sender = peer.connection.getSenders().find(s => s.track?.kind === 'video');
				if (sender) {
					await sender.replaceTrack(originalTrack);
				}
			}

			this.localStream.videoTrack = originalTrack;
			this.localStream.screenShareTrack = null;
		}

		// Stop screen share stream
		if (this.localStream.screenShareStream) {
			this.localStream.screenShareStream.getTracks().forEach(track => track.stop());
			this.localStream.screenShareStream = null;
		}

		this.screenShareEnabled = false;

		// Notify other participants
		if (this.currentCallId) {
			gateway.send({
				op: Op.DISPATCH,
				d: {
					t: 'VIDEO_SCREEN_STOP',
					d: {
						call_id: this.currentCallId,
						is_screen_share: false
					}
				}
			});
		}

		videoCallStore.toggleScreenShare();
		console.log('[VideoCall] Screen share stopped');
	}

	// Toggle screen share
	async toggleScreenShare(): Promise<void> {
		if (this.screenShareEnabled) {
			await this.stopScreenShare();
		} else {
			await this.startScreenShare();
		}
	}

	// Get local video stream
	getLocalStream(): MediaStream | null {
		return this.localStream?.stream || null;
	}

	// Get remote video stream for a user
	getRemoteStream(userId: string): MediaStream | null {
		const peer = this.peers.get(userId);
		return peer?.remoteStream || null;
	}

	// Cleanup
	private cleanup(): void {
		console.log('[VideoCall] Cleaning up');

		// Close all peer connections
		for (const [userId] of this.peers) {
			this.closePeerConnection(userId);
		}

		// Stop local stream
		if (this.localStream) {
			if (this.localStream.stream) {
				this.localStream.stream.getTracks().forEach(track => track.stop());
			}
			if (this.localStream.screenShareStream) {
				this.localStream.screenShareStream.getTracks().forEach(track => track.stop());
			}
			this.localStream = null;
		}

		this.currentCallId = null;
		this.screenShareEnabled = false;
	}

	// Destroy manager
	destroy(): void {
		this.cleanup();
		this.cleanupFunctions.forEach(fn => fn());
		this.cleanupFunctions = [];
	}
}

// Singleton instance
let instance: VideoCallManager | null = null;

export function getVideoCallManager(): VideoCallManager {
	if (!instance && browser) {
		instance = new VideoCallManager();
	}
	return instance!;
}

export function destroyVideoCallManager() {
	if (instance) {
		instance.destroy();
		instance = null;
	}
}
