# JWT & Authentication Security Best Practices

**Research Date:** 2026-02-13  
**Focus:** Token security, refresh token rotation, session management  
**Sources:** Curity, Descope, Slack Security Docs, Auth0, IETF RFCs

---

## Executive Summary

JWT-based auth is foundational but often misimplemented. Key takeaways for Hearth:

1. **Short-lived access tokens** (5-15 min) + **refresh token rotation with reuse detection**
2. **HttpOnly, Secure, SameSite cookies** — never localStorage
3. **Validate issuer, audience, algorithm** on every request
4. **Consider Proof of Possession (DPoP)** for high-security scenarios
5. **Server-side session store** to enable instant revocation

---

## 1. JWT Token Architecture

### Access Token Design
| Property | Recommendation | Rationale |
|----------|---------------|-----------|
| Lifetime | 5-15 minutes | Limits exposure window if stolen |
| Algorithm | ES256 or EdDSA | Faster, more secure than RS256 |
| Storage | HttpOnly cookie | Immune to XSS token theft |
| Claims | Minimal | Don't leak PII/internal details |

### Refresh Token Design
| Property | Recommendation | Rationale |
|----------|---------------|-----------|
| Lifetime | 7-30 days | Balance UX vs security |
| Format | Opaque (not JWT) | Server-side validation, revocable |
| Storage | Secure, HttpOnly cookie | Same as access token |
| Rotation | **Every use** | Limits stolen token utility |

### Claims to Validate (Every Request)
```
iss  - Issuer must match exactly (https://auth.hearth.chat)
aud  - Audience must include the API (hearth-api)
exp  - Token not expired
iat  - Issued-at reasonable (not future)
alg  - Algorithm in allowlist (ES256, not 'none'!)
```

⚠️ **Critical**: Never accept `alg: none` or `alg: noNe` — use strict allowlist matching.

---

## 2. Refresh Token Rotation + Reuse Detection

### How It Works
```
1. Client sends refresh_token to /auth/refresh
2. Server validates token, marks it as USED
3. Server issues NEW access_token + NEW refresh_token
4. Old refresh_token is now invalid
```

### Reuse Detection (Critical)
If a refresh token is used **twice**, it indicates theft:
- Attacker stole token before rotation
- Both attacker and legitimate user now have a token
- One will inevitably try to use an already-used token

**Response to Reuse Detection:**
```python
if stored_token.used:
    # BREACH DETECTED - revoke ALL tokens for this user
    db.query(RefreshToken).filter_by(user_id=user_id).delete()
    db.query(AccessToken).filter_by(user_id=user_id).delete()
    
    # Force full re-authentication
    return 401, "Session compromised - please log in again"
    
    # Optional: Alert user via email/notification
    notify_user_security_event(user_id, "session_revoked")
```

### Database Schema for Tokens
```sql
CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    token_hash VARCHAR(64) NOT NULL,  -- SHA-256 of actual token
    device_fingerprint VARCHAR(256),   -- Optional: tie to device
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,               -- NULL until used
    revoked_at TIMESTAMPTZ,            -- Manual revocation
    replaced_by UUID REFERENCES refresh_tokens(id),
    
    INDEX idx_token_hash (token_hash),
    INDEX idx_user_active (user_id, used_at, revoked_at)
);
```

---

## 3. Token Storage: Cookies vs localStorage

### Why HttpOnly Cookies Win

| Threat | localStorage | HttpOnly Cookie |
|--------|-------------|-----------------|
| XSS token theft | ❌ Vulnerable | ✅ Protected |
| CSRF attacks | ✅ Immune | ⚠️ Needs SameSite/CSRF token |
| Browser extensions | ❌ Accessible | ✅ Protected |

### Cookie Configuration
```typescript
// Access token cookie
res.cookie('access_token', token, {
  httpOnly: true,
  secure: true,           // HTTPS only
  sameSite: 'strict',     // Or 'lax' for OAuth redirects
  maxAge: 15 * 60 * 1000, // 15 minutes
  path: '/api',           // Limit scope
  domain: '.hearth.chat'
});

// Refresh token cookie
res.cookie('refresh_token', token, {
  httpOnly: true,
  secure: true,
  sameSite: 'strict',
  maxAge: 7 * 24 * 60 * 60 * 1000, // 7 days
  path: '/auth/refresh',   // Only sent to refresh endpoint
  domain: '.hearth.chat'
});
```

### CSRF Protection for Cookie-Based Auth
Since cookies are sent automatically, implement CSRF protection:
```typescript
// Option 1: Double-submit cookie
const csrfToken = crypto.randomUUID();
res.cookie('csrf', csrfToken, { sameSite: 'strict' });
// Client must send X-CSRF-Token header matching cookie

// Option 2: SameSite=Strict + custom header requirement
// Require X-Requested-With: XMLHttpRequest on all mutations
```

