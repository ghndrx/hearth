---
name: Voice Messages
description: Audio message recording and playback in text channels (Discord-style voice notes)
type: feature
priority: P0
complexity: Medium
dependencies: Voice system, file storage, audio processing
---

# Voice Messages

## Discord Equivalent
Discord's voice message feature allowing users to record and send short audio clips directly in text channels, with waveform visualization and playback controls.

## User Value Proposition
- **Mobile-First Communication**: Essential for mobile users who prefer voice over typing
- **Accessibility**: Helps users with typing difficulties or language barriers
- **Emotional Context**: Voice tone conveys more than text for personal communication
- **User Acquisition**: Critical mobile feature that drives user adoption and retention
- **Competitive Necessity**: Missing this puts Hearth at significant disadvantage vs Discord

## Technical Complexity: P0 (Medium)
**Backend Changes:**
- Audio recording/upload system with compression (WebM/Opus)
- Waveform generation and visualization data
- Audio duration and size limits (max 3 minutes, 8MB)
- Voice message metadata storage and indexing
- Audio transcription for accessibility (optional)

**Frontend Changes:**
- Record button in message input with visual feedback
- Waveform visualization during recording/playback
- Audio playback controls with progress tracking
- Mobile-optimized recording interface (hold-to-record)
- Permission prompts for microphone access

**Database Schema:**
```sql
-- Enhanced messages table for voice messages
ALTER TABLE messages ADD COLUMN voice_duration INTEGER; -- seconds
ALTER TABLE messages ADD COLUMN waveform_data TEXT; -- JSON array of amplitude values
ALTER TABLE messages ADD COLUMN transcript TEXT; -- optional AI transcription

-- Voice message metadata
CREATE TABLE voice_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id UUID REFERENCES messages(id),
    file_url VARCHAR NOT NULL,
    duration INTEGER NOT NULL, -- seconds
    waveform JSONB, -- amplitude data for visualization
    transcript TEXT, -- optional transcription
    created_at TIMESTAMP DEFAULT NOW()
);
```

## Implementation Sketch

### Recording System
```javascript
class VoiceMessageRecorder {
    async startRecording() {
        this.stream = await navigator.mediaDevices.getUserMedia({ audio: true });
        this.recorder = new MediaRecorder(this.stream, {
            mimeType: 'audio/webm;codecs=opus'
        });

        this.analyzeAudio(); // For waveform generation
        this.recorder.start();
    }

    analyzeAudio() {
        const audioContext = new AudioContext();
        const source = audioContext.createMediaStreamSource(this.stream);
        const analyser = audioContext.createAnalyser();

        // Generate waveform data in real-time
        this.visualizeWaveform(analyser);
    }
}
```

### Backend Processing
```go
func (s *MessageService) ProcessVoiceMessage(audioFile []byte) (*VoiceMessage, error) {
    // Validate audio format and duration
    duration, err := s.getAudioDuration(audioFile)
    if err != nil || duration > 180 { // 3 minute limit
        return nil, ErrInvalidVoiceMessage
    }

    // Generate waveform visualization data
    waveform, err := s.generateWaveform(audioFile)
    if err != nil {
        return nil, err
    }

    // Upload to storage
    fileURL, err := s.storage.UploadVoiceMessage(audioFile)
    if err != nil {
        return nil, err
    }

    // Optional: Generate transcription
    transcript, _ := s.ai.TranscribeAudio(audioFile)

    return &VoiceMessage{
        FileURL:    fileURL,
        Duration:   duration,
        Waveform:   waveform,
        Transcript: transcript,
    }, nil
}
```

### Mobile Optimization
- **Hold-to-Record**: Long press to start, release to send
- **Swipe to Cancel**: Swipe up while recording to cancel
- **Background Recording**: Continue recording during app backgrounding
- **Compression**: Adaptive bitrate based on network conditions

## Dependencies
1. **LiveKit Voice System**: Microphone access and audio processing ✅
2. **File Storage System**: S3/MinIO for audio file storage ✅
3. **Mobile Client**: Native mobile app for optimal UX
4. **Audio Processing**: Server-side audio analysis for waveforms

## Success Metrics
- Voice message adoption >40% of mobile users within 30 days
- Average session length increase +20% on mobile
- Voice message retention: >60% of users who send one send more
- Mobile user acquisition improvement +25%

## Implementation Priority
**P0** - Critical mobile feature gap. Discord's voice messages are heavily used and expected by mobile-first users. Missing this feature significantly hurts mobile user acquisition and retention. Essential for competitive parity.

## Feature Breakdown
### Phase 1: Core Recording
- Basic voice message recording and playback
- Waveform visualization
- Mobile hold-to-record interface

### Phase 2: Enhancement
- Audio transcription for accessibility
- Voice message reactions
- Playback speed controls (1.5x, 2x)

### Phase 3: Advanced Features
- Voice message threads
- Voice message search via transcription
- Cross-device playback synchronization

## Technical Considerations
- **Privacy**: Voice messages should respect E2E encryption settings
- **Storage Costs**: Implement aggressive compression and expiration policies
- **Bandwidth**: Progressive download for longer messages
- **Accessibility**: Always provide transcription options