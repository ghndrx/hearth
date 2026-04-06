# PRD: Advanced Member Onboarding & Screening

**Date:** 2026-04-04  
**Status:** In Planning
**Priority:** P1
**Complexity:** Medium

## Problem Statement

While Hearth has basic welcome screens, it lacks Discord's sophisticated member onboarding flow including safety questions, role assignments, and progressive disclosure that helps communities manage growth and safety.

## Discord Equivalent

Discord's Member Screening feature with safety questions, verification requirements, and onboarding flows that communities use to filter members and provide guided setup.

## User Value Proposition

- **Community Managers**: Better tools to filter and onboard new members safely
- **Safety**: Reduce spam, raids, and low-quality members through screening
- **User Experience**: Guided onboarding reduces confusion for new members
- **Engagement**: Better initial experience leads to higher retention

## Current Implementation Analysis

**What Exists:**
- Basic welcome screens (`WelcomeScreen` model)
- Member screening API endpoints already present in routes.go
- Screening approval/rejection workflows

**What's Missing:**
- Rich onboarding form builder
- Progressive role assignment during onboarding
- Safety question templates 
- Automated screening based on account age/verification
- Onboarding analytics and funnel tracking

## Enhanced Features Needed

### 1. Rich Onboarding Form Builder
- Drag-and-drop form builder for server admins
- Multiple question types: text, multiple choice, agreement checkboxes
- Conditional logic: show questions based on previous answers
- Template library: safety templates, role selection templates

### 2. Automated Screening Rules
- Account age requirements (e.g., Discord account >30 days old)
- Phone verification requirements
- IP reputation checking
- Automatic approval for low-risk users
- Escalation to manual review for high-risk users

### 3. Progressive Role Assignment
- Assign basic roles automatically upon passing screening
- Progressive permissions: unlock features over time/activity
- Integration with server's role hierarchy
- Temporary roles during probationary periods

### 4. Safety Question Templates
- Anti-raid questions (math problems, reading comprehension)
- Community-specific questions (rules understanding, intent)
- Behavioral agreement checkboxes
- Custom safety filters based on community type

### 5. Onboarding Analytics
- Screening funnel analytics (started, completed, approved rates)
- Drop-off analysis by question type
- Member quality metrics post-onboarding
- A/B testing for onboarding flows

## Technical Implementation

### Backend Extensions
1. **Enhanced Screening Models**: Extend existing screening system with form builder data structures
2. **Automated Rules Engine**: Add rule evaluation for auto-approval/rejection  
3. **Analytics Service**: Track onboarding funnel metrics
4. **Template System**: Pre-built onboarding templates for different community types

### Frontend Components
1. **Admin Form Builder**: Rich drag-and-drop interface for creating onboarding flows
2. **Member Onboarding Flow**: Step-by-step guided experience
3. **Analytics Dashboard**: Onboarding performance metrics for admins
4. **Template Gallery**: Browse and apply onboarding templates

### Database Schema Extensions
```sql
-- Enhanced screening with form data
ALTER TABLE server_screening ADD COLUMN form_config JSON;
ALTER TABLE server_screening ADD COLUMN automation_rules JSON;

-- Screening responses from users  
CREATE TABLE screening_responses (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id),
    server_id UUID REFERENCES servers(id),
    responses JSON,
    status TEXT, -- pending, approved, rejected, escalated
    created_at TIMESTAMP
);

-- Onboarding templates
CREATE TABLE onboarding_templates (
    id UUID PRIMARY KEY, 
    name TEXT,
    description TEXT,
    form_config JSON,
    category TEXT, -- safety, role-selection, community-specific
    usage_count INTEGER DEFAULT 0
);
```

## Implementation Timeline

### Phase 1: Enhanced Form Builder (4-5 weeks)
- Rich form configuration for existing screening system  
- Extended question types and conditional logic
- Template system with safety-focused templates
- Admin UI for form building

### Phase 2: Automation & Intelligence (3-4 weeks)
- Automated screening rules engine
- Account age, verification, IP reputation checking
- Automatic approval workflows
- Escalation to manual review

### Phase 3: Analytics & Optimization (2-3 weeks)
- Onboarding funnel tracking
- Admin analytics dashboard  
- Performance metrics and optimization suggestions
- A/B testing infrastructure for onboarding flows

## Success Metrics

- **Safety Improvement**: Reduced spam/raid incidents in communities using enhanced screening
- **Member Quality**: Higher retention and engagement of screened members vs non-screened
- **Admin Adoption**: % of large servers (>500 members) using enhanced onboarding
- **Completion Rates**: Screening completion rates by template type

## Dependencies

1. **Existing Screening System**: Build upon current screening API and infrastructure
2. **User Verification**: Account age, phone verification systems  
3. **Analytics Infrastructure**: Event tracking and reporting system
4. **Template Management**: Content management for onboarding templates

## Technical Complexity: P1 (Medium)

**Estimated Timeline:** 9-12 weeks

**Risk Factors:**
- Integration complexity with existing screening system
- Performance impact of automated rule evaluation
- User experience balance between safety and friction

## Future Enhancements

- AI-powered risk assessment for automated screening
- Integration with third-party safety services
- Community-specific onboarding best practices
- Advanced behavioral analysis during probationary periods