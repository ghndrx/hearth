---
name: Native Mobile Apps - Production Deployment
description: Ship iOS and Android native apps to app stores for mobile-first user acquisition
type: critical_gap
priority: P0
---

# Native Mobile Apps - Production Deployment

## Discord Equivalent
Discord iOS/Android apps with 70%+ mobile user base, push notifications, offline-first architecture

## User Value Proposition
- **Mobile-first accessibility**: 70% of Discord users are mobile-first
- **App store presence**: Critical for mainstream discovery and acquisition
- **Native performance**: Superior UX vs web app with proper gestures, navigation
- **Push notifications**: Real-time engagement even when app closed
- **Offline capabilities**: Basic message viewing when network unavailable

## Technical Complexity Estimate
**Priority: P0** (User acquisition blocker)
**Timeline: 8-12 weeks to ship v1.0**

Current status: Architecture complete, apps 80% implemented but not shipped

## Implementation Sketch

### iOS App (Swift/SwiftUI)
- ✅ **Architecture**: Complete SwiftUI structure documented
- ✅ **Core UI**: Login, server list, channel list, message composer
- ⚠️ **WebSocket Integration**: Needs production WebSocket gateway connection  
- ⚠️ **Push Notifications**: APNs integration needed
- ⚠️ **App Store**: TestFlight → App Store submission process

### Android App (Kotlin/Jetpack Compose)  
- ✅ **Architecture**: Complete Compose structure documented
- ✅ **Core UI**: Login, navigation, message rendering
- ⚠️ **WebSocket Integration**: Needs production gateway connection
- ⚠️ **Push Notifications**: FCM integration needed  
- ⚠️ **Play Store**: Beta track → Play Store submission

### Backend Changes Required
- APNs certificate configuration for iOS push
- FCM service account for Android push  
- Mobile-optimized WebSocket opcodes (reduce bandwidth)
- Device registration endpoints for push tokens

## Dependencies
1. **WebSocket gateway** - ✅ Already mobile-compatible
2. **Push notification service** - ⚠️ Needs APNs/FCM integration
3. **App store accounts** - ⚠️ Developer accounts needed
4. **CI/CD for mobile** - ⚠️ FastLane/GitHub Actions for builds

## Success Metrics
- App store approval within 2 weeks of submission
- 1000+ app store downloads in first month
- <2s cold start time on average devices
- 95%+ crash-free sessions
- Mobile push notification open rate >15%

## Risk Mitigation
- **App store rejection**: Follow platform guidelines, test with beta users
- **Performance issues**: Profile on low-end devices, optimize critical paths
- **WebSocket connectivity**: Implement connection retry with exponential backoff