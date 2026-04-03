# NSFW Content & Safety System

**Feature Name**: Advanced Content Safety & NSFW Controls
**Discord Equivalent**: NSFW channel marking, content warnings, safety settings
**Priority**: P1 - High Priority
**Estimated Complexity**: 6-8 weeks

## Problem Statement

Discord's NSFW (Not Safe For Work) content system is essential for community safety and legal compliance. Users expect robust content filtering, age verification, and safety controls to protect minors and maintain appropriate environments.

## User Value Proposition

- **Safety First**: Protect minors from inappropriate content through age verification
- **Community Standards**: Enable communities to clearly mark and segregate mature content
- **Legal Compliance**: Meet platform liability requirements for content hosting
- **User Control**: Give users granular control over content they're exposed to

## Current State

Hearth lacks comprehensive NSFW content controls:
- No NSFW channel marking system
- No age verification for mature content access
- Limited content filtering capabilities
- No user safety preferences

## Proposed Solution

### Core Features

1. **NSFW Channel Controls**
   - Channel-level NSFW marking by moderators
   - Age gate requiring birthdate verification
   - Visual indicators and warnings before access
   - Default hiding of NSFW channels from users under 18

2. **Content Safety Filters**
   - Automatic image/video content scanning (ML-based)
   - Text content analysis for explicit material
   - User-configurable safety filter levels (off/medium/high)
   - Override controls for channel moderators

3. **User Safety Settings**
   - Account age verification (birthdate)
   - Safety filter preferences in user settings
   - NSFW content opt-in requirement for new accounts
   - Parental controls for accounts under 18

4. **Moderation Tools**
   - Bulk NSFW marking for channels
   - Content reporting and appeals system
   - Automated content removal for policy violations
   - Audit trail for safety actions

## Technical Implementation

### Database Schema
```sql
-- Channel NSFW marking
ALTER TABLE channels ADD COLUMN nsfw BOOLEAN DEFAULT FALSE;

-- User safety preferences
ALTER TABLE users ADD COLUMN birthdate DATE;
ALTER TABLE users ADD COLUMN safety_filter_level VARCHAR(20) DEFAULT 'medium';
ALTER TABLE users ADD COLUMN nsfw_access_enabled BOOLEAN DEFAULT FALSE;

-- Content safety tracking
CREATE TABLE content_safety_logs (
  id BIGSERIAL PRIMARY KEY,
  content_type VARCHAR(50) NOT NULL, -- 'message', 'attachment', 'emoji'
  content_id BIGINT NOT NULL,
  safety_action VARCHAR(50) NOT NULL, -- 'flagged', 'removed', 'approved'
  confidence_score FLOAT,
  reviewer_id BIGINT REFERENCES users(id),
  created_at TIMESTAMP DEFAULT NOW()
);
```

### API Endpoints
- `PUT /channels/:id/nsfw` - Mark/unmark channel as NSFW
- `GET /users/@me/safety` - Get user safety settings
- `PUT /users/@me/safety` - Update safety preferences
- `POST /content/report` - Report inappropriate content

### Content Scanning Integration
- Integrate with AWS Rekognition or similar for image analysis
- Text analysis using trained models for explicit content detection
- Configurable thresholds for automatic actions

## Dependencies

1. **Age Verification System** - User birthdate collection and validation
2. **Content Scanning Service** - ML models for automated detection
3. **Moderation Dashboard** - UI for safety management tools
4. **Appeal System** - Process for users to contest safety actions

## Success Metrics

- NSFW content properly gated behind age verification
- Reduction in inappropriate content reports
- User safety satisfaction scores
- Compliance with platform safety standards

## Technical Complexity Assessment

**P1 Priority** - Essential for community safety and legal compliance

**Implementation Effort**: 6-8 weeks
- Week 1-2: Database schema and basic NSFW marking
- Week 3-4: Content scanning integration
- Week 5-6: User safety settings and age verification
- Week 7-8: Moderation tools and appeals process

## Business Impact

- **Risk Mitigation**: Reduces legal liability for inappropriate content
- **User Trust**: Demonstrates commitment to safety and appropriate boundaries
- **Community Health**: Enables healthy community standards and expectations
- **Competitive Parity**: Matches Discord's safety feature set

This system is essential for any platform hosting user-generated content and community discussions.