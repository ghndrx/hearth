# Database Sharding Strategies for Chat Messages

> Research compiled: 2026-02-13
> Focus: Database partitioning and sharding for message storage at scale
> Sources: Discord, Slack/Vitess, Matrix Synapse, PostgreSQL/Citus engineering blogs

## Executive Summary

Chat applications face unique database challenges: append-heavy write patterns, time-ordered reads, hot partitions from active channels, and exponential growth. This document explores how major platforms solve these problems and provides implementation guidance for Hearth across both self-hosted and large-scale deployments.

---

## 1. Industry Approaches

### 1.1 Discord: Cassandra → ScyllaDB

Discord stores **trillions of messages** and evolved through three database generations:

| Era | Database | Scale | Issues |
|-----|----------|-------|--------|
| 2015 | MongoDB | 100M messages | Data/index exceeded RAM, unpredictable latency |
| 2017 | Cassandra | Billions (12 nodes) | Hot partitions, GC pauses, compaction issues |
| 2022 | ScyllaDB | Trillions (72 nodes) | Current solution |

#### Key Architecture Decisions

**Partition Key Strategy:**
```sql
-- Discord's minimal message schema (CQL)
CREATE TABLE messages (
    channel_id bigint,
    bucket int,          -- Static time window
    message_id bigint,   -- Snowflake ID (chronologically sortable)
    author_id bigint,
    content text,
    PRIMARY KEY ((channel_id, bucket), message_id)
) WITH CLUSTERING ORDER BY (message_id DESC);
```

**Why this works:**
- **channel_id + bucket** = partition key (all messages for a channel/time-window colocated)
- **message_id** = clustering key (sorted within partition)
- Bucket prevents unbounded partition growth
- Newer messages naturally accessed most → hot data fits in cache

**Hot Partition Problem:**
- Large servers (100K+ users) generate orders of magnitude more traffic than small friend groups
- Solution: **Request coalescing** in Rust data services layer
  - If multiple users request the same row simultaneously, query database once
  - Consistent hash-based routing ensures same channel → same service instance
  - Dramatically reduces database hotspotting

**Performance After Migration:**
| Metric | Cassandra | ScyllaDB |
|--------|-----------|----------|
| Read p99 latency | 40-125ms | 15ms |
| Write p99 latency | 5-70ms | 5ms |
| Node count | 177 | 72 |
| Storage per node | 4TB avg | 9TB |

---

### 1.2 Slack: MySQL + Vitess

Slack maintains MySQL but uses **Vitess** for horizontal scaling and shard management.

#### Original Architecture Problems

```
workspace_sharded model:
├── All data for workspace X on shard N
├── Shards contain thousands of workspaces
└── Problems:
    ├── Large enterprise customers exhaust single shard
    ├── Hot spots from active workspaces
    └── New features (Slack Connect, Enterprise Grid) break workspace paradigm
```

#### Vitess Solution

Vitess provides:
- **Flexible sharding keys** - shard by channel_id instead of workspace_id
- **Transparent routing** - application doesn't manage shard topology
- **Operational tooling** - automatic failover, backups, resharding
- **MySQL compatibility** - existing queries work with minimal changes

**Current Scale:**
- 2.3 million QPS at peak (2M reads, 300K writes)
- Median query latency: 2ms
- p99 latency: 11ms
- 99% of traffic migrated from legacy to Vitess

**Resharding for COVID Surge:**
During March 2020, Slack saw 50% query rate increase in one week. They horizontally resharded their busiest keyspace using Vitess's splitting workflows—impossible with their original architecture.

---

### 1.3 Matrix Synapse: PostgreSQL + Workers

Matrix's approach is interesting for self-hosters because it shows how to scale a single-database system.

#### Scalability Evolution

**Original bottleneck:** Single Python process with GIL limitations.

