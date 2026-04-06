---
name: Activity & Rich Presence System
description: Comprehensive user activity detection including gaming, music, and custom activities with rich status display
type: feature
priority: P1
discord_equivalent: Rich Presence / Activity Detection
---

# Activity & Rich Presence System

## Overview

**Discord equivalent**: Rich Presence + Activity Detection
**User value proposition**: Social engagement through activity awareness - see what friends are playing, listening to, or doing
**Technical complexity**: P1 (High complexity, multi-platform integration)

## Problem Statement

Current Hearth only has basic custom status (text + emoji) documented in existing PRDs. Missing the **social awareness layer** that makes Discord feel alive:

- **No gaming activity detection**: Can't see "Playing Counter-Strike 2"
- **No music integration**: Missing "Listening to Spotify"
- **No rich presence API**: Games can't show detailed status
- **No activity history**: No social context about user behavior

This significantly reduces social engagement, especially in gaming communities where activity awareness drives conversation and group formation.

## Solution

Implement comprehensive activity detection and rich presence system:

### Core Activity Types
1. **Gaming Activities**: Auto-detected running games
2. **Music Activities**: Spotify/Apple Music integration
3. **Streaming Activities**: OBS/streaming software detection
4. **Custom Activities**: User-defined activities with rich data
5. **Application Activities**: Discord/browser/other app usage

### Rich Presence Features
- **Detailed status**: Game state, song info, custom data
- **Timestamps**: "Started 25 minutes ago", "Ends in 3:42"
- **Images**: Game artwork, album covers, custom icons
- **Interactive buttons**: "Join Game", "Listen Along"
- **Party information**: "In a party of 4", "2 of 6 slots"

## Technical Implementation

### Backend Changes
```go
// New models
type UserActivity struct {
    ID          uuid.UUID            `json:"id"`
    UserID      uuid.UUID            `json:"user_id"`
    Type        ActivityType         `json:"type"`
    Name        string               `json:"name"`
    Details     string               `json:"details,omitempty"`
    State       string               `json:"state,omitempty"`
    StartedAt   *time.Time           `json:"started_at,omitempty"`
    EndsAt      *time.Time           `json:"ends_at,omitempty"`
    LargeImage  string               `json:"large_image,omitempty"`
    SmallImage  string               `json:"small_image,omitempty"`
    PartyID     string               `json:"party_id,omitempty"`
    PartySize   int                  `json:"party_size,omitempty"`
    PartyMax    int                  `json:"party_max,omitempty"`
    Buttons     []ActivityButton     `json:"buttons,omitempty"`
    Metadata    map[string]interface{} `json:"metadata,omitempty"`
    CreatedAt   time.Time            `json:"created_at"`
    UpdatedAt   time.Time            `json:"updated_at"`
}

type ActivityType string
const (
    ActivityTypeGame     ActivityType = "GAME"
    ActivityTypeMusic    ActivityType = "MUSIC"
    ActivityTypeStreaming ActivityType = "STREAMING"
    ActivityTypeCustom   ActivityType = "CUSTOM"
    ActivityTypeApp      ActivityType = "APPLICATION"
)

type ActivityButton struct {
    Label string `json:"label"`
    URL   string `json:"url"`
}
```

### Platform-Specific Detection

#### Desktop Client (Required)
```javascript
// Electron main process
const ActivityDetector = {
  // Game detection via process monitoring
  detectGames: () => {
    // Windows: WMI process queries
    // macOS: Activity Monitor API
    // Linux: /proc filesystem
  },

  // Music detection via media APIs
  detectMusic: () => {
    // Windows: Windows.Media.Control API
    // macOS: MediaPlayer framework
    // Linux: MPRIS D-Bus interface
  }
}
```

#### Web Client (Limited)
```javascript
// Browser-based detection (limited capabilities)
const WebActivityDetector = {
  // Only custom activities and some web-based detection
  setCustomActivity: (activity) => { /* ... */ },
  detectTabActivity: () => { /* YouTube, Spotify Web, etc. */ }
}
```

### API Integration

#### Spotify Integration
```go
type SpotifyIntegration struct {
    AccessToken  string
    RefreshToken string
    ExpiresAt    time.Time
}

// Spotify Web API calls
func (s *SpotifyIntegration) GetCurrentTrack() (*SpotifyTrack, error)
```

#### Rich Presence API (for game developers)
```javascript
// Game integration SDK
const HearthRPC = {
  setActivity: {
    details: "Playing ranked match",
    state: "In game (Round 15/30)",
    startTimestamp: Date.now(),
    largeImageKey: "csgo_logo",
    smallImageKey: "rank_global_elite",
    partySize: 2,
    partyMax: 5,
    buttons: [
      { label: "Join Game", url: "hearth://join/game123" }
    ]
  }
}
```

## Dependencies

### Prerequisites
1. **Desktop client** (Electron app) - **CRITICAL DEPENDENCY**
2. **Presence system** ✅ (already implemented)
3. **User settings system** ✅ (already implemented)

### New Dependencies
- **Desktop application development** (Electron/Tauri)
- **Platform-specific APIs** (Windows WMI, macOS MediaPlayer, Linux D-Bus)
- **OAuth integrations** (Spotify, Apple Music APIs)
- **Rich Presence SDK** (for game developers)

## Success Metrics

- **Activity adoption**: 70% of desktop users show at least one activity per session
- **Social engagement**: 25% increase in DMs when friends see interesting activities
- **Gaming communities**: 40% of gaming servers have members showing game activities
- **Music integration**: 30% of users connect Spotify within 7 days

## Implementation Phases

### Phase 1: Desktop Foundation (4 weeks)
- Electron desktop client development
- Basic process detection (games/apps)
- Activity broadcasting to backend
- Simple activity display in UI

### Phase 2: Rich Data & Music (3 weeks)
- Spotify/Apple Music integration
- Rich presence data structure
- Timestamps and duration tracking
- Enhanced activity display

### Phase 3: Developer API & Polish (2 weeks)
- Rich Presence SDK for game developers
- Activity buttons and interactions
- Privacy controls and activity filtering
- Performance optimization

## Privacy & Settings

- **Activity privacy**: Per-activity type visibility controls
- **Game detection opt-out**: Disable specific game tracking
- **Music privacy**: Hide listening activity option
- **Activity history**: Optional activity logging/stats

## Risk Assessment

**High Risk**: Requires desktop client development and platform-specific integrations

**Potential Issues**:
- **Platform compatibility**: Different APIs across Windows/macOS/Linux
- **Privacy concerns**: Process monitoring permissions
- **Performance impact**: Background detection overhead
- **Desktop client adoption**: Users must install native app

**Mitigation**:
- **Progressive rollout**: Start with web-based custom activities
- **Clear privacy controls**: Transparent about what's detected
- **Lightweight detection**: Minimal system resource usage
- **Fallback support**: Rich presence works without desktop client

## Example Activity Display
```
🎮 Playing Counter-Strike 2
   Competitive Match • Round 12/30
   Started 23 minutes ago
   [🎯 Join Game] [📊 View Stats]

🎵 Listening to Spotify
   "Bohemian Rhapsody" by Queen
   on album A Night at the Opera
   [▶️ Listen Along]

📺 Streaming on Twitch
   "Building a Discord Clone"
   12 viewers • Started 1 hour ago
   [👀 Watch Stream]
```

This transforms Hearth from a static communication tool into a dynamic social platform where users can discover what their friends are doing and join activities together.