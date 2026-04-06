---
name: Custom Status & Rich Presence
description: User custom status messages, activities, and rich presence integration
type: feature
priority: P1
discord_equivalent: Custom status, activity status, rich presence API
estimated_complexity: Medium
---

# Custom Status & Rich Presence

## Discord Equivalent
Direct match to Discord's custom status messages, activity detection, and rich presence API for games/applications.

## User Value Proposition
- **Personal expression**: Share mood, availability, or current activity
- **Social discovery**: See what friends are playing/doing
- **Gaming integration**: Rich presence for games with play buttons
- **Professional use**: Status for work availability and focus time

## Technical Complexity: P1 (Medium)
- User status management system
- Activity detection (optional)
- Rich presence API for third parties
- Real-time status broadcasting
- Mobile status sync

## Implementation Sketch

### Backend Components
1. **Database Schema**
   ```sql
   CREATE TABLE user_statuses (
     user_id UUID PRIMARY KEY REFERENCES users(id),
     status_type VARCHAR(20) DEFAULT 'online', -- online, idle, dnd, invisible
     custom_text VARCHAR(128),
     custom_emoji VARCHAR(10),
     expires_at TIMESTAMP,
     updated_at TIMESTAMP DEFAULT NOW()
   );

   CREATE TABLE user_activities (
     id UUID PRIMARY KEY,
     user_id UUID NOT NULL REFERENCES users(id),
     activity_type VARCHAR(30) NOT NULL, -- game, streaming, listening, watching, custom
     name VARCHAR(128) NOT NULL,
     details VARCHAR(128),
     state VARCHAR(128),
     large_image VARCHAR(256),
     small_image VARCHAR(256),
     start_timestamp TIMESTAMP,
     created_at TIMESTAMP DEFAULT NOW()
   );

   CREATE TABLE rich_presence_applications (
     id VARCHAR(50) PRIMARY KEY, -- Discord application ID format
     name VARCHAR(100) NOT NULL,
     icon VARCHAR(256),
     verified BOOLEAN DEFAULT false,
     created_at TIMESTAMP DEFAULT NOW()
   );
   ```

2. **API Endpoints**
   - `PUT /users/@me/status` - Set custom status
   - `POST /users/@me/activities` - Update activity
   - `GET /users/{id}/presence` - Get user presence
   - `POST /applications/{id}/rpc` - Rich presence webhook

3. **WebSocket Events**
   - `PRESENCE_UPDATE` - Broadcast status changes
   - `ACTIVITY_UPDATE` - Real-time activity changes

### Frontend Components
1. **StatusPicker.svelte** - Status selection modal
2. **UserPresence.svelte** - Display user status/activity
3. **ActivityCard.svelte** - Rich activity display
4. **StatusSettings.svelte** - Status preferences

### Key Features
1. **Custom Status**
   - Text message up to 128 characters
   - Optional emoji (custom or Unicode)
   - Expiration timer (1h, 4h, today, don't clear)
   - Status type (online, idle, do not disturb, invisible)

2. **Activity Detection**
   - Currently playing games (desktop client)
   - Streaming detection (OBS, XSplit)
   - Spotify/music integration
   - Custom activities

3. **Rich Presence**
   - Game/app details with images
   - Current state ("In Match", "Main Menu")
   - Timer support (elapsed/remaining)
   - Join/spectate buttons (if supported)

4. **Privacy Controls**
   - Hide activity from specific users
   - Disable activity detection globally
   - Status visibility per server

## Dependencies
- [ ] User presence system (basic online/offline)
- [ ] WebSocket real-time updates (✅ implemented)
- [ ] Desktop application for activity detection
- [ ] OAuth2 integration for third-party apps

## Success Metrics
- Custom status adoption rate > 40%
- Activity detection accuracy > 90%
- Rich presence API usage by 5+ applications
- Real-time status update latency < 2 seconds

## Implementation Timeline
- Phase 1: Basic custom status (2 weeks)
- Phase 2: Activity detection for games (3 weeks)
- Phase 3: Rich presence API (2 weeks)
- Phase 4: Mobile app integration (2 weeks)