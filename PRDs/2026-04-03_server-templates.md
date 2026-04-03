---
name: Server Templates
description: Server setup templates for quick community creation
type: feature
priority: P1
complexity: Medium
dependencies: Server creation, permission system, channel management
---

# Server Templates

## Discord Equivalent
Discord's server templates allow users to create pre-configured servers with channels, roles, and permissions based on successful community blueprints.

## User Value Proposition
- **Faster Onboarding**: New server creators get working setup in minutes vs hours
- **Best Practices**: Templates embody proven community structures and moderation
- **Reduced Friction**: Lower barrier to entry for non-technical community creators
- **Community Growth**: More successful servers = larger user base

## Technical Complexity: P1 (Medium)
**Backend Changes:**
- Server template storage and versioning system
- Template creation from existing servers (snapshot)
- Template instantiation with customization options
- Template sharing and discovery
- Usage analytics and popularity tracking

**Frontend Changes:**
- Template browser with categories and previews
- Create server from template flow
- Template creation wizard from existing server
- Template customization options (name, description, icon)
- Popular/featured templates showcase

## Implementation Sketch
1. **Database Schema**:
   - server_templates table (id, name, description, creator_id, usage_count)
   - template_data JSONB (channels, roles, permissions, settings)
   - template_categories table for organization
   - template_usage_stats for analytics

2. **Template Categories**:
   - Gaming, Study Groups, Art/Creative, Tech, Business
   - Music, Roleplay, Anime/Manga, Sports, General

3. **Template Data**:
   - Channel structure with categories and permissions
   - Role hierarchy and permissions
   - Server settings (verification level, content filter)
   - Welcome message templates

4. **API Endpoints**:
   - `GET /guilds/templates` - Browse available templates
   - `POST /guilds/templates/:code` - Create server from template
   - `POST /guilds/:id/templates` - Create template from server
   - `GET /guilds/templates/:code` - Get template details

## Dependencies
- Server creation system (✅ implemented)
- Channel management (✅ implemented)
- Role/permission system (✅ implemented)
- JSON schema validation (❌ needs implementation)

## Success Metrics
- Template usage for new servers >45%
- Server creation completion rate +35% with templates
- Average channels created per new server +60%
- Template-created servers have 25% better 7-day retention