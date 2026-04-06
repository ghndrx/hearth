---
name: Server Insights Dashboard
description: Analytics and metrics dashboard for server owners to track community health and growth
type: feature
priority: P1
complexity: Medium
dependencies: Analytics service, database aggregation, permissions system
---

# Server Insights Dashboard

## Discord Equivalent
Discord's Server Insights feature providing server owners and administrators with detailed analytics about member activity, channel performance, community growth, and engagement metrics.

## User Value Proposition
- **Community Management**: Data-driven decisions for server growth and engagement
- **Content Strategy**: Understanding which channels and content drive engagement
- **Moderation Insights**: Identifying problematic patterns and successful interventions
- **Growth Tracking**: Monitor server health and identify growth opportunities
- **Premium Justification**: Advanced analytics can drive server boost subscriptions

## Technical Complexity: P1 (Medium)
**Backend Changes:**
- Analytics data collection and aggregation system
- Time-series database for metrics storage
- Permission-based access to server insights
- Real-time and historical data processing
- Privacy-compliant data anonymization

**Frontend Changes:**
- Server insights dashboard with charts and metrics
- Exportable reports and data visualization
- Date range filtering and metric comparison
- Member engagement analysis tools
- Channel performance analytics

**Database Schema:**
```sql
-- Server metrics aggregation
CREATE TABLE server_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id UUID REFERENCES servers(id),
    metric_type VARCHAR NOT NULL, -- 'members', 'messages', 'voice_time', etc.
    metric_date DATE NOT NULL,
    metric_value BIGINT NOT NULL,
    metadata JSONB, -- Additional context
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(server_id, metric_type, metric_date)
);

-- Channel analytics
CREATE TABLE channel_analytics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    channel_id UUID REFERENCES channels(id),
    date DATE NOT NULL,
    message_count INTEGER DEFAULT 0,
    unique_authors INTEGER DEFAULT 0,
    reaction_count INTEGER DEFAULT 0,
    voice_minutes INTEGER DEFAULT 0, -- For voice channels
    thread_count INTEGER DEFAULT 0, -- For forum/text channels
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(channel_id, date)
);

-- Member activity tracking
CREATE TABLE member_activity (
    user_id UUID,
    server_id UUID,
    date DATE,
    messages_sent INTEGER DEFAULT 0,
    reactions_given INTEGER DEFAULT 0,
    voice_minutes INTEGER DEFAULT 0,
    threads_created INTEGER DEFAULT 0,
    last_active TIMESTAMP,
    PRIMARY KEY (user_id, server_id, date),
    FOREIGN KEY (user_id, server_id) REFERENCES members(user_id, server_id)
);
```

## Implementation Sketch

### Analytics Collection Service
```go
type AnalyticsService struct {
    db     *database.DB
    redis  *redis.Client
    config *AnalyticsConfig
}

type ServerMetrics struct {
    ServerID    string                 `json:"server_id"`
    Period      string                 `json:"period"` // daily, weekly, monthly
    Metrics     map[string]interface{} `json:"metrics"`
    GeneratedAt time.Time             `json:"generated_at"`
}

func (s *AnalyticsService) CollectMessageMetrics(message *Message) {
    // Real-time metric updates
    s.redis.Incr(fmt.Sprintf("metrics:server:%s:messages:today", message.ServerID))
    s.redis.Incr(fmt.Sprintf("metrics:channel:%s:messages:today", message.ChannelID))
    s.redis.HSet(
        fmt.Sprintf("metrics:member:%s:%s:today", message.AuthorID, message.ServerID),
        "messages", 1,
    )

    // Batch write to database hourly
    s.queueMetricUpdate("server_messages", message.ServerID, 1)
}

func (s *AnalyticsService) GenerateServerInsights(serverID string, period string) (*ServerInsights, error) {
    insights := &ServerInsights{
        ServerID: serverID,
        Period:   period,
        Overview: s.getOverviewMetrics(serverID, period),
        Members:  s.getMemberMetrics(serverID, period),
        Channels: s.getChannelMetrics(serverID, period),
        Activity: s.getActivityMetrics(serverID, period),
        Growth:   s.getGrowthMetrics(serverID, period),
    }

    return insights, nil
}
```

