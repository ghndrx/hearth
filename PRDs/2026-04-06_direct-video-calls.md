---
name: Direct Video/Audio Calls
description: Private peer-to-peer video and audio calls between users outside of voice channels
type: Core Feature
---

# Direct Video/Audio Calls

## Discord Equivalent
1:1 and small group (2-10 users) direct video/audio calls with call screens, ringing, and WebRTC signaling

## User Value Proposition
- **Essential baseline expectation** — Users expect to call friends directly
- **Privacy-focused communication** — Calls don't require joining public voice channels
- **Mobile parity** — Critical for mobile app user retention
- **Relationship building** — Private calls foster closer user connections

## Technical Complexity: P0
**Estimated effort**: 10-14 weeks

## Implementation Sketch

### Backend Changes
1. **New Models**
   ```go
   // Call model with states: initiating, ringing, active, ended, declined
   type Call struct {
       ID           string
       InitiatorID  string
       Participants []string
       Type         CallType // audio, video
       Status       CallStatus
       StartedAt    *time.Time
       EndedAt      *time.Time
       Duration     time.Duration
   }
   ```

2. **WebSocket Events**
   - `CALL_CREATE` - Initiate call with user(s)
   - `CALL_RING` - Send ring notification to participants
   - `CALL_ANSWER` - Accept incoming call
   - `CALL_DECLINE` - Decline call
   - `CALL_END` - End active call
   - `CALL_UPDATE` - Update call state (mute/video toggle)

3. **WebRTC Signaling**
   - ICE candidate exchange via WebSocket
   - Offer/answer SDP negotiation
   - STUN/TURN server coordination for NAT traversal

4. **API Endpoints**
   - `POST /api/v1/users/@me/calls` - Initiate call
   - `POST /api/v1/calls/{id}/answer` - Answer call
   - `POST /api/v1/calls/{id}/decline` - Decline call
   - `DELETE /api/v1/calls/{id}` - End call

### Frontend Changes
1. **Call Interface Components**
   - `CallScreen.svelte` - Active call UI with video streams
   - `IncomingCallModal.svelte` - Ring screen with answer/decline
   - `CallControls.svelte` - Mute, camera, hang up controls
   - `CallNotification.svelte` - Incoming call notification

2. **WebRTC Integration**
   - Peer connection management
   - Local/remote stream handling
   - Audio/video device selection
   - Call quality indicators

3. **User Context Updates**
   - Call button in user profiles/DM headers
   - Call status in user presence
   - Call history in DM channels

## Dependencies
- **WebRTC infrastructure** — STUN/TURN servers configured
- **Push notifications** — For incoming calls when app is background
- **Audio/video permissions** — Browser media access
- **Mobile app readiness** — Essential for mobile call experience

## Success Metrics
- Call completion rate >80%
- Average call duration >3 minutes  
- Call quality rating >4.0/5.0
- Mobile call adoption >40% of total calls

## Risk Mitigation
- **WebRTC complexity** — Start with audio-only, add video later
- **Mobile app dependency** — May need to coordinate with mobile team
- **TURN server costs** — Monitor bandwidth usage and implement usage limits