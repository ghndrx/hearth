# Community Onboarding & Welcome System

## Discord Equivalent
Discord's Welcome Screen, Server Guide, Rules Screen, and Member Screening features

## User Value Proposition
- **Server Owners**: Reduce moderation burden and improve new member experience
- **New Members**: Clear understanding of community rules and culture
- **Communities**: Higher quality members and reduced churn

## Technical Complexity Estimate
**P1** - Important for community quality but moderate implementation complexity

## Implementation Sketch

### Core Components
```
Models:
- WelcomeConfig (server_id, enabled, welcome_channels, rules_text, screening_enabled)
- ServerRules (server_id, rules, acceptance_required, updated_at)
- MemberScreening (server_id, questions, verification_level, auto_approve)
- OnboardingStep (server_id, step_type, content, required, order)
- MemberOnboarding (user_id, server_id, completed_steps, status, started_at)

Welcome Types:
- Welcome Screen (overview, rules, channels)
- Member Screening (questions, verification)
- Role Selection (self-assignable roles)
- Channel Introduction Tour
- Community Guidelines Quiz
```

### Welcome Screen Features
- **Server Description**: Rich text overview of the community
- **Featured Channels**: Highlight important channels with descriptions
- **Server Rules**: Clear, readable rules with acknowledgment requirement
- **Visual Branding**: Custom banners, colors, and server theme
- **Quick Actions**: Join roles, follow channels, enable notifications

### Member Screening System
- **Custom Questions**: Server-specific screening questions
- **Verification Requirements**: Phone/email verification
- **Auto-Approval**: Automatic approval based on criteria
- **Manual Review**: Moderator review queue for applications
- **Rejection Handling**: Customizable rejection messages

### Onboarding Flow
- **Progressive Disclosure**: Step-by-step introduction to features
- **Interactive Tutorial**: Guided tour of server layout and features
- **Role Assignment**: Self-service role selection interface
- **Channel Introductions**: Automated welcome messages in key channels
- **Buddy System**: Pair new members with welcomers/mentors

### Community Guidelines
- **Rules Editor**: Rich text editor for server rules
- **Rules Versioning**: Track changes and re-acceptance requirements
- **Acknowledgment Tracking**: Record when users accept rules
- **Violation Reference**: Link moderation actions to specific rules
- **Multi-Language**: Support for multiple rule languages

### Analytics & Insights
- **Onboarding Completion Rates**: Track drop-off points
- **Member Retention**: Correlation with onboarding completion
- **Question Effectiveness**: Which screening questions work best
- **Welcome Message Engagement**: Interaction with welcome content

---

## Invite Link System

### Core Invite Features
- **Standard Invite Links**: Generated 8-12 character codes, URL-safe base64
- **Channel-Scoped Invites**: Links optionally target a specific channel as landing point
- **Configurable Expiry**: 30min, 1hr, 6hr, 12hr, 1day, 7days, or never
- **Max Uses**: Single-use (for exclusive invites) or unlimited
- **Vanity URLs** (Pro/Premium): Custom short slugs (`hearth.gg/mycommunity`) for branded sharing
- **QR Code Generation**: Mobile-friendly scannable codes for any invite link
- **Invite Revocation**: Admin UI to instantly invalidate any active invite

### Invite Data Model
```
Invite:
  - id: UUID
  - code: string (unique, indexed)
  - server_id: UUID
  - channel_id: UUID (optional, null = server-level invite)
  - created_by: UUID
  - max_uses: int (0 = unlimited)
  - used_count: int
  - expires_at: timestamp (null = never)
  - revoked_at: timestamp (null = active)
  - created_at: timestamp
```

### Invite Analytics
- Track which invites bring the most joins
- Attribution: show who created each invite, joins attributed to inviter
- "Friend joined" notification when a contact joins a shared server
- Invite creator can see aggregate join stats without seeing individual users

### Invite Link UX
- One-tap copy with "Copied!" toast confirmation
- Expiry/uses remaining shown inline in share UI
- Graceful expiry: friendly message + link to request new invite when expired
- "Invite people to this channel" right-click/hover action on any channel

---

## Server Discovery & Public Directory

### Discovery Eligibility
Servers must meet minimum thresholds to appear in public directory:
- Minimum 50 members
- At least 10 members active in past 7 days
- Server description (minimum 50 characters)
- Community feature enabled (rules channel configured)

### Directory Entry
```
DiscoveryEntry:
  - server_id: UUID
  - name: string
  - icon_url: string
  - banner_url: string
  - description: string (50-512 chars)
  - category: enum (Gaming, Music, Technology, Art, Science, Lifestyle, Other)
  - tags: string[] (up to 5 custom tags)
  - member_count: int
  - online_count: int
  - created_at: timestamp
  - featured: boolean (staff-curated)
```

### Discovery Features
- **Category Browsing**: Flat list of categories, scrollable server cards
- **Keyword Search**: Name, description, and tag matching
- **Pre-Join Preview**: Full server name, icon, banner, member/online count, description, rules preview, channel list, tags
- **Server Cards**: Icon, name, member+online count, description snippet, top tags
- **Server Detail Modal**: Full preview before joining
- **Mutual Friend Indicators**: "X friends are in this server" on discovery and invite pages
- **Reporting**: Users can report servers from discovery view

### Discovery UX Principles
- Discovery that shows low-quality/abandoned servers destroys trust
- Minimum activity thresholds filter out dead servers
- Pre-join preview reduces post-join regret and spam invite complaints
- Show member growth trend, not just raw count

### Cross-Instance Federation (Future)
If Hearth runs federated instances, discovery directories federate via:
- `/_hearth/public_servers?instance=remote.hearth.example`
- Each instance publishes its public servers
- Aggregators crawl across instances respecting opt-out settings

---

## Pre-Join Preview

Shown when clicking any invite link or viewing a server in discovery:

1. **Header**: Server icon + banner image, name, online/member count
2. **Description**: Server's own description text
3. **Rules Preview**: First 2-3 rules or rules channel pinned messages
4. **Channel List**: Visible channels with topic/description
5. **Tags**: Category + custom tags
6. **Social Proof**: "X members joined in the past week", "X friends are in this server"
7. **Join CTA**: Prominent "Join Server" button

After joining:
1. "Welcome to [Server]!" toast confirmation
2. Optional rules acknowledgement modal (if configured)
3. Land in invite's target channel (or server default)
4. Pinned messages shown
5. Brief tour tooltip for first-time members (optional per-server config)

## Dependencies
- Rich text editor components
- Multi-step form system
- Advanced permission system for screening roles
- Analytics infrastructure for tracking completion
- Notification system for onboarding reminders

## Success Metrics
- New member completion rate of onboarding
- Member retention at 7, 30, 90 days post-join
- Reduction in moderation actions for new members
- Server owner adoption of welcome features
- Time to first meaningful interaction for new members