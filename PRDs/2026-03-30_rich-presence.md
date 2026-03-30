---
name: Rich Presence & Game Integration
description: Display user activity including games, music, and custom status with rich metadata and integration with gaming platforms
type: feature
priority: P0
---

# Rich Presence & Game Integration

## Discord Equivalent
Discord's Rich Presence system showing:
- Currently playing games with detailed status (level, character, party info)
- Spotify/Apple Music listening status with album art and controls
- Custom Rich Presence via Discord RPC (developers can integrate their apps)
- "Ask to Join" functionality for multiplayer games
- Activity time tracking and game library

## User Value Proposition
- **Community Discovery**: Users find others playing the same games and can connect
- **Social Gaming**: Easy group formation with "Ask to Join" and party invites
- **Platform Identity**: Hearth becomes a gaming-focused social platform like Discord
- **User Engagement**: Rich presence creates conversation starters and social interaction
- **Developer Ecosystem**: Rich Presence API attracts game developers to integrate

## Technical Complexity: P0 (High)

### Core Features

#### 1. Automatic Game Detection
- Process scanning on desktop (Windows/macOS/Linux)
- Game database with rich metadata (titles, artwork, genres)
- Steam/Epic Games/Xbox Game Pass integration
- Mobile game detection (iOS/Android)
- Console game detection via API (PlayStation/Xbox/Nintendo)

#### 2. Rich Presence Display
- Game title, current activity (In Menu, In Game, etc.)
- Progress information (level, score, time played)
- Party information (2 of 4 players, Join button)
- Custom artwork and branding per game
- Elapsed/remaining time displays

#### 3. Music Integration
- Spotify API integration (what's playing, album art)
- Apple Music integration
- YouTube Music integration
- Local media player detection (VLC, iTunes, etc.)
- Play/pause controls in user profile

#### 4. Developer SDK
- Rich Presence SDK for game developers
- Custom activity states and metadata
- Party/lobby integration APIs
- Join/spectate functionality
- Achievement and progress sharing

#### 5. Privacy Controls
- Activity privacy settings (Everyone, Friends, Nobody)
- Game-specific privacy overrides
- "Show as Playing" toggle per detected application
- Invisible mode (appear offline while using Hearth)

### Implementation Sketch
```
/presence/
├── detection/
│   ├── process_scanner.go    # Desktop process detection
│   ├── mobile_detector.go    # Mobile app detection
│   └── console_api.go        # Gaming console APIs
├── integrations/
│   ├── spotify.go           # Music service APIs
│   ├── steam.go             # Gaming platform APIs
│   ├── epic.go              # Epic Games integration
│   └── xbox.go              # Xbox Live API
├── rpc/
│   ├── server.go            # Rich Presence RPC server
│   ├── client_sdk.go        # SDK for developers
│   └── protocol.go          # RPC protocol definitions
├── metadata/
│   ├── game_db.go           # Game metadata database
│   ├── artwork.go           # Asset management
│   └── updater.go           # Database updates
└── frontend/
    ├── PresenceDisplay.svelte
    ├── ActivityCard.svelte
    ├── PrivacySettings.svelte
    └── NowPlayingWidget.svelte
```

### Database Schema
```sql
-- User activities table
CREATE TABLE user_activities (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type VARCHAR(32) NOT NULL, -- 'game', 'music', 'custom'
    name VARCHAR(128) NOT NULL,
    details VARCHAR(128),
    state VARCHAR(128),
    application_id VARCHAR(64),
    large_image VARCHAR(512),
    small_image VARCHAR(512),
    party_id VARCHAR(128),
    party_size INTEGER,
    party_max INTEGER,
    start_timestamp BIGINT,
    end_timestamp BIGINT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Game database
CREATE TABLE games_database (
    id VARCHAR(64) PRIMARY KEY, -- Steam ID, Epic ID, etc.
    name VARCHAR(128) NOT NULL,
    icon_url VARCHAR(512),
    banner_url VARCHAR(512),
    genres TEXT[],
    developer VARCHAR(128),
    publisher VARCHAR(128),
    platforms TEXT[],
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- User privacy settings for activities
CREATE TABLE user_activity_privacy (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    default_visibility VARCHAR(16) DEFAULT 'everyone', -- 'everyone', 'friends', 'nobody'
    show_game_activity BOOLEAN DEFAULT TRUE,
    show_music_activity BOOLEAN DEFAULT TRUE,
    show_custom_activity BOOLEAN DEFAULT TRUE,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Per-application privacy overrides
CREATE TABLE app_privacy_overrides (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    application_id VARCHAR(64) NOT NULL,
    visibility VARCHAR(16) NOT NULL,
    PRIMARY KEY (user_id, application_id)
);
```

### Key Technical Challenges
- **Cross-Platform Detection**: Games detection across Windows, macOS, Linux, mobile
- **Performance**: Low CPU impact from continuous process scanning
- **Privacy**: Secure handling of user gaming data and activity
- **Rate Limits**: Managing API limits from Steam, Spotify, etc.
- **Real-time Updates**: Instant presence updates across all clients

## Dependencies
- Desktop client application (needed for process scanning)
- Mobile app development (for mobile game detection)
- Third-party API integrations (Steam, Spotify, Xbox, PlayStation)
- WebSocket real-time presence system (✓ exists)
- User privacy settings framework (needs enhancement)

## Success Metrics
- **User Engagement**: >60% of users have Rich Presence enabled
- **Social Discovery**: 25% increase in new friendships/connections via gaming
- **Session Length**: +15% average session time with rich presence enabled
- **Developer Adoption**: 100+ games integrate Rich Presence SDK in Year 1
- **Gaming Focus**: 40% of user conversations mention gaming content

## Rollout Strategy
1. **Phase 1**: Basic game detection and display (desktop only)
2. **Phase 2**: Music integration (Spotify)
3. **Phase 3**: Rich Presence SDK for developers
4. **Phase 4**: Mobile and console detection
5. **Phase 5**: "Ask to Join" and party features

## Competitive Analysis
- **Discord**: Market leader, comprehensive gaming integration
- **Steam**: Strong for Steam games but limited to Steam ecosystem
- **Xbox Social**: Good Xbox integration but platform-specific
- **Hearth Opportunity**: Cross-platform gaming hub with better privacy controls