**Solution:** Worker architecture with event streaming:
```
                    ┌─────────────┐
                    │  Redis Pub  │
                    │    /Sub     │
                    └──────┬──────┘
                           │
    ┌──────────────────────┼──────────────────────┐
    │                      │                      │
┌───▼───┐           ┌──────▼──────┐        ┌──────▼──────┐
│ Event │           │   Event     │        │   Event     │
│Persist│           │  Persist    │        │  Persist    │
│Worker1│           │  Worker2    │        │  Worker3    │
└───┬───┘           └──────┬──────┘        └──────┬──────┘
    │                      │                      │
    └──────────────────────┼──────────────────────┘
                           │
                    ┌──────▼──────┐
                    │  PostgreSQL │
                    │   (shared)  │
                    └─────────────┘
```

**Sharding Strategy:**
- Events sharded by **room_id** to workers
- Vector clock positions track "persisted up-to" across workers
- Cache invalidation streamed via Redis

**Database Challenges:**
- `state_groups_state` table grows enormous (100GB+ common)
- Room complexity limits help constrain problematic rooms
- Regular maintenance required (state compressor, purge old rooms)

---

### 1.4 PostgreSQL Native: Citus Extension

For self-hosters wanting PostgreSQL scalability without NoSQL complexity:

#### Partitioning Types

| Type | Use Case | Example |
|------|----------|---------|
| **RANGE** | Time-series data | Partition by month/week |
| **LIST** | Category-based | Partition by region |
| **HASH** | Even distribution | Partition by channel_id |

#### Combined Strategy for Chat

```sql
-- Create partitioned messages table
CREATE TABLE messages (
    id BIGINT GENERATED ALWAYS AS IDENTITY,
    channel_id BIGINT NOT NULL,
    author_id BIGINT NOT NULL,
    content TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
) PARTITION BY RANGE (created_at);

-- Create monthly partitions
CREATE TABLE messages_2026_01 PARTITION OF messages
    FOR VALUES FROM ('2026-01-01') TO ('2026-02-01');
CREATE TABLE messages_2026_02 PARTITION OF messages
    FOR VALUES FROM ('2026-02-01') TO ('2026-03-01');

-- Add index for channel queries
CREATE INDEX ON messages (channel_id, created_at DESC);
```

#### Citus for Horizontal Scaling

```sql
-- Distribute messages across cluster by channel_id
SELECT create_distributed_table('messages', 'channel_id');

-- Co-locate related tables
SELECT create_distributed_table('reactions', 'channel_id',
    colocate_with => 'messages');
```

**Benefits:**
- Range partitions by time → efficient drops of old data
- Hash distribution by channel → parallel query execution
- Colocated tables → efficient joins within same channel

---

## 2. Hearth Implementation Strategy

### 2.1 Tiered Approach Based on Scale

#### Tier 1: Self-Hosted (< 10K users)

**Database:** Single PostgreSQL instance with native partitioning

```sql
-- Monthly range partitioning only
CREATE TABLE messages (
    id BIGSERIAL PRIMARY KEY,
    channel_id BIGINT NOT NULL REFERENCES channels(id),
    author_id BIGINT NOT NULL REFERENCES users(id),
    content TEXT,
    attachments JSONB DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ
) PARTITION BY RANGE (created_at);

-- Automate partition creation
CREATE OR REPLACE FUNCTION create_monthly_partition()
RETURNS void AS $$
DECLARE
    partition_date DATE := DATE_TRUNC('month', NOW() + INTERVAL '1 month');
    partition_name TEXT := 'messages_' || TO_CHAR(partition_date, 'YYYY_MM');
    start_date DATE := partition_date;
    end_date DATE := partition_date + INTERVAL '1 month';
BEGIN
    EXECUTE format(
        'CREATE TABLE IF NOT EXISTS %I PARTITION OF messages
         FOR VALUES FROM (%L) TO (%L)',
        partition_name, start_date, end_date
    );
END;
$$ LANGUAGE plpgsql;

-- Schedule via pg_cron
SELECT cron.schedule('create_partitions', '0 0 25 * *', 
    'SELECT create_monthly_partition()');
```

**Benefits:**
- Simple operations
- Easy backups (full database)
- No distributed system complexity
- Partition pruning speeds queries

**Scaling limits:** ~100K concurrent connections, ~50K messages/second

