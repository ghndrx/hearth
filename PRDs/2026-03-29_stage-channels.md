# Stage Channels PRD

## Feature Overview
Stage channels for live audio broadcasts to large audiences with speaker management and audience controls.

## Discord Equivalent
Discord Stage Channels - Audio-focused channels where designated speakers can present to a listening audience.

## User Value Proposition
- **Community Building**: Host town halls, Q&A sessions, presentations, and live discussions
- **Scalable Audio**: Support hundreds of listeners with minimal audio quality degradation
- **Moderation Control**: Clear speaker/audience separation with raise-hand functionality
- **Event Integration**: Natural fit with existing event system for scheduled broadcasts

## Technical Complexity Estimate
**P1** - Medium complexity

## Implementation Sketch
### Backend Changes
- New channel type: `CHANNEL_TYPE_STAGE`
- Stage-specific permissions: `MANAGE_STAGE`, `SPEAKER_PRIORITY`, `REQUEST_TO_SPEAK`
- WebRTC modifications for broadcast-style audio (1-to-many optimization)
- Stage state management: active speakers, audience, speaker queue
- Integration with voice infrastructure but optimized for many listeners

### Frontend Changes
- Stage channel UI with speaker/audience distinction
- Raise hand button and speaker request queue for moderators
- Stage controls for moderators (invite/remove speakers, manage audience)
- Visual indicators for speaking status and audio quality

### Database Schema
```sql
ALTER TABLE channels ADD COLUMN stage_settings JSONB;
-- stage_settings: {max_speakers: 10, auto_approve_speakers: false, ...}

CREATE TABLE stage_participants (
    user_id UUID NOT NULL,
    channel_id UUID NOT NULL,
    role VARCHAR(20) NOT NULL, -- 'speaker', 'audience', 'moderator'
    requested_speaker_at TIMESTAMP,
    PRIMARY KEY (user_id, channel_id)
);
```

## Dependencies
- Voice infrastructure must be operational
- Permissions system for stage-specific roles
- Event integration for scheduled stage events

## Success Metrics
- Stage channel creation rate
- Average audience size per stage
- Speaker engagement (requests to speak)
- Audio quality metrics for large audiences