### Dashboard Frontend Component
```typescript
interface ServerInsights {
    overview: OverviewMetrics;
    members: MemberMetrics;
    channels: ChannelMetrics[];
    activity: ActivityMetrics;
    growth: GrowthMetrics;
}

interface OverviewMetrics {
    totalMembers: number;
    activeMembers: number; // Last 7 days
    messagesThisPeriod: number;
    voiceMinutesThisPeriod: number;
    averageOnlineMembers: number;
    retentionRate: number; // % of new members still active after 30 days
}

interface ChannelMetrics {
    channelId: string;
    channelName: string;
    messageCount: number;
    uniqueAuthors: number;
    reactionCount: number;
    voiceMinutes?: number; // Voice channels only
    threadCount?: number;  // Forum/text channels
    engagementScore: number; // Calculated metric
}

class ServerInsightsDashboard {
    async loadInsights(serverId: string, period: string) {
        const insights = await api.get(`/servers/${serverId}/insights`, {
            params: { period }
        });

        this.renderOverview(insights.overview);
        this.renderMemberActivity(insights.members);
        this.renderChannelPerformance(insights.channels);
        this.renderGrowthTrends(insights.growth);
    }

    renderChannelPerformance(channels: ChannelMetrics[]) {
        // Sort by engagement score
        const sortedChannels = channels.sort((a, b) =>
            b.engagementScore - a.engagementScore
        );

        // Render top performing channels chart
        this.chartLibrary.renderBarChart({
            data: sortedChannels.slice(0, 10),
            xField: 'channelName',
            yField: 'engagementScore',
            title: 'Top Performing Channels'
        });
    }
}
```

### Key Insights & Metrics
```go
type InsightsCalculator struct{}

func (ic *InsightsCalculator) CalculateEngagementScore(channel *ChannelMetrics) float64 {
    // Weighted engagement score
    messageScore := float64(channel.MessageCount) * 1.0
    authorScore := float64(channel.UniqueAuthors) * 2.0    // Diversity bonus
    reactionScore := float64(channel.ReactionCount) * 0.5
    voiceScore := float64(channel.VoiceMinutes) * 0.1

    totalScore := messageScore + authorScore + reactionScore + voiceScore

    // Normalize by channel age and member count
    return totalScore / float64(channel.AgeInDays * channel.ServerMemberCount / 100)
}

func (ic *InsightsCalculator) CalculateRetentionRate(serverID string, days int) float64 {
    newMembers := ic.getNewMembersCount(serverID, days)
    stillActiveMembers := ic.getStillActiveMembersCount(serverID, days)

    if newMembers == 0 {
        return 0
    }

    return float64(stillActiveMembers) / float64(newMembers) * 100
}
```

## Key Dashboard Sections

### 1. Overview Dashboard
- **Member Statistics**: Total, active (7d), growth rate, retention
- **Activity Metrics**: Messages/day, voice time, reactions, threads
- **Engagement Score**: Overall server health metric
- **Peak Activity Hours**: When community is most active

### 2. Member Analytics
- **Active vs Inactive Members**: Breakdown by activity level
- **New Member Trends**: Join rate, retention curves
- **Top Contributors**: Most active members by various metrics
- **Member Journey**: Onboarding and engagement funnels

### 3. Channel Performance
- **Channel Rankings**: By activity, engagement, growth
- **Channel Usage Patterns**: Peak times, message distribution
- **Content Analysis**: Most reacted messages, popular topics
- **Voice Channel Utilization**: Usage patterns and peak times

### 4. Growth & Health
- **Growth Trends**: Member acquisition over time
- **Churn Analysis**: Member departure patterns
- **Engagement Trends**: Message frequency, voice usage over time
- **Community Health Score**: Composite metric of server vitality

## Dependencies
1. **Analytics Infrastructure**: Time-series data collection ✅ (Prometheus mentioned in codebase)
2. **Permission System**: Admin/owner access controls ✅
3. **Database Aggregation**: PostgreSQL with analytics queries ✅
4. **Chart Library**: Frontend data visualization (Chart.js, D3, etc.)

## Success Metrics
- Server owner engagement with insights dashboard >70%
- Data-driven moderation decisions increase >40%
- Server growth optimization through insights >25%
- Premium subscription conversion from insights usage >15%

## Implementation Priority
**P1** - Important for server management and community building. Helps server owners make informed decisions about community growth and engagement. Differentiates Hearth for serious community builders.

## Feature Breakdown
### Phase 1: Basic Analytics
- Member count and growth tracking
- Message and voice activity metrics
- Channel performance comparison
- Basic engagement scoring

### Phase 2: Advanced Insights
- Member journey and retention analysis
- Predictive analytics (growth forecasts)
- Custom metric tracking
- Automated reports and alerts

### Phase 3: Community Tools
- Engagement recommendations
- A/B testing for server changes
- Community health diagnostics
- Integration with moderation tools

## Privacy Considerations
- **Data Anonymization**: Aggregate metrics without exposing individual users
- **Retention Policy**: Analytics data retention limits (e.g., 2 years)
- **User Consent**: Clear privacy policy for analytics collection
- **Opt-out Options**: Allow users to exclude their data from analytics
- **GDPR Compliance**: Right to data portability and deletion

## Technical Considerations
- **Performance**: Pre-aggregate metrics to avoid real-time computation
- **Scalability**: Efficient time-series data storage and querying
- **Real-time Updates**: Balance between accuracy and performance
- **Export Features**: CSV/JSON export for external analysis
- **API Access**: Allow third-party integrations for advanced users