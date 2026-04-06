# Voice Activities & Games

## Feature Name
Voice Activities & Games Integration

## Discord Equivalent
Discord Activities - Built-in games and activities like poker, chess, watch-together, drawing boards, and custom web apps that users can play together in voice channels.

## User Value Proposition
- **Engagement**: Keep users in voice channels longer with interactive content
- **Community Building**: Shared activities create stronger social bonds
- **Retention**: Fun activities encourage regular platform usage
- **Differentiation**: Premium social features that set Hearth apart from basic chat apps

## Technical Complexity Estimate
**P1** - Complex integration requiring:
- WebRTC screen sharing infrastructure (already exists)
- Embedded iframe/web app system with voice integration
- Activity state synchronization across participants
- Custom game/activity development or third-party integrations

## Implementation Sketch

### High-Level Architecture
1. **Activity Framework**: Plugin system for voice channel activities
2. **Built-in Activities**:
   - Watch Together (shared video streaming)
   - Poker/card games with voice chat
   - Drawing/whiteboard with voice collaboration
   - Chess/board games
3. **Third-Party Integration**: SDK for developers to create custom activities
4. **Voice Integration**: Activities overlay on existing voice infrastructure

### Core Components
- Activity launcher UI in voice channels
- Embedded web app container with voice context
- Activity state management and synchronization
- Permission system for activity access
- Activity marketplace for discovery

## Dependencies
- **Must ship first**:
  - Stable voice channel infrastructure ✅ (already implemented)
  - Screen sharing capabilities ✅ (already implemented)
  - WebSocket real-time communication ✅ (already implemented)

## Success Metrics
- Voice channel session duration (+30%)
- Daily active voice users (+15%)
- User retention in servers with activities (+20%)