#### Tier 2: Medium Scale (10K - 100K users)

**Database:** PostgreSQL + Citus OR read replicas + connection pooling

**Option A: Read Replicas**
```yaml
# PgBouncer configuration for read/write splitting
databases:
  hearth_write = host=primary.db port=5432 dbname=hearth
  hearth_read = host=replica1.db,replica2.db port=5432 dbname=hearth
```

**Option B: Citus Sharding**
```sql
-- Distribute tables by server_id for multi-tenant pattern
SELECT create_distributed_table('messages', 'server_id');
SELECT create_distributed_table('channels', 'server_id');
SELECT create_reference_table('users');  -- Small, frequently joined

-- Messages sharded by server, partitioned by time
-- Each server's data stays together for efficient queries
```

#### Tier 3: Large Scale (100K+ users)

**Database:** ScyllaDB or PostgreSQL + Citus cluster

**Message Schema (ScyllaDB):**
```sql
CREATE TABLE messages (
    server_id bigint,
    channel_id bigint,
    bucket int,              -- Weekly or daily bucket
    message_id bigint,       -- Snowflake timestamp-based ID
    author_id bigint,
    content text,
    attachments frozen<list<frozen<attachment>>>,
    embeds frozen<list<frozen<embed>>>,
    reactions frozen<map<text, frozen<set<bigint>>>>,
    PRIMARY KEY ((channel_id, bucket), message_id)
) WITH CLUSTERING ORDER BY (message_id DESC)
  AND compaction = {'class': 'TimeWindowCompactionStrategy',
                    'compaction_window_size': 7,
                    'compaction_window_unit': 'DAYS'};
```

### 2.2 Hot Partition Mitigation

Implement request coalescing service (inspired by Discord):

```typescript
// Data service layer with request coalescing
class MessageDataService {
  private pendingRequests = new Map<string, Promise<Message[]>>();

  async getMessages(channelId: string, limit: number): Promise<Message[]> {
    const cacheKey = `${channelId}:${limit}`;
    
    // Check if request already in flight
    const pending = this.pendingRequests.get(cacheKey);
    if (pending) {
      return pending; // Subscribe to existing request
    }

    // Start new request
    const request = this.fetchFromDatabase(channelId, limit)
      .finally(() => {
        this.pendingRequests.delete(cacheKey);
      });

    this.pendingRequests.set(cacheKey, request);
    return request;
  }
}
```

**Consistent hash routing:**
```typescript
// Route requests for same channel to same service instance
function routeRequest(channelId: string): ServiceInstance {
  const hash = murmurhash3(channelId);
  const instanceIndex = hash % serviceInstances.length;
  return serviceInstances[instanceIndex];
}
```

### 2.3 Message ID Strategy

Use Snowflake-style IDs for chronological sorting:

```typescript
// Snowflake ID: 64-bit = timestamp(42) + node_id(10) + sequence(12)
class SnowflakeGenerator {
  private static EPOCH = 1704067200000n; // 2024-01-01
  private static NODE_BITS = 10n;
  private static SEQUENCE_BITS = 12n;
  
  private nodeId: bigint;
  private sequence = 0n;
  private lastTimestamp = 0n;

  constructor(nodeId: number) {
    this.nodeId = BigInt(nodeId);
  }

  generate(): bigint {
    let timestamp = BigInt(Date.now()) - SnowflakeGenerator.EPOCH;
    
    if (timestamp === this.lastTimestamp) {
      this.sequence = (this.sequence + 1n) & 4095n; // 12-bit mask
      if (this.sequence === 0n) {
        // Wait for next millisecond
        while (timestamp <= this.lastTimestamp) {
          timestamp = BigInt(Date.now()) - SnowflakeGenerator.EPOCH;
        }
      }
    } else {
      this.sequence = 0n;
    }
    
    this.lastTimestamp = timestamp;
    
    return (timestamp << 22n) | (this.nodeId << 12n) | this.sequence;
  }
}
```

**Benefits:**
- Sortable without secondary index
- Encodes creation time (useful for pagination)
- Distributed generation without coordination
- Extracts timestamp: `new Date(Number((id >> 22n) + EPOCH))`

---

## 3. Data Lifecycle Management

### 3.1 Time-Based Archival

```sql
-- Archive messages older than retention period
CREATE OR REPLACE FUNCTION archive_old_messages(
    retention_days INT DEFAULT 365
) RETURNS void AS $$
BEGIN
    -- Move to archive table
    INSERT INTO messages_archive 
    SELECT * FROM messages 
    WHERE created_at < NOW() - (retention_days || ' days')::INTERVAL;
    
    -- Drop old partitions
    FOR partition_name IN 
        SELECT tablename FROM pg_tables 
        WHERE tablename LIKE 'messages_20%'
        AND tablename < 'messages_' || TO_CHAR(
            NOW() - (retention_days || ' days')::INTERVAL, 'YYYY_MM'
        )
    LOOP
        EXECUTE 'DROP TABLE ' || partition_name;
    END LOOP;
END;
$$ LANGUAGE plpgsql;
```

### 3.2 Cold Storage Strategy

```yaml
# Tiered storage configuration
storage:
  hot:
    database: postgresql
    retention: 90_days
    indexes: full
  warm:
    database: postgresql_archive
    retention: 365_days
    indexes: minimal  # channel_id, created_at only
  cold:
    storage: s3_parquet
    retention: forever
    compression: zstd
    access: batch_only
```

---

## 4. Query Optimization Patterns

### 4.1 Channel Message Fetch

Most common query - optimize heavily:

```sql
-- Use covering index
CREATE INDEX idx_messages_channel_fetch ON messages 
    (channel_id, created_at DESC) 
    INCLUDE (id, author_id, content, attachments);

-- Query uses index-only scan when possible
SELECT id, author_id, content, attachments, created_at
FROM messages
WHERE channel_id = $1
  AND created_at > $2  -- Partition pruning
ORDER BY created_at DESC
LIMIT 50;
```

### 4.2 Search Across Channels

```sql
-- For full-text search, consider separate search index
-- PostgreSQL option:
CREATE INDEX idx_messages_search ON messages 
    USING GIN (to_tsvector('english', content));

-- Better: External search engine (Meilisearch, Typesense)
-- Async index updates via CDC or event stream
```

---

## 5. Recommendations for Hearth

### Phase 1: Foundation (MVP)
1. **Start with PostgreSQL** + native range partitioning by month
2. **Snowflake IDs** for messages from day one (prevents migration pain)
3. **Monthly partition automation** via pg_cron
4. Design for eventual sharding (server_id in all queries)

### Phase 2: Growth (10K+ users)
1. Add **read replicas** with PgBouncer routing
2. Implement **request coalescing** layer
3. Add **message caching** (Redis/DragonflyDB) for recent messages

### Phase 3: Scale (100K+ users)
1. Deploy **Citus** for horizontal PostgreSQL sharding
2. OR migrate to **ScyllaDB** for Cassandra-compatible performance
3. Implement **data service layer** with consistent hashing

### Key Principles
- **Shard by channel/server** - keeps related data together
- **Partition by time** - efficient archival and queries
- **Coalesce requests** - protect database from thundering herds
- **Snowflake IDs** - distributed generation, natural ordering

---

## References

1. [Discord: How Discord Stores Trillions of Messages](https://discord.com/blog/how-discord-stores-trillions-of-messages) (2023)
2. [Slack: Scaling Datastores with Vitess](https://slack.engineering/scaling-datastores-at-slack-with-vitess/)
3. [Matrix: How We Fixed Synapse's Scalability](https://matrix.org/blog/2020/11/03/how-we-fixed-synapse-s-scalability/)
4. [Citus: Understanding Partitioning and Sharding](https://www.citusdata.com/blog/2023/08/04/understanding-partitioning-and-sharding-in-postgres-and-citus/)
5. [ScyllaDB: Discord Migration Case Study](https://www.scylladb.com/tech-talk/how-discord-migrated-trillions-of-messages-from-cassandra-to-scylladb/)
