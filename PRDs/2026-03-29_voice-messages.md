# Voice Messages PRD

## Feature Overview
Audio message recording and playback system for text channels, enabling quick voice communication without joining voice channels.

## Discord Equivalent
Discord Voice Messages - Short audio recordings sent as messages in text channels with waveform visualization and playback controls.

## User Value Proposition
- **Quick Communication**: Faster than typing for complex explanations
- **Emotional Context**: Voice tone conveys emotion and nuance better than text
- **Accessibility**: Easier for users with typing difficulties or language barriers
- **Mobile-First**: Natural interaction model for mobile users

## Technical Complexity Estimate
**P1** - Medium complexity

## Implementation Sketch
### Voice Message Features
- Record up to 5 minutes of audio per message
- Waveform visualization during recording and playback
- Playback speed controls (0.5x, 1x, 1.25x, 2x)
- Voice message reactions and replies
- Automatic transcription (optional)
- Voice message search within transcriptions

### Recording Interface
- Push-to-talk or tap-to-record modes
- Real-time waveform display during recording
- Recording timer and remaining time indicator
- Cancel/delete recording before sending
- Preview playback before sending

### Backend Changes
```go
type VoiceMessage struct {
    ID           string    `json:"id"`
    MessageID    string    `json:"message_id"`
    Duration     int       `json:"duration"` // seconds
    FileURL      string    `json:"file_url"`
    Waveform     []int     `json:"waveform"` // amplitude data for visualization
    Transcription *string  `json:"transcription,omitempty"`
    FileSize     int64     `json:"file_size"`
    CreatedAt    time.Time `json:"created_at"`
}

type Message struct {
    // ... existing fields ...
    VoiceMessage *VoiceMessage `json:"voice_message,omitempty"`
}
```

### Audio Processing
- Audio encoding: Opus codec for quality and compression
- Waveform generation during upload processing
- Audio normalization and noise reduction
- File size limits based on server boost level
- CDN storage for voice message files

### Frontend Changes
- Voice message recorder component with visual feedback
- Waveform display component for playback
- Audio player controls integrated into message layout
- Mobile recording interface with hold-to-record gesture
- Voice message transcript display (if available)

### Database Schema
```sql
CREATE TABLE voice_messages (
    id UUID PRIMARY KEY,
    message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    file_path VARCHAR(500) NOT NULL,
    file_size BIGINT NOT NULL,
    duration_seconds INTEGER NOT NULL,
    waveform JSONB, -- array of amplitude values
    transcription TEXT,
    transcription_confidence DECIMAL(3,2),
    created_at TIMESTAMP DEFAULT NOW()
);

-- Index for voice message queries
CREATE INDEX idx_voice_messages_message_id ON voice_messages(message_id);
CREATE INDEX idx_voice_messages_transcription ON voice_messages USING gin(to_tsvector('english', transcription));
```

### Permissions & Settings
- Server permission: `SEND_VOICE_MESSAGES`
- User setting: auto-play voice messages
- Channel setting: voice messages enabled/disabled
- Mobile push notification settings for voice messages

### Transcription Integration
- Optional speech-to-text using AWS Transcribe or similar
- Transcription confidence scoring
- Manual transcription correction by sender
- Search integration for voice message content

## Dependencies
- File upload infrastructure for audio files
- Audio processing pipeline (encoding, waveform generation)
- CDN/storage solution for voice message files
- Optional: Speech-to-text service integration

## Success Metrics
- Voice message send rate vs text messages
- Voice message completion rate (listened to end)
- User adoption rate for voice messages
- Average voice message duration