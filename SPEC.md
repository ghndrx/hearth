# Video Calling Core System - Implementation Specification

## Overview
This document outlines the implementation plan for the Video Calling Core System, providing 1-on-1 and group video calls with screen sharing. This implementation leverages the existing WebSocket gateway infrastructure and voice signaling system.

## Architecture

### Backend Components

#### 1. Video Signaling Service (`backend/internal/websocket/video.go`)
- Extends voice signaling for video-specific messages
- Handles VIDEO_OFFER, VIDEO_ANSWER, VIDEO_ICE_CANDIDATE
- Manages call state (ringing, connected, ended)
- Supports screen sharing signaling

#### 2. Gateway Integration (`gateway.go`)
- Add VIDEO_* message type handlers
- Route video messages through video service

### Frontend Components

#### 1. Video Call Manager (`frontend/src/lib/voice/VideoCallManager.ts`)
- Extends VoiceConnectionManager with video support
- Manages local video stream (camera)
- Handles screen sharing MediaStream
- Manages video peer connections

#### 2. Video Call Store (`frontend/src/lib/stores/videoCall.ts`)
- Call state management (ringing, connected, ended)
- Participant video state tracking
- Screen sharing state

#### 3. Video UI Components
- VideoCallModal - Main video call interface
- VideoGrid - Participant video grid layout
- VideoControls - Mute, camera, screen share, hang up controls

## Message Types

### Backend → Client Events
```
VIDEO_RING_START      - Incoming call notification
VIDEO_RING_ACCEPT     - Call accepted, start connecting
VIDEO_RING_DECLINE    - Call declined
VIDEO_RING_END        - Call ended
VIDEO_STATE_UPDATE    - Participant video state changed
VIDEO_SERVER_UPDATE   - Full video state for joining
```

### Client → Server Messages
```
VIDEO_RING            - Initiate a call
VIDEO_RING_RESPONSE   - Accept/decline incoming call
VIDEO_LEAVE           - Leave/end call
VIDEO_STATE_UPDATE    - Update own video state (mute, camera on/off)
VIDEO_SCREEN_START    - Start screen sharing
VIDEO_SCREEN_STOP     - Stop screen sharing
```

### WebRTC Signaling (within VIDEO_OFFER/ANSWER/CANDIDATE)
```
VIDEO_OFFER           - SDP offer for video connection
VIDEO_ANSWER         - SDP answer for video connection
VIDEO_ICE_CANDIDATE  - ICE candidate for NAT traversal
```

## Call States

```
IDLE         - No active call
RINGING_OUT  - Outgoing call in progress (waiting for accept)
RINGING_IN   - Incoming call notification
CONNECTING   - Call accepted, establishing connection
CONNECTED    - Call active
RECONNECTING - Connection lost, attempting to reconnect
ENDED        - Call ended
```

## Implementation Phases

### Phase 1: Core Backend Signaling
1. Create `video.go` with VideoSignalingService
2. Add VIDEO_* constants and message types
3. Integrate with gateway message routing
4. Add video state repository methods

### Phase 2: Core Frontend Infrastructure
1. Create `videoCall.ts` store
2. Create `VideoCallManager.ts` extending voice WebRTC
3. Add video message handlers to gateway store

### Phase 3: 1-on-1 Video Calls
1. Implement call initiation (ring)
2. Implement call acceptance/decline
3. Implement WebRTC video connection
4. Implement video controls (mute, camera toggle)

### Phase 4: Screen Sharing
1. Screen share stream capture
2. Screen share signaling
3. Screen share toggle controls

### Phase 5: Group Calls
1. Multi-participant video grid
2. Dynamic participant management
3. Group call state synchronization

## Data Structures

### VideoCallState (Backend)
```go
type VideoCallState struct {
    CallID        uuid.UUID
    ChannelID     uuid.UUID
    CallType      CallType // DIRECT, GROUP, CHANNEL
    InitiatorID   uuid.UUID
    State         CallState
    StartedAt     time.Time
    EndedAt       *time.Time
}
```

### VideoParticipant (Backend)
```go
type VideoParticipant struct {
    UserID         uuid.UUID
    CallID         uuid.UUID
    IsCameraOn     bool
    IsScreenShare  bool
    IsMuted        bool
    JoinedAt       time.Time
}
```

### VideoStateData (WebSocket Event)
```go
type VideoStateData struct {
    CallID       uuid.UUID    `json:"call_id"`
    ChannelID    uuid.UUID    `json:"channel_id"`
    State        CallState     `json:"state"`
    Participants []Participant `json:"participants"`
}
```

## Technical Notes

### WebRTC Configuration
- ICE Servers: STUN (Google) + configurable TURN
- Video codec: VP8/VP9/H.264 (browser dependent)
- Audio codec: Opus
- Resolution: 720p default, adaptive based on bandwidth

### Screen Sharing
- Uses `navigator.mediaDevices.getDisplayMedia()`
- Separate MediaStream for screen share track
- Replaces camera track when sharing

### Connection Flow
1. User initiates call → VIDEO_RING sent to target
2. Target receives ring → UI shows incoming call
3. Target accepts → VIDEO_RING_ACCEPT sent
4. Initiator receives accept → Creates WebRTC offer
5. SDP exchange via VIDEO_OFFER/ANSWER
6. ICE candidates exchanged via VIDEO_ICE_CANDIDATE
7. Connection established → VIDEO_STATE_UPDATE broadcast

## File Structure

```
backend/internal/websocket/
├── video.go              # Video signaling service
├── video_signaling_test.go
└── ...

frontend/src/lib/
├── voice/
│   ├── VideoCallManager.ts  # Video WebRTC manager
│   └── ...
├── stores/
│   ├── videoCall.ts         # Video call state store
│   └── ...
└── components/
    ├── VideoCallModal.svelte
    ├── VideoGrid.svelte
    └── VideoControls.svelte
```
