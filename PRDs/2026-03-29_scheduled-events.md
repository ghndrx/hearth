---
name: Scheduled Events
description: Server events with RSVP, calendar integration, and event notifications
type: feature
priority: P0
---

# Scheduled Events

## Discord Equivalent
Discord's Scheduled Events feature allows server admins to create events with times, descriptions, voice/stage channel locations, and external locations. Users can RSVP and get notifications.

## User Value Proposition
- **Community Building**: Enables servers to organize gaming sessions, meetings, social events
- **Discovery**: Users can see upcoming events in their servers
- **Engagement**: RSVP system builds commitment and helps with planning
- **Integration**: Calendar integration for better time management

## Technical Complexity: P0 (High)

### Implementation Sketch
1. **Data Model**: Extend existing `event.go` model with:
   - Event scheduling (start/end times, timezone)
   - RSVP tracking (interested, attending, maybe)
   - Location types (voice channel, stage, external)
   - Recurring event patterns

2. **API Endpoints**:
   - `POST /servers/{id}/events` - Create event
   - `GET /servers/{id}/events` - List events
   - `PUT /events/{id}/rsvp` - RSVP to event
   - `GET /events/{id}/attendees` - Get attendee list

3. **UI Components**:
   - EventCalendar.svelte (month/week/day views)
   - EventCreator.svelte (creation flow)
   - EventCard.svelte (event display)
   - RSVPButton.svelte (attendance tracking)

4. **Notifications**:
   - Event reminders (15min, 1hr, 1day before)
   - RSVP notifications to organizers
   - Event updates/cancellations

### Dependencies
- Existing notification system
- Server permissions framework
- Calendar/timezone handling library

### Success Metrics
- Event creation rate per server
- RSVP conversion rate
- Event attendance rate
- User engagement with event notifications