# PRD: Discord Nitro Feature Parity

**Date:** 2026-04-04
**Status:** In Planning  
**Priority:** P1
**Complexity:** Medium

## Problem Statement

While Hearth has a premium subscription system, it lacks several key Discord Nitro features that power users expect, potentially limiting subscription value and user retention.

## Discord Equivalent

Discord Nitro ($9.99/month) and Nitro Basic ($2.99/month) with specific feature sets that create clear subscription value.

## User Value Proposition

- **Premium Users**: Expect feature parity with Discord Nitro for file uploads, emoji, streaming quality
- **Revenue Growth**: Increase subscription conversion by offering comparable value
- **User Retention**: Prevent churn to Discord for Nitro-exclusive features

## Missing Nitro Features Analysis

### Currently Missing (High Impact):
1. **Enhanced File Upload Limits**
   - Discord Nitro: 500MB uploads vs 25MB free
   - Current Status: Unknown limits in Hearth

2. **HD Video Streaming/Screen Share**
   - Discord Nitro: 1080p 60fps streaming vs 720p free
   - Current Status: Video calling exists but quality tiers unclear

3. **Custom Profile Themes/Banners** 
   - Discord Nitro: Animated avatars, custom profile colors, banners
   - Current Status: Basic avatar/banner support exists

4. **Server Profile Customization**
   - Discord Nitro: Per-server avatars and nicknames
   - Current Status: Unknown if implemented

### Currently Missing (Medium Impact):
1. **Longer Messages** 
   - Discord Nitro: 4000 character limit vs 2000 free
   - Current Status: Unknown message limits

2. **Enhanced Emoji Features**
   - Discord Nitro: Use emojis from any server, animated emojis
   - Current Status: Custom emoji system exists

3. **Advanced Screen Share**
   - Discord Nitro: Application-specific screen share, higher quality
   - Current Status: Basic screen share implemented

## Technical Requirements

### Subscription Tiers
```
Hearth Basic ($2.99/month):
- 100MB file uploads (vs 25MB free)
- HD voice quality
- Custom emoji usage across servers
- Basic profile customization

Hearth Premium ($9.99/month):
- 500MB file uploads
- 1080p 60fps video/screen share
- Animated avatars & banners
- Per-server profiles
- 4000 character messages
- Priority support
```

### Backend Implementation
1. **File Upload Service**: Implement tiered upload limits based on subscription
2. **Video Quality Service**: Subscription-based quality tiers for video/screen share
3. **Profile Service**: Extended profile customization with subscription checks
4. **Message Service**: Variable character limits based on subscription tier

### Frontend Implementation
1. **Upload UI**: Show remaining file size based on tier
2. **Video Controls**: Quality selector for premium users
3. **Profile Editor**: Enhanced customization options for subscribers
4. **Subscription Upsell**: In-context upgrade prompts

## Implementation Sketch

### Phase 1: Core Premium Features (4-6 weeks)
1. Enhanced file upload limits with subscription checking
2. HD video streaming quality tiers
3. Cross-server emoji usage for premium users
4. Basic subscription upsell UI

### Phase 2: Advanced Customization (4-6 weeks) 
1. Animated avatar support
2. Custom profile banners and themes
3. Per-server profile customization
4. Extended message length limits

### Phase 3: Polish & Analytics (2-3 weeks)
1. Subscription analytics and conversion tracking
2. Feature usage metrics per tier
3. A/B testing for upsell messaging
4. Customer feedback integration

## Success Metrics

- **Conversion Rate**: % of free users upgrading to premium tiers
- **Feature Adoption**: % of premium users using enhanced features
- **Churn Reduction**: Decreased user migration to Discord for Nitro features
- **Revenue Impact**: Monthly recurring revenue growth from enhanced tiers

## Dependencies

1. **Payment System**: Existing Stripe integration for tiered billing
2. **File Storage**: CDN/storage scaling for larger file uploads
3. **Video Infrastructure**: Quality tier support in streaming system
4. **Analytics**: Subscription feature usage tracking

## Technical Complexity: P1 (Medium)

**Estimated Timeline:** 10-15 weeks total

**Risk Factors:**
- Video quality streaming infrastructure requirements
- File storage cost implications for large uploads
- Complex subscription entitlement checking across all features

## Future Enhancements

- Advanced server boosting system
- Premium-only Discord integration features
- Early access to experimental features
- Premium customer support channels