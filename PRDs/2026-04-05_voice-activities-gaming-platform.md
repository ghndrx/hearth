---
name: Voice Activities Gaming Platform
description: Discord-style interactive games and activities in voice channels
type: feature
priority: P0
implementation_weeks: 8-12
---

# Voice Activities Gaming Platform

## Discord Equivalent
Discord's Voice Activities - interactive games and entertainment in voice channels including Poker, Chess, Watch Together, Betrayal, and custom activities.

## Problem Statement
Voice channels in Hearth lack engaging interactive activities, making them less sticky than Discord voice. Users expect built-in games and activities when in voice channels together.

## User Value Proposition
- **Increased Voice Engagement**: 40%+ more time spent in voice channels with activities
- **Social Connection**: Shared activities strengthen community bonds beyond conversation
- **Competitive Parity**: Essential Discord feature that users expect in any Discord alternative
- **Retention Driver**: Voice activities are proven to increase daily active usage

## Technical Complexity: P0 (8-12 weeks)

### Implementation Requirements

#### Backend Models (NEW)
```go
// Voice Activity System
type VoiceActivity struct {
    ID          string                 `json:"id"`
    ChannelID   string                 `json:"channel_id"`
    ActivityType VoiceActivityType     `json:"activity_type"`
    State       VoiceActivityState     `json:"state"`
    Participants []VoiceActivityUser   `json:"participants"`
    Settings    VoiceActivitySettings  `json:"settings"`
    CreatedBy   string                 `json:"created_by"`
    StartedAt   time.Time             `json:"started_at"`
}

type VoiceActivityType string
const (
    ActivityPoker       VoiceActivityType = "poker"
    ActivityChess       VoiceActivityType = "chess"  
    ActivityWatchTogether VoiceActivityType = "watch_together"
    ActivityBetrayal    VoiceActivityType = "betrayal"
    ActivityCustom      VoiceActivityType = "custom"
)
```

#### Core Features
1. **Activity Launcher**: Start activities from voice channel UI
2. **Participant Management**: Join/leave activity without leaving voice
3. **Game State Sync**: Real-time game state synchronization via WebSocket
4. **Activity Overlay**: UI overlay within voice channel interface
5. **Watch Together**: YouTube/Twitch synchronized viewing
6. **Game Library**: Extensible system for adding new activities

#### LiveKit Integration
- Leverage existing voice infrastructure 
- Add activity data channels for game state
- Screen sharing integration for Watch Together

### Dependencies
- ✅ Voice Channels (implemented)
- ✅ LiveKit Integration (implemented) 
- ✅ WebSocket Gateway (implemented)
- ⚠️ Screen Share Enhancement (needs completion)

## Success Metrics
- 60%+ of voice sessions include activity usage
- 2.5x average voice session duration
- 80% user satisfaction on voice experience surveys

## Implementation Phases
1. **Phase 1 (4 weeks)**: Core activity framework + Poker
2. **Phase 2 (3 weeks)**: Chess + Watch Together
3. **Phase 3 (2 weeks)**: Activity discovery UI + optimization
4. **Phase 4 (3 weeks)**: Custom activity SDK for developers