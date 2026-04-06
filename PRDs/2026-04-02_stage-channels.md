---
name: Stage Channels
description: Live audio broadcast channels with speaker management for large community events
type: Feature PRD
priority: P0
---

# Stage Channels

## Discord Equivalent
Direct 1:1 match with Discord's Stage Channels:
- Live audio broadcasts to unlimited listeners
- Speaker/audience role management
- Request to speak functionality
- Moderated audio events
- Stage discovery and notifications

## User Value Proposition
**Essential for large community engagement** - Stage channels enable:
- **Large Events**: Host live talks, Q&As, and community discussions
- **Content Creation**: Podcasts, live interviews, and educational content
- **Community Building**: Regular programming builds audience loyalty
- **Moderation Control**: Structured audio events with speaker management
- **Discovery**: Find and join interesting live conversations

## Technical Complexity: P0 (High Impact, High Complexity)

### Implementation Sketch
```go
// Models
type StageChannel struct {
    ID                string    `json:"id" db:"id"`
    ServerID          string    `json:"server_id" db:"server_id"`
    Name              string    `json:"name" db:"name"`
    Topic             *string   `json:"topic" db:"topic"`
    Privacy           string    `json:"privacy" db:"privacy"` // PUBLIC, SERVER_ONLY
    DiscoveryDisabled bool      `json:"discovery_disabled" db:"discovery_disabled"`
    CreatedBy         string    `json:"created_by" db:"created_by"`
    CreatedAt         time.Time `json:"created_at" db:"created_at"`
    UpdatedAt         time.Time `json:"updated_at" db:"updated_at"`
}

type StageInstance struct {
    ID                 string     `json:"id" db:"id"`
    ChannelID          string     `json:"channel_id" db:"channel_id"`
    Topic              string     `json:"topic" db:"topic"`
    PrivacyLevel       string     `json:"privacy_level" db:"privacy_level"`
    DiscoveryDisabled  bool       `json:"discovery_disabled" db:"discovery_disabled"`
    StartedBy          string     `json:"started_by" db:"started_by"`
    StartedAt          time.Time  `json:"started_at" db:"started_at"`
    EndedAt            *time.Time `json:"ended_at" db:"ended_at"`
    ParticipantCount   int        `json:"participant_count" db:"participant_count"`
}

type StageParticipant struct {
    StageID      string    `json:"stage_id" db:"stage_id"`
    UserID       string    `json:"user_id" db:"user_id"`
    Role         string    `json:"role" db:"role"` // SPEAKER, MODERATOR, AUDIENCE
    RequestedAt  *time.Time `json:"requested_at" db:"requested_at"` // For speak requests
    JoinedAt     time.Time `json:"joined_at" db:"joined_at"`
    Muted        bool      `json:"muted" db:"muted"`
    Suppressed   bool      `json:"suppressed" db:"suppressed"`
}
```

### Key Features
1. **Stage Management**
   - Create/end stage instances
   - Set stage topic and privacy level
   - Manage speaker permissions
   - Audience/speaker role switching
   - Moderator controls (mute, suppress, remove)

2. **Audio Infrastructure**
   - WebRTC-based low-latency audio streaming
   - Scalable audio mixing for unlimited audience
   - Quality-adaptive audio encoding
   - Echo cancellation and noise suppression
   - Audio fallback for poor connections

3. **Participant Management**
   - Request to speak queue
   - Moderator approval/denial of speak requests
   - Automatic role assignment (speaker/audience)
   - Speaker spotlight and priority audio
   - Audience hand-raising functionality

4. **Discovery & Notifications**
   - Public stage discovery feed
   - Stage start notifications for interested users
   - Cross-server stage promotion (with permission)
   - Stage recording and replay (premium feature)

### Technical Architecture
```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   WebRTC SFU    │    │  Audio Mixer    │    │  CDN/Relay      │
│  (Selective     │    │  (Server-side   │    │  (Global        │
│   Forwarding)   │────│   processing)   │────│   Distribution) │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                        │                        │
         └────────────────────────┼────────────────────────┘
                                  │
                    ┌─────────────────┐
                    │  Stage Service  │
                    │  (Role mgmt,    │
                    │   permissions,  │
                    │   notifications)│
                    └─────────────────┘
```

### API Endpoints
- `POST /channels/{id}/stage` - Start stage instance
- `PATCH /stages/{id}` - Update stage (topic, privacy)
- `DELETE /stages/{id}` - End stage instance
- `POST /stages/{id}/speakers` - Request to speak
- `PUT /stages/{id}/participants/{user_id}` - Manage participant role
- `GET /stages/discover` - Public stage discovery

## Dependencies
1. **Advanced Voice Infrastructure** - Low-latency audio streaming at scale
2. **WebRTC SFU Service** - Selective forwarding unit for audio routing
3. **Permission System Enhancement** - Stage-specific permissions
4. **Real-time Events** - WebSocket notifications for stage updates
5. **CDN Infrastructure** - Global audio distribution

## Success Metrics
- Stage creation rate and duration
- Average audience size per stage
- Speaker-to-audience conversion rate
- Stage discovery engagement
- Community retention through regular programming

## Timeline Estimate
- **Phase 1** (6 weeks): Basic stage channel creation + audio infrastructure
- **Phase 2** (4 weeks): Speaker management + request to speak
- **Phase 3** (3 weeks): Discovery feed + advanced moderation
- **Phase 4** (2 weeks): Performance optimization + mobile support

**Total: 15 weeks for full Discord Stage Channel parity**

## Risk Assessment
- **High**: Complex audio infrastructure requires significant engineering investment
- **Medium**: WebRTC scaling challenges for large audiences
- **Low**: Feature adoption risk mitigated by Discord's proven success