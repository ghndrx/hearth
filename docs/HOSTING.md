# Hearth — Hosting & SaaS Guide

**Version:** 1.0  
**Last Updated:** 2026-02-11

---

## Overview

Hearth can be deployed in three ways:

| Model | Description | Target |
|-------|-------------|--------|
| **Self-Hosted** | User runs on own infrastructure | Power users, privacy-focused |
| **Managed Hosting** | We run it, they use it | Teams, organizations |
| **SaaS Multi-Tenant** | Shared infrastructure | Consumer, small groups |

---

## Self-Hosted (Open Source)

The core Hearth product. Free, open source, runs anywhere.

### One-Line Install
```bash
curl -sSL https://get.hearth.chat | bash
```

### With FusionAuth (Enterprise SSO)
```bash
curl -sSL https://get.hearth.chat | bash -s -- --with-fusionauth
```

### With Custom Domain
```bash
curl -sSL https://get.hearth.chat | bash -s -- --domain chat.example.com
```

### Requirements
- Docker + Docker Compose
- 1 GB RAM minimum
- 10 GB storage

---

## Managed Hosting (Hearth Cloud)

For teams who want Hearth without the ops work.

### Tiers

| Tier | Users | Storage | Voice | Price |
|------|-------|---------|-------|-------|
| **Starter** | 25 | 5 GB | 5 concurrent | $29/mo |
| **Team** | 100 | 25 GB | 25 concurrent | $99/mo |
| **Business** | 500 | 100 GB | 100 concurrent | $299/mo |
| **Enterprise** | Unlimited | Custom | Unlimited | Custom |

### Features by Tier

| Feature | Starter | Team | Business | Enterprise |
|---------|---------|------|----------|------------|
| Custom domain | ❌ | ✅ | ✅ | ✅ |
| SSO (OIDC/SAML) | ❌ | ❌ | ✅ | ✅ |
| Audit log export | ❌ | ✅ | ✅ | ✅ |
| SLA | - | 99.5% | 99.9% | 99.99% |
| Support | Community | Email | Priority | Dedicated |
| Data residency | - | - | US/EU | Custom |
| Backup frequency | Daily | Daily | Hourly | Real-time |

### Architecture (Per-Customer)

Each managed customer gets isolated infrastructure:

```
┌─────────────────────────────────────────────────────────────┐
│                     Hearth Cloud                             │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────┐ │
│  │   Customer A    │  │   Customer B    │  │  Customer C │ │
│  ├─────────────────┤  ├─────────────────┤  ├─────────────┤ │
│  │ Hearth Instance │  │ Hearth Instance │  │ Hearth Inst │ │
│  │ PostgreSQL      │  │ PostgreSQL      │  │ PostgreSQL  │ │
│  │ Redis           │  │ Redis           │  │ Redis       │ │
│  │ S3 Bucket       │  │ S3 Bucket       │  │ S3 Bucket   │ │
│  └─────────────────┘  └─────────────────┘  └─────────────┘ │
│                                                             │
│  ┌───────────────────────────────────────────────────────┐ │
│  │                 Shared Services                        │ │
│  │  Load Balancer │ Monitoring │ Backups │ DNS           │ │
│  └───────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

### Provisioning Flow

```
1. Customer signs up at hearth.chat/signup
2. Select tier and payment
3. Auto-provision infrastructure (Kubernetes)
4. DNS setup (customer.hearth.chat or custom domain)
5. Admin account created, welcome email sent
6. Customer configures server, invites members
```

### Technology Stack

| Component | Technology |
|-----------|------------|
| Orchestration | Kubernetes (EKS/GKE) |
| Database | RDS PostgreSQL per customer |
| Cache | ElastiCache Redis per customer |
| Storage | S3 bucket per customer |
| CDN | CloudFront |
| DNS | Route53 |
| SSL | Let's Encrypt via cert-manager |
| Monitoring | Prometheus + Grafana |
| Logging | Loki |
| Provisioning | Terraform + Helm |

---

## SaaS Multi-Tenant (Consumer)

Shared infrastructure for individual users and small groups. 
Similar to Discord's free tier.

### Model

- Free tier with limits
- Optional premium features
- Shared infrastructure, isolated data

### Tiers

| Feature | Free | Premium ($4.99/mo) |
|---------|------|-------------------|
| Servers | 5 | 100 |
| File uploads | 10 MB | 100 MB |
| Custom emoji | 10 | 100 |
| Voice quality | 64 kbps | 256 kbps |
| Screen share | 720p | 1080p |
| Custom status | ❌ | ✅ |
| Animated avatar | ❌ | ✅ |
| Profile banner | ❌ | ✅ |

### Architecture (Multi-Tenant)

Shared infrastructure with data isolation:

```
┌─────────────────────────────────────────────────────────────┐
│                   Hearth SaaS Platform                       │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │              Application Tier (Stateless)            │   │
│  │  Hearth Pod  │  Hearth Pod  │  Hearth Pod  │  ...   │   │
│  └─────────────────────────────────────────────────────┘   │
│                           │                                 │
│  ┌─────────────────────────────────────────────────────┐   │
│  │              Data Tier (Shared, Isolated)            │   │
│  │                                                       │   │
│  │  PostgreSQL (row-level security by user/server)      │   │
│  │  Redis (namespace by user)                            │   │
│  │  S3 (prefix by user: s3://bucket/user_123/...)       │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### Data Isolation

Row-level security in PostgreSQL:

```sql
-- Enable RLS
ALTER TABLE servers ENABLE ROW LEVEL SECURITY;

-- Users can only see servers they're members of
CREATE POLICY server_access ON servers
    USING (
        id IN (
            SELECT server_id FROM members 
            WHERE user_id = current_setting('app.user_id')::uuid
        )
        OR owner_id = current_setting('app.user_id')::uuid
    );
```

---

## Infrastructure Costs (Estimates)

### Self-Hosted
- **Your infrastructure costs only**
- Raspberry Pi to cloud, you choose

### Managed Hosting (Per Customer)

| Component | Monthly Cost |
|-----------|--------------|
| EKS node share | $20 |
| RDS (db.t4g.small) | $25 |
| ElastiCache (cache.t4g.micro) | $12 |
| S3 (10 GB) | $0.25 |
| Data transfer | $5 |
| **Total** | **~$62** |

Margin at $99/mo tier: ~37%

### SaaS Multi-Tenant (Per 1000 Users)

| Component | Monthly Cost |
|-----------|--------------|
| EKS (2 nodes) | $150 |
| RDS (db.r6g.large) | $200 |
| ElastiCache (cache.r6g.large) | $150 |
| S3 (100 GB) | $2.50 |
| Data transfer | $50 |
| **Total** | **~$550** |

At 5% premium conversion ($4.99): ~$250/mo revenue per 1000 users
Break-even: ~2200 users with 5% conversion

---

## Deployment Automation

### Managed Hosting Provisioning

```yaml
# customer-values.yaml (Helm)
customer:
  id: cust_abc123
  name: "Acme Corp"
  domain: chat.acme.com
  tier: business

resources:
  requests:
    cpu: "500m"
    memory: "1Gi"
  limits:
    cpu: "2"
    memory: "4Gi"

database:
  size: db.t4g.medium
  storage: 50Gi

storage:
  bucket: hearth-acme-prod
  quota: 100Gi
```

### Terraform Module

```hcl
module "hearth_customer" {
  source = "./modules/hearth-customer"

  customer_id   = "cust_abc123"
  customer_name = "Acme Corp"
  domain        = "chat.acme.com"
  tier          = "business"

  # Database
  db_instance_class = "db.t4g.medium"
  db_storage_gb     = 50

  # Redis
  redis_node_type = "cache.t4g.small"

  # Storage
  s3_quota_gb = 100

  # Kubernetes
  namespace = "hearth-cust-abc123"
}
```

---

## Billing Integration

### Stripe Integration

```go
type Subscription struct {
    CustomerID   string
    Tier         Tier
    Status       SubscriptionStatus
    CurrentPeriodStart time.Time
    CurrentPeriodEnd   time.Time
    StripeSubscriptionID string
}

func (s *BillingService) HandleWebhook(event stripe.Event) error {
    switch event.Type {
    case "customer.subscription.created":
        return s.provisionCustomer(event)
    case "customer.subscription.updated":
        return s.updateTier(event)
    case "customer.subscription.deleted":
        return s.deprovisionCustomer(event)
    case "invoice.payment_failed":
        return s.handlePaymentFailure(event)
    }
    return nil
}
```

### Usage Metering

Track for overage billing or limits:
- Storage used (GB)
- Voice minutes
- Active users
- API calls

---

## Migration Tools

### Self-Hosted to Managed

```bash
# Export from self-hosted
hearth export --all --output backup.tar.gz

# Import to managed (via support)
# We handle the migration
```

### Managed to Self-Hosted

```bash
# Request export from dashboard
# Download backup.tar.gz

# Import to self-hosted
hearth import --input backup.tar.gz
```

---

## Support Tiers

| Level | Response Time | Channels |
|-------|---------------|----------|
| Community | Best effort | GitHub, Discord |
| Email | 48 hours | Email |
| Priority | 4 hours | Email, chat |
| Dedicated | 1 hour | Slack, phone, dedicated CSM |

---

## Legal Considerations

### Terms of Service
- Acceptable use policy
- Data retention policies
- Account termination

### Privacy Policy
- Data collection (minimal)
- Data sharing (none)
- GDPR compliance
- Data deletion requests

### Data Processing Agreement
- For enterprise customers
- HIPAA BAA available
- SOC 2 compliance (roadmap)

---

*End of Hosting Guide*
