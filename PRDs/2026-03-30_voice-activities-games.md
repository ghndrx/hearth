---
name: Voice Activities & Games
description: Built-in activities and games playable directly in voice channels
type: feature
priority: P1
complexity: High
dependencies: Voice system, WebRTC, iframe sandboxing
---

# Voice Activities & Games

## Discord Equivalent
Discord's Activities feature allowing users to play games like Poker, Chess, Watch Together, Gartic Phone, and other interactive experiences directly in voice channels.

## User Value Proposition
- **Social Gaming**: Facilitates group activities without external tools
- **Voice Channel Stickiness**: Keeps users engaged in voice longer
- **Community Bonding**: Shared activities strengthen relationships
- **Competitive Differentiation**: Modern platform feature expected by Gen Z users

## Technical Complexity: P1 (High)
**Backend Changes:**
- Activity lifecycle management (start, join, leave, end)
- WebRTC screen sharing integration for shared experiences
- Activity state synchronization across participants
- Third-party activity iframe sandboxing and security

**Frontend Changes:**
- Activity launcher UI in voice channels
- Embedded activity iframe containers
- Activity participant management
- Screen sharing integration for activities

## Implementation Sketch
1. **Activity Framework**:
   - Activity manifest system (metadata, permissions, capabilities)
   - Sandboxed iframe execution environment
   - Activity-to-voice bridge for participant management
   - Real-time state sync via WebSocket

2. **Core Activities** (Phase 1):
   - **Watch Together**: YouTube/streaming sync for group viewing
   - **Whiteboard**: Collaborative drawing/brainstorming
   - **Poker**: Texas Hold'em for voice channels
   - **Chess**: 1v1 chess with spectators

3. **API Integration**:
   - `POST /channels/:id/activities` - Start activity
   - `POST /activities/:id/join` - Join activity
   - `DELETE /activities/:id/participants/@me` - Leave activity
   - WebSocket events for activity state changes

4. **Security Model**:
   - CSP-restricted iframes for third-party activities
   - Activity permission system (server/channel based)
   - User consent for screen sharing activities

## Dependencies
- Voice system with LiveKit (✅ implemented)
- Screen sharing (✅ implemented)
- WebSocket gateway (✅ implemented)
- Permission system (✅ implemented)

## Success Metrics
- Voice channel session length +40%
- Activity adoption >30% of voice users
- User retention from activities +25%