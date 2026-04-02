---
name: Premium Subscription System
description: Tiered premium subscriptions with enhanced features for sustainable monetization
type: Feature PRD
priority: P0
---

# Premium Subscription System (Hearth+)

## Discord Equivalent
Direct 1:1 match with Discord Nitro system:
- Individual premium subscriptions (Basic/Classic tiers)
- Server boosting system
- Premium perks and enhanced limits
- Subscription management and billing
- Gift subscriptions and promotions

## User Value Proposition
**Critical business sustainability feature** - Premium subscriptions provide:
- **Enhanced Limits**: Higher file uploads, longer messages, more servers
- **Quality of Life**: Better emoji, stickers, custom status, priority support
- **Server Benefits**: Boost favorite communities with premium features
- **Exclusivity**: Special badges, early access, premium-only features
- **Supporting Platform**: Users pay to support open-source Discord alternative

## Technical Complexity: P0 (Business Critical, High Complexity)

### Implementation Sketch
```go
// Models
type Subscription struct {
    ID                string     `json:"id" db:"id"`
    UserID            string     `json:"user_id" db:"user_id"`
    PlanID            string     `json:"plan_id" db:"plan_id"`
    Status            string     `json:"status" db:"status"` // ACTIVE, CANCELED, PAST_DUE
    PeriodStart       time.Time  `json:"period_start" db:"period_start"`
    PeriodEnd         time.Time  `json:"period_end" db:"period_end"`
    CancelAtPeriodEnd bool       `json:"cancel_at_period_end" db:"cancel_at_period_end"`
    PaymentMethodID   string     `json:"payment_method_id" db:"payment_method_id"`
    StripeSubID       string     `json:"stripe_subscription_id" db:"stripe_subscription_id"`
    CreatedAt         time.Time  `json:"created_at" db:"created_at"`
    UpdatedAt         time.Time  `json:"updated_at" db:"updated_at"`
}

type SubscriptionPlan struct {
    ID          string            `json:"id" db:"id"`
    Name        string            `json:"name" db:"name"`
    Description string            `json:"description" db:"description"`
    Price       int               `json:"price" db:"price"` // cents per month
    Currency    string            `json:"currency" db:"currency"`
    Features    map[string]interface{} `json:"features" db:"features"`
    StripePlanID string           `json:"stripe_plan_id" db:"stripe_plan_id"`
    Active      bool              `json:"active" db:"active"`
    SortOrder   int               `json:"sort_order" db:"sort_order"`
}

type ServerBoost struct {
    ID         string    `json:"id" db:"id"`
    ServerID   string    `json:"server_id" db:"server_id"`
    UserID     string    `json:"user_id" db:"user_id"`
    SlotNumber int       `json:"slot_number" db:"slot_number"` // User can have 2 boost slots
    CreatedAt  time.Time `json:"created_at" db:"created_at"`
    ExpiresAt  time.Time `json:"expires_at" db:"expires_at"`
}

type ServerBoostTier struct {
    ServerID         string    `json:"server_id" db:"server_id"`
    Level           int       `json:"level" db:"level"` // 0, 1, 2, 3
    BoostCount      int       `json:"boost_count" db:"boost_count"`
    RequiredBoosts  int       `json:"required_boosts" db:"required_boosts"`
    Features        []string  `json:"features" db:"features"`
    UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}
```

### Premium Plans
1. **Hearth+ Basic ($2.99/month)**
   - Custom emoji usage across servers
   - 50MB file uploads (vs 8MB free)
   - HD video streaming
   - Custom tag (#0001-#9999)
   - Priority customer support
   - Early access to new features

2. **Hearth+ Premium ($9.99/month)**
   - All Basic features
   - 100MB file uploads
   - Server boost slots (2 per subscription)
   - Premium sticker packs access
   - Custom status with emoji
   - HD screen sharing
   - Higher quality voice (256kbps)
   - Message editing history

3. **Server Boosts ($4.99/month each)**
   - Boost favorite servers to unlock features
   - Higher audio quality for all members
   - Increased emoji slots (50 → 100 → 150 → 250)
   - Custom server banner
   - Vanity invite URLs
   - Upload limit increases for all members

### Key Features
1. **Subscription Management**
   - Stripe integration for billing
   - Multi-currency support
   - Payment method management
   - Automatic renewal and cancellation
   - Prorated upgrades/downgrades
   - Gift subscription system

2. **Feature Gating**
   - Real-time premium feature checks
   - Graceful degradation when subscription lapses
   - Usage tracking and limit enforcement
   - Premium badge display
   - Early access feature flags

3. **Server Boosting**
   - Boost slot assignment and tracking
   - Server boost level calculation
   - Progressive feature unlocking
   - Boost gifting system
   - Boost analytics for server owners

4. **Premium Store**
   - Premium sticker pack marketplace
   - Profile customization options
   - Exclusive themes and cosmetics
   - Digital collectibles (future)

### Enhanced Limits & Features
```yaml
Free Users:
  - File Upload: 8MB
  - Servers: 100
  - Message Length: 2000 characters
  - Custom Emoji: Server only
  - Voice Quality: 96kbps

Premium Basic:
  - File Upload: 50MB
  - Servers: 200
  - Message Length: 4000 characters
  - Custom Emoji: Cross-server usage
  - Voice Quality: 128kbps

Premium Plus:
  - File Upload: 100MB
  - Servers: 200
  - Message Length: 4000 characters
  - All Premium Basic features
  - Voice Quality: 256kbps
  - Premium Stickers: Full access
  - Server Boosts: 2 slots included
```

### API Endpoints
- `GET /subscriptions/plans` - List available plans
- `POST /subscriptions` - Create subscription
- `GET /users/@me/subscription` - Get current subscription
- `PUT /subscriptions/{id}` - Update subscription
- `DELETE /subscriptions/{id}` - Cancel subscription
- `POST /servers/{id}/boost` - Boost server
- `GET /store/stickers` - Premium sticker marketplace

## Dependencies
1. **Payment Processing** - Stripe integration for billing
2. **Feature Flag System** - Real-time premium feature gating
3. **User Management Enhancement** - Subscription status tracking
4. **Email Service** - Billing notifications and receipts
5. **Analytics System** - Subscription metrics and churn analysis

## Success Metrics
- Monthly recurring revenue (MRR)
- Subscription conversion rate (free → premium)
- Churn rate and retention curves
- Average revenue per user (ARPU)
- Server boost adoption rate
- Feature usage by subscription tier

## Timeline Estimate
- **Phase 1** (4 weeks): Core subscription system + Stripe integration
- **Phase 2** (3 weeks): Premium features implementation + limits
- **Phase 3** (3 weeks): Server boosting system
- **Phase 4** (2 weeks): Premium marketplace + advanced features

**Total: 12 weeks for complete premium system**

## Revenue Projections
**Conservative estimates for open-source Discord alternative:**
- 10,000 daily active users → 500 premium subscribers (5% conversion)
- Average plan price: $6/month
- Monthly revenue: $3,000
- Annual revenue: $36,000 (sustainable for small team)

**Growth trajectory:**
- Year 1: $50,000 ARR
- Year 2: $150,000 ARR
- Year 3: $500,000+ ARR