---
name: Server Templates
description: Pre-built server configurations for easy community setup
type: feature
priority: P0
discord_equivalent: Server templates with channels, roles, and permissions
estimated_complexity: Medium
---

# Server Templates

## Discord Equivalent
Direct match to Discord's server templates system - shareable server configurations with channels, roles, and permissions.

## User Value Proposition
- **Quick setup**: Create gaming, study, or business servers instantly
- **Best practices**: Pre-configured channels and permissions
- **Community growth**: Share successful server layouts
- **Onboarding**: Reduce barrier to entry for new server creators

## Technical Complexity: P0 (Medium)
- Template creation and storage system
- Server cloning with role/permission mapping
- Template marketplace/discovery
- Incremental template updates

## Implementation Sketch

### Backend Components
1. **Database Schema**
   ```sql
   CREATE TABLE server_templates (
     id UUID PRIMARY KEY,
     name VARCHAR(100) NOT NULL,
     description TEXT,
     creator_id UUID NOT NULL REFERENCES users(id),
     source_server_id UUID REFERENCES servers(id),
     category VARCHAR(50), -- gaming, education, business, community
     tags VARCHAR(100)[],
     usage_count INTEGER DEFAULT 0,
     is_official BOOLEAN DEFAULT false,
     is_featured BOOLEAN DEFAULT false,
     created_at TIMESTAMP DEFAULT NOW(),
     updated_at TIMESTAMP DEFAULT NOW()
   );

   CREATE TABLE template_data (
     template_id UUID PRIMARY KEY REFERENCES server_templates(id),
     server_config JSONB NOT NULL, -- server settings, icon, banner
     channels_config JSONB NOT NULL, -- channel structure and permissions
     roles_config JSONB NOT NULL, -- roles and permissions
     integrations_config JSONB DEFAULT '{}' -- webhooks, bots, etc
   );

   CREATE TABLE template_usage (
     id UUID PRIMARY KEY,
     template_id UUID NOT NULL REFERENCES server_templates(id),
     user_id UUID NOT NULL REFERENCES users(id),
     server_id UUID NOT NULL REFERENCES servers(id),
     created_at TIMESTAMP DEFAULT NOW()
   );
   ```

2. **API Endpoints**
   - `GET /templates` - Browse templates with filtering
   - `POST /servers/{id}/template` - Create template from server
   - `POST /templates/{id}/use` - Create server from template
   - `PUT /templates/{id}` - Update template (creator only)
   - `GET /templates/{id}/preview` - Preview template structure

### Frontend Components
1. **TemplateExplorer.svelte** - Browse template marketplace
2. **TemplateCreator.svelte** - Create template from current server
3. **TemplatePreview.svelte** - Preview template before use
4. **ServerSetupWizard.svelte** - Enhanced server creation with templates

### Key Features
1. **Template Creation**
   - Generate from existing server
   - Select which channels/roles to include
   - Add description and tags
   - Set template visibility (public/private)

2. **Template Marketplace**
   - Browse by category (Gaming, Study, Business, etc.)
   - Search by tags and keywords
   - Official templates by Hearth team
   - Community-created templates

3. **Template Application**
   - Preview template structure before use
   - Customize server name and region
   - Automatic role and permission setup
   - Channel category and ordering preservation

4. **Template Management**
   - Update templates when source server changes
   - Usage analytics for creators
   - Report inappropriate templates
   - Version control for template updates

### Template Categories
- **Gaming**: Voice channels for different games, LFG channels, tournament setup
- **Study Groups**: Subject channels, voice study rooms, resource sharing
- **Business**: Department channels, announcement systems, meeting rooms
- **Community**: General discussion, events, introductions
- **Creative**: Art sharing, feedback channels, collaboration spaces

## Dependencies
- [ ] Server creation working (✅ implemented)
- [ ] Role and permission system (✅ implemented)
- [ ] Channel management (✅ implemented)
- [ ] Server discovery system (✅ implemented)

## Success Metrics
- Template usage rate on new servers > 60%
- Community template creation > 100 templates
- Template-created servers have higher retention (+20%)
- Reduced time-to-first-message for new servers

## Implementation Timeline
- Phase 1: Basic template creation and usage (3 weeks)
- Phase 2: Template marketplace and discovery (2 weeks)
- Phase 3: Official template collection (1 week)
- Phase 4: Advanced features and analytics (2 weeks)