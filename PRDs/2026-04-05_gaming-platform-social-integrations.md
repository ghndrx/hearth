# PRD: Gaming Platform Social Integrations

**Date:** 2026-04-05
**Status:** Proposed  
**Priority:** P1
**Complexity:** High

## Problem Statement

While Hearth has rich presence and activity tracking, it lacks deep social integration with gaming platforms (Steam, Xbox Live, PlayStation Network, Battle.net) that would enable cross-platform friend discovery and gaming-focused community building.

## Discord Equivalent

Discord's Connections feature allows linking Steam, Xbox, PlayStation, and other gaming accounts for friend discovery, game library sharing, and rich social features that make Discord the default choice for gaming communities.

## User Value Proposition

- **Gaming Community Growth**: Enable organic community discovery through shared games and platforms
- **Cross-Platform Social Graph**: Connect with friends across gaming platforms within Hearth servers
- **User Acquisition**: Attract gaming communities through superior gaming platform integration
- **Competitive Differentiation**: Exceed Discord's gaming integration depth

## Technical Requirements

### Core Features
1. **Platform Authentication**: OAuth integration with Steam, Xbox Live, PSN, Battle.net, Epic Games
2. **Friend Discovery**: Import gaming friends and suggest Hearth servers/users with shared games
3. **Game Library Integration**: Display owned games, currently playing status across platforms  
4. **Gaming-Focused Server Discovery**: Recommend servers based on game libraries and preferences
5. **Cross-Platform Party Creation**: Form gaming groups across different platforms within Hearth

### Advanced Features
1. **Achievement Sharing**: Sync and display gaming achievements in user profiles
2. **Game Time Analytics**: Cross-platform gaming time tracking and insights
3. **Gaming Event Integration**: Link gaming events with Hearth community events
4. **Platform-Specific Features**: Steam Workshop integration, Xbox Game Pass tracking

## Implementation Sketch

**Backend (8-10 weeks):**
- OAuth providers for gaming platforms
- Social graph analysis service
- Game library sync and matching algorithms
- Privacy controls for gaming data sharing

**Frontend (6-8 weeks):**
- Account connections settings page
- Gaming-focused friend finder
- Enhanced user profiles with gaming data
- Game-based server discovery interface

**Mobile (4-6 weeks):**
- Gaming platform authentication
- Friend suggestion notifications
- Gaming status integration

## Dependencies

- User consent and privacy framework
- Enhanced user profile system
- Server discovery algorithm improvements
- Friend/social system enhancements

## Success Metrics

- Gaming platform connection adoption rate >40% within 6 months
- Server discovery through gaming connections increases by 200%
- Gaming community user retention +25%
- Cross-platform gaming session creation +300%

## Competitive Analysis

**Discord Strengths:** Basic account linking, game detection
**Opportunity:** Deep social integration, advanced friend discovery, cross-platform party features
**Risk:** Gaming platforms may limit API access or change terms