---
name: Message Scheduling
description: Schedule messages to be sent at specific times with timezone support
type: feature
priority: P1
---

# Message Scheduling

## Discord Equivalent
Message scheduling allows users to write messages that are automatically sent at a specified future time, useful for announcements, reminders, and cross-timezone communication.

## User Value Proposition
- **Timezone Management**: Send messages when recipients are online
- **Announcements**: Schedule important server announcements
- **Productivity**: Plan communication in advance
- **Automation**: Reduce manual overhead for regular updates

## Technical Complexity: P1 (Medium)

### Implementation Sketch
1. **Data Model**: New scheduled message model:
   - Scheduled time (with timezone)
   - Message content (text, embeds, attachments)
   - Target channel/user
   - Status (pending, sent, cancelled, failed)
   - Author and permissions

2. **Scheduling Engine**:
   - Background job processor (Redis-based queue)
   - Timezone handling and validation
   - Retry logic for failed sends
   - Bulk scheduling capabilities
   - Permission validation at send time

3. **API Endpoints**:
   - `POST /channels/{id}/schedule` - Schedule message
   - `GET /users/@me/scheduled` - List user's scheduled messages
   - `DELETE /scheduled/{id}` - Cancel scheduled message
   - `PUT /scheduled/{id}` - Edit scheduled message

4. **UI Components**:
   - ScheduleMessageModal.svelte (scheduling interface)
   - DateTimePicker.svelte (date/time selection)
   - ScheduledMessagesList.svelte (manage scheduled messages)
   - ScheduleButton.svelte (trigger scheduling)

5. **Permissions & Limits**:
   - Same permissions required as regular message sending
   - Rate limits on scheduled message creation
   - Maximum scheduling window (e.g., 1 year)
   - Cleanup of old scheduled messages

### Dependencies
- Background job queue system
- Timezone handling library
- Existing message sending pipeline
- Permission system

### Success Metrics
- Scheduled message creation rate
- Successfully sent scheduled messages
- User adoption of scheduling feature
- Cancellation rate (indicator of utility)