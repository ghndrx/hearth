---
name: Custom Emoji System
description: Server and global custom emoji upload, management, and usage system
type: feature
priority: P0
complexity: Medium
dependencies: File storage, permission system
---

# Custom Emoji System

## Discord Equivalent
Discord's custom emoji system allowing servers to upload and manage custom emojis, with Nitro users able to use emojis across servers.

## User Value Proposition
- **Server Identity**: Custom emojis create unique server branding and culture
- **Expression**: Users can express themselves beyond standard Unicode emojis
- **Premium Feature**: Cross-server emoji usage drives premium subscriptions
- **Community Engagement**: Popular custom emojis become part of server identity

## Technical Complexity: P0 (Medium)
**Backend Changes:**
- Emoji upload and storage system (static/animated)
- Emoji permission and usage tracking
- Cross-server emoji resolution for premium users
- Emoji pack import/export functionality

**Frontend Changes:**
- Emoji picker with custom emoji categories
- Server emoji management interface
- Animated emoji rendering (GIF/WebP)
- Emoji shortcode autocomplete (:customemoji:)

## Implementation Sketch
1. **Storage System**:
   - Image files in S3/MinIO (<256KB limit static, <512KB animated)
   - Emoji metadata in PostgreSQL (name, server_id, creator, usage_count)

2. **Permission Model**:
   - Server permission: "Manage Emojis", "Use External Emojis"
   - Premium tier: Cross-server emoji usage
   - Rate limits: 5 uploads/hour per user, 50 emojis per server (premium: 100)

3. **API Endpoints**:
   - `POST /servers/:id/emojis` - Upload emoji
   - `GET /servers/:id/emojis` - List server emojis
   - `DELETE /servers/:id/emojis/:emojiId` - Delete emoji
   - `GET /emojis/:emojiId` - Get emoji details

4. **Frontend Components**:
   - Enhanced emoji picker with custom tabs
   - Server settings emoji management page
   - Emoji usage analytics
   - Bulk emoji import tools

## Dependencies
- File attachment system (✅ implemented)
- Premium subscription system (✅ implemented)
- Permission system (✅ implemented)
- Message rendering system (✅ implemented)

## Success Metrics
- Custom emoji uploads per server >10
- Cross-server emoji usage drives 15% premium conversion
- Message engagement with custom emojis +20%