# Video Calling & Group Video System

## Feature Name
Comprehensive Video Calling System

## Discord Equivalent
Discord's video calling capabilities including direct video calls, group video calls, camera/screen sharing, picture-in-picture, video quality adaptation, and integrated voice/video controls within channels and DMs.

## User Value Proposition
- **Complete Communication**: Voice + video for full Discord-equivalent experience
- **Mobile/Desktop Parity**: Essential for modern remote communication and gaming
- **Community Building**: Video events, group calls, and face-to-face interaction
- **Competitive Necessity**: Video calling is table stakes for Discord alternatives

## Technical Complexity Estimate
**P0** - Critical priority, very high complexity requiring:
- WebRTC video implementation with multi-peer support
- Camera/video device management
- Video quality adaptation and bandwidth optimization
- Mobile video optimization
- Group video call coordination (SFU architecture)

## Implementation Sketch

### High-Level Architecture
1. **Video Infrastructure**:
   - WebRTC video streams with adaptive bitrate
   - Selective Forwarding Unit (SFU) for efficient group video
   - Video codec support (VP8, VP9, H.264, AV1)
   - Resolution scaling (480p, 720p, 1080p)
   - Frame rate adaptation (15fps, 30fps, 60fps)

2. **Video Call Types**:
   - **Direct Video Calls**: 1-on-1 video in DMs
   - **Group Video Calls**: Multi-participant video (up to 25 users)
   - **Channel Video**: Video enabled in voice channels
   - **Screen Share Video**: Application/screen capture with audio

3. **Video Features**:
   ```go
   type VideoCall struct {
       ID          uuid.UUID     `json:"id"`
       ChannelID   uuid.UUID     `json:"channel_id"`
       Type        VideoCallType `json:"type"` // dm, group_dm, channel
       Participants []VideoParticipant `json:"participants"`
       StartedAt   time.Time     `json:"started_at"`
       EndedAt     *time.Time    `json:"ended_at"`
       MaxParticipants int       `json:"max_participants"`
   }

   type VideoParticipant struct {
       UserID      uuid.UUID         `json:"user_id"`
       StreamID    string           `json:"stream_id"`
       IsCamera    bool             `json:"is_camera"`
       IsScreen    bool             `json:"is_screen"`
       Resolution  VideoResolution  `json:"resolution"`
       Quality     VideoQuality     `json:"quality"`
       Muted       bool             `json:"muted"`
       VideoMuted  bool             `json:"video_muted"`
   }
   ```

4. **Video Quality Management**:
   - Automatic quality adaptation based on bandwidth
   - Manual quality controls (auto, 1080p, 720p, 480p)
   - Bandwidth estimation and adjustment
   - CPU usage monitoring and optimization

5. **Mobile Optimization**:
   - Hardware acceleration when available
   - Battery usage optimization
   - Data usage controls (Wi-Fi only, limited data mode)
   - Background video pause/resume

6. **UI Components**:
   - Picture-in-picture mode
   - Grid view for group calls (2x2, 3x3 layouts)
   - Full-screen video mode
   - Video controls overlay (mute, camera, screen share, hang up)
   - Participant list with video status
   - Chat overlay during video calls

## Dependencies
- **Prerequisites**:
  - WebRTC voice system ✅ (implemented)
  - Screen sharing foundation ✅ (partially implemented)
  - Voice state management ✅ (implemented)
  - Media device permissions ⚠️ (needs camera permissions)

- **Blocking Requirements**:
  - SFU media server deployment (requires infrastructure)
  - Frontend video rendering components
  - Mobile camera/video optimization
  - Bandwidth monitoring and QoS

- **Integration Points**:
  - Existing voice infrastructure ✅ (can extend)
  - Push notification system ✅ (for call alerts)
  - Presence system ✅ (for availability status)

## Success Metrics
- **Adoption**: 60% of voice calls include video within 6 months
- **Quality**: <2% call drop rate, avg 720p quality maintained
- **Performance**: Video calls start within 3 seconds of initiation
- **Mobile**: 80% of mobile users use video calling feature
- **Engagement**: 25% increase in average call duration with video

## Risk Mitigation
- **Infrastructure Cost**: Start with peer-to-peer, move to SFU for groups
- **Mobile Performance**: Aggressive optimization, fallback to voice-only
- **Bandwidth Issues**: Quality adaptation, data usage warnings
- **Privacy Concerns**: Clear camera indicators, easy mute controls
- **Cross-Platform**: Start with web, ensure mobile web compatibility

## Rollout Strategy
1. **Phase 1**: 1-on-1 direct video calls in DMs (P2P WebRTC)
2. **Phase 2**: Group video calls up to 5 participants (simple SFU)
3. **Phase 3**: Channel video calling and screen share integration
4. **Phase 4**: Advanced features (PiP, quality controls, recordings)

## Security TODO (from security-webrtc-video-calls.md)
- [ ] **WSS mandatory** for all video signaling (VIDEO_RING, VIDEO_OFFER, etc.)
- [ ] **TURN credential rotation** — hourly HMAC credentials, never static
- [ ] **ICE candidate filtering** — prevent IP leaks by filtering host candidates
- [ ] **Permission checks** — verify channel membership before establishing calls
- [ ] **Audit logging** — log call initiation, join/leave, screen share events
- [ ] **E2EE consideration** — Discord-style app-layer encryption for v2 (privacy codes)