---

## 4. Session Management (Slack Pattern)

Slack and Discord use **server-side session stores** alongside JWTs for:
- Instant session revocation (logout all devices)
- Session enumeration (show user their active sessions)
- Device/IP tracking for security alerts

### Hybrid Architecture for Hearth
```
┌──────────────────────────────────────────────────────┐
│                   Client Request                      │
├──────────────────────────────────────────────────────┤
│  1. Extract JWT from HttpOnly cookie                 │
│  2. Validate JWT signature + claims (fast, stateless)│
│  3. Check session_id claim against Redis/DB          │
│     - Is session revoked?                            │
│     - Does device fingerprint match?                 │
│  4. If valid, proceed with request                   │
└──────────────────────────────────────────────────────┘
```

### Session Store (Redis)
```typescript
interface Session {
  userId: string;
  deviceId: string;
  ip: string;
  userAgent: string;
  createdAt: Date;
  lastActiveAt: Date;
  revokedAt?: Date;
}

// Key: session:{session_id}
// TTL: Match refresh token lifetime
await redis.setex(
  `session:${sessionId}`,
  7 * 24 * 60 * 60,  // 7 days
  JSON.stringify(session)
);
```

### User-Facing Session Management
```typescript
// GET /api/sessions - List user's active sessions
// Returns: device, IP, location (GeoIP), last active

// DELETE /api/sessions/:id - Revoke specific session
// DELETE /api/sessions - Revoke all sessions (logout everywhere)
```

---

## 5. Algorithm Security

### Recommended Algorithms
| Use Case | Algorithm | Notes |
|----------|-----------|-------|
| Signing (preferred) | **EdDSA** | Fastest, most secure |
| Signing (compatible) | **ES256** | Wide support |
| Signing (legacy) | RS256 | Slower, but universal |
| ❌ Avoid | HS256 | Shared secret = risky |
| ❌ Never | none | Unsigned tokens |

### Algorithm Validation
```typescript
// ALWAYS use allowlist, not denylist
const ALLOWED_ALGORITHMS = ['ES256', 'EdDSA'];

jwt.verify(token, publicKey, {
  algorithms: ALLOWED_ALGORITHMS,  // Strict allowlist
  issuer: 'https://auth.hearth.chat',
  audience: 'hearth-api'
});
```

---

## 6. Proof of Possession (DPoP) — Future Enhancement

For high-security scenarios (admin actions, financial), consider DPoP:

```
1. Client generates ephemeral key pair
2. Client signs a "proof" JWT with private key
3. Access token contains hash of public key
4. API validates both access token AND proof signature
```

**Benefit**: Stolen tokens are useless without the private key.

**Hearth Consideration**: Implement for admin panel access, not general chat.

---

## 7. Implementation Checklist for Hearth

### Phase 1: Core Auth (MVP)
- [ ] ES256 JWT signing with key rotation capability
- [ ] 15-minute access tokens in HttpOnly cookies
- [ ] 7-day opaque refresh tokens with rotation
- [ ] Refresh token reuse detection → revoke all user sessions
- [ ] Validate iss, aud, alg on every request
- [ ] CSRF protection (SameSite + custom header)

### Phase 2: Session Management
- [ ] Redis session store with device fingerprinting
- [ ] User-visible active sessions list
- [ ] "Logout all devices" functionality
- [ ] Suspicious login alerts (new device/location)

### Phase 3: Enhanced Security
- [ ] IP allowlisting for API tokens (like Slack)
- [ ] DPoP for admin panel access
- [ ] Automatic session termination on password change
- [ ] Gradual trust elevation (re-auth for sensitive actions)

---

## 8. TODO: PRD Gap Analysis

After reviewing this research against current PRD:

1. **ADD**: Specify refresh token rotation as requirement
2. **ADD**: Define session management user stories
   - "As a user, I can see my active sessions and revoke them"
   - "As a user, I'm notified of logins from new devices"
3. **ADD**: Specify cookie-based token storage (not localStorage)
4. **CLARIFY**: JWT algorithm choice (recommend ES256)
5. **ADD**: Rate limiting on /auth endpoints (covered in future research)

---

## References

- [RFC 7519 - JWT](https://www.rfc-editor.org/rfc/rfc7519)
- [RFC 9449 - DPoP](https://www.rfc-editor.org/rfc/rfc9449)
- [Curity JWT Best Practices](https://curity.io/resources/learn/jwt-best-practices/)
- [Auth0 Refresh Token Rotation](https://auth0.com/blog/refresh-tokens-what-are-they-and-when-to-use-them/)
- [Slack Security Best Practices](https://docs.slack.dev/security/)
- [OWASP Session Management Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html)
