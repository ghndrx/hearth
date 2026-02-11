# Hearth — Security Model

**Version:** 1.0  
**Last Updated:** 2026-02-11

---

## Security Principles

1. **Defense in Depth** — Multiple layers of security controls
2. **Least Privilege** — Minimal permissions by default
3. **Zero Trust** — Verify every request, trust nothing implicitly
4. **Transparency** — Open source, auditable code
5. **Privacy First** — Collect minimal data, protect what we store

---

## Authentication

### Password Storage
- **Algorithm:** Argon2id (winner of Password Hashing Competition)
- **Parameters:**
  - Memory: 64 MB
  - Iterations: 3
  - Parallelism: 4
  - Salt: 16 bytes (crypto/rand)
  - Hash: 32 bytes

```go
// Example configuration
argon2.IDKey(password, salt, 3, 64*1024, 4, 32)
```

### Session Management
- **Access Token:** JWT (RS256), 15-minute expiry
- **Refresh Token:** Opaque, 7-day expiry, single-use
- **Token Rotation:** Refresh issues new access + refresh pair
- **Revocation:** Refresh tokens stored in DB, deletable

### Multi-Factor Authentication
- **Method:** TOTP (RFC 6238)
- **Compatible:** Google Authenticator, Authy, 1Password
- **Backup Codes:** 10 single-use codes on setup
- **Recovery:** Email-based with additional verification

### OAuth2 / SSO
- **Providers:** Google, GitHub, Discord (optional)
- **OIDC:** OpenID Connect support for enterprise SSO
- **Account Linking:** OAuth can link to existing accounts

---

## Authorization

### Permission Model
See [ARCHITECTURE.md](ARCHITECTURE.md#4-permission-system) for full permission details.

Key principles:
- Permissions are additive (roles stack)
- Channel overrides can allow or deny
- Administrator bypasses all checks
- Server owner has implicit Administrator

### Permission Calculation Order
1. Check if user is server owner → all permissions
2. Calculate base from @everyone role
3. Add permissions from all user roles
4. If Administrator bit set → all permissions
5. Apply channel role overrides (deny then allow)
6. Apply channel user overrides (deny then allow)

### API Authorization
Every API endpoint:
1. Validates JWT signature and expiry
2. Extracts user ID from token
3. Checks user exists and is not banned
4. Calculates permissions for target resource
5. Returns 403 if insufficient permissions

---

## Data Protection

### Encryption in Transit
- **Protocol:** TLS 1.3 only (TLS 1.2 for legacy if needed)
- **Cipher Suites:** AEAD only (AES-GCM, ChaCha20-Poly1305)
- **Certificates:** Let's Encrypt or user-provided
- **HSTS:** Enabled with 1-year max-age

### Encryption at Rest
- **Database:** Optional at-rest encryption (Postgres TDE or disk)
- **Media Files:** Stored with random filenames, no directory listing
- **Secrets:** Environment variables or encrypted config file
- **Backups:** Should be encrypted by operator

### Sensitive Data Handling
| Data | Storage | Notes |
|------|---------|-------|
| Passwords | Argon2id hash | Never logged or transmitted |
| Email | Encrypted column | Decrypted only for delivery |
| IP Addresses | Hashed or encrypted | Rotated after 30 days |
| Tokens | Hashed | Only hash stored in DB |
| MFA Secrets | Encrypted | Decrypted only for verification |

---

## Input Validation

### Message Content
- Maximum 2000 characters
- Markdown sanitized (no script injection)
- HTML stripped (except allowed Markdown output)
- Rate limited: 5 messages/5 seconds per channel

### File Uploads
- Maximum 8 MB (configurable)
- File type validation by magic bytes, not extension
- Allowed types: Images, videos, audio, documents
- Executable files blocked (.exe, .bat, .sh, etc.)
- Optional ClamAV scanning

### API Inputs
- JSON schema validation on all endpoints
- String lengths enforced
- Enum values validated
- IDs validated as proper UUIDs

---

## Rate Limiting

### Global Limits
| Endpoint Type | Limit | Window |
|--------------|-------|--------|
| Authentication | 5 requests | 1 minute |
| API (authenticated) | 50 requests | 1 second |
| API (unauthenticated) | 10 requests | 1 second |
| File Upload | 10 uploads | 1 minute |
| WebSocket Messages | 120 messages | 1 minute |

### Per-Resource Limits
- Message sends: 5 per 5 seconds per channel
- Reactions: 10 per second
- Server creation: 10 per day
- DM creation: 10 per day

### Rate Limit Headers
```
X-RateLimit-Limit: 50
X-RateLimit-Remaining: 45
X-RateLimit-Reset: 1707616800
Retry-After: 5
```

---

## Attack Mitigation

### XSS (Cross-Site Scripting)
- Content Security Policy (CSP) header
- All user content sanitized on display
- React/Svelte auto-escapes by default
- No `dangerouslySetInnerHTML` without sanitization

### CSRF (Cross-Site Request Forgery)
- SameSite=Strict cookies
- CSRF tokens for state-changing actions
- Origin header validation

### SQL Injection
- All queries use parameterized statements
- No string concatenation for queries
- ORM with query builder

### SSRF (Server-Side Request Forgery)
- Link preview fetches use allowlist
- Private IP ranges blocked
- Timeout on external requests
- User-agent identifies as Hearth

### DoS / DDoS
- Rate limiting at all layers
- Connection limits per IP
- Request body size limits
- Query complexity limits (GraphQL if used)

---

## Audit Logging

### Logged Events
All moderation and configuration changes:
- Role changes
- Channel changes
- Member kicks/bans
- Permission changes
- Server settings changes
- Webhook operations

### Log Format
```json
{
  "id": "uuid",
  "server_id": "uuid",
  "user_id": "uuid",
  "action": "MEMBER_BAN_ADD",
  "target_id": "uuid",
  "target_type": "user",
  "changes": [
    {"key": "reason", "new": "Spam"}
  ],
  "reason": "Repeated spam warnings",
  "ip_hash": "sha256...",
  "timestamp": "2026-02-11T03:55:00Z"
}
```

### Retention
- Default: 45 days
- Configurable per instance
- Exportable for compliance

---

## Incident Response

### Vulnerability Reporting
- Email: security@hearth.example.com
- Response time: 48 hours
- Disclosure: Coordinated, 90 days

### Severity Levels
| Level | Description | Response |
|-------|-------------|----------|
| Critical | RCE, auth bypass | Immediate patch |
| High | Data exposure | Patch within 7 days |
| Medium | Limited impact | Patch within 30 days |
| Low | Minimal impact | Next release |

### Security Updates
- Announced via GitHub Security Advisories
- Patch releases for critical/high
- Upgrade guide provided

---

## Operator Security Checklist

### Before Deployment
- [ ] Generate strong `SECRET_KEY` (32+ random bytes)
- [ ] Configure TLS with valid certificate
- [ ] Set up firewall rules (only expose 443, 8443)
- [ ] Enable rate limiting
- [ ] Configure backup encryption

### Ongoing
- [ ] Monitor security advisories
- [ ] Apply updates promptly
- [ ] Review audit logs regularly
- [ ] Rotate secrets annually
- [ ] Test backups monthly

### Optional Hardening
- [ ] Enable MFA requirement for admins
- [ ] Configure SSO/OIDC
- [ ] Set up intrusion detection
- [ ] Use read-only root filesystem
- [ ] Run as non-root user
- [ ] Enable database encryption

---

*End of Security Model*
