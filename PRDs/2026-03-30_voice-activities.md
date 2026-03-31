---
name: Voice Activities & Games
description: Built-in activities and games for voice channels including Watch Together, Poker, Chess, and other social activities
type: feature
priority: P0
---

# Voice Activities & Games

## Discord Equivalent
Discord's "Activities" feature in voice channels - including Watch Together (YouTube), Poker Night, Chess in the Park, Sketch Heads, Word Snacks, and other built-in games and entertainment.

## User Value Proposition
- **Engagement**: Keeps users in voice channels longer with interactive content
- **Community Building**: Shared activities create stronger social bonds
- **Retention**: Activities provide reasons to stay on the platform beyond just communication
- **Monetization**: Premium activities can drive subscription revenue
- **User Acquisition**: Unique social features differentiate from competitors

## Technical Complexity: P0 (High)

### Core Implementation
- **Activity Framework**: Plugin system for embedding third-party activities in voice channels
- **Screen Sharing Integration**: Activities run in shared screen space visible to all participants
- **Real-time Sync**: Low-latency state synchronization for multiplayer games
- **Voice Integration**: Seamless audio mixing with activity audio/effects
- **Mobile Support**: Activities must work across desktop and mobile platforms

### Activity Types
1. **Media Consumption** (Watch Together)
   - YouTube integration with synchronized playback
   - Shared video controls (play/pause/seek)
   - Chat overlay during viewing

2. **Classic Games**
   - Chess with spectator mode
   - Poker with virtual chips
   - Drawing/guessing games (Pictionary-style)
   - Word games and trivia

3. **Social Activities**
   - Virtual whiteboards for collaboration
   - Music listening parties (Spotify/Apple Music)
   - Code collaboration environments

### Implementation Sketch
```
/activities/
├── framework/          # Core activity system
│   ├── lifecycle.go   # Start/stop/join/leave
│   ├── sync.go        # Real-time state management
│   └── permissions.go # Who can start activities
├── integrations/      # Third-party services
│   ├── youtube.go     # Watch Together
│   ├── spotify.go     # Music sync
│   └── webhook.go     # Custom activity webhooks
├── games/            # Built-in games
│   ├── chess/
│   ├── poker/
│   └── drawing/
└── frontend/         # Activity UI components
    ├── ActivityLauncher.svelte
    ├── ActivityOverlay.svelte
    └── game-widgets/
```

### Key Technical Challenges
- **Low Latency**: Games require <100ms sync for good UX
- **Scale**: Must support 25+ participants in some activities
- **Mobile Performance**: Smooth gameplay on lower-end devices
- **Cross-Platform**: Consistent experience across web/desktop/mobile

## Dependencies
- Voice channel infrastructure (✓ exists)
- Screen sharing system (✓ exists)
- Real-time messaging framework (✓ exists)
- WebRTC optimization for low-latency sync
- Mobile app parity for activity support

## Success Metrics
- Voice channel session duration (+25% target)
- Daily active voice users (+15% target)
- Activity usage rate (>40% of voice sessions)
- User retention improvement (+10% monthly retention)
- Premium conversion from exclusive activities

## Competitive Analysis
- **Discord**: Market leader, 50+ activities available
- **Microsoft Teams**: Limited games, business-focused
- **Zoom**: Whiteboard but minimal gaming
- **Hearth Opportunity**: Focus on creator economy - let communities build custom activities