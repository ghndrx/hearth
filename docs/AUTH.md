# Hearth — Authentication Architecture

**Version:** 1.0  
**Last Updated:** 2026-02-11

---

## Overview

Hearth supports multiple authentication providers through a pluggable architecture. Operators can choose:

| Provider | Best For | Complexity |
|----------|----------|------------|
| **Native** | Simple deployments, homelabs | ⭐ Easy |
| **FusionAuth** | Enterprise, SSO, compliance | ⭐⭐ Medium |
| **Authentik** | Existing Authentik users | ⭐⭐ Medium |
| **Keycloak** | Enterprise, existing Keycloak | ⭐⭐⭐ Complex |
| **Generic OIDC** | Any OIDC-compatible IdP | ⭐⭐ Medium |

---

## Architecture

### Pluggable Provider Interface

```go
// AuthProvider defines the interface for authentication backends
type AuthProvider interface {
    // Provider metadata
    Name() string
    Type() ProviderType // native, fusionauth, oidc, etc.
    
    // User lifecycle
    Register(ctx context.Context, req RegisterRequest) (*User, error)
    Login(ctx context.Context, req LoginRequest) (*AuthResult, error)
    Logout(ctx context.Context, sessionID string) error
    RefreshToken(ctx context.Context, refreshToken string) (*AuthResult, error)
    
    // Password management (native only, others delegate to IdP)
    ChangePassword(ctx context.Context, userID string, req ChangePasswordRequest) error
    ResetPassword(ctx context.Context, email string) error
    ConfirmPasswordReset(ctx context.Context, token, newPassword string) error
    
    // MFA (may delegate to IdP)
    EnableMFA(ctx context.Context, userID string) (*MFASetup, error)
    VerifyMFA(ctx context.Context, userID string, code string) error
    DisableMFA(ctx context.Context, userID string) error
    
    // Session management
    GetSessions(ctx context.Context, userID string) ([]Session, error)
    RevokeSession(ctx context.Context, sessionID string) error
    RevokeAllSessions(ctx context.Context, userID string) error
    
    // User info (may sync from IdP)
    GetUser(ctx context.Context, userID string) (*User, error)
    UpdateUser(ctx context.Context, userID string, req UpdateUserRequest) (*User, error)
    DeleteUser(ctx context.Context, userID string) error
    
    // OAuth2/OIDC flows (for external IdP)
    GetAuthorizationURL(ctx context.Context, state string) (string, error)
    HandleCallback(ctx context.Context, code, state string) (*AuthResult, error)
}

type ProviderType string

const (
    ProviderNative     ProviderType = "native"
    ProviderFusionAuth ProviderType = "fusionauth"
    ProviderAuthentik  ProviderType = "authentik"
    ProviderKeycloak   ProviderType = "keycloak"
    ProviderOIDC       ProviderType = "oidc"
)
```

### Request/Response Types

```go
type RegisterRequest struct {
    Email    string `json:"email"`
    Username string `json:"username"`
    Password string `json:"password"`
    
    // Optional OAuth token for linking
    OAuthToken string `json:"oauth_token,omitempty"`
}

type LoginRequest struct {
    Email    string `json:"email"`
    Password string `json:"password"`
    MFACode  string `json:"mfa_code,omitempty"`
    
    // For OAuth flows
    OAuthCode string `json:"oauth_code,omitempty"`
}

type AuthResult struct {
    User         *User  `json:"user"`
    AccessToken  string `json:"access_token"`
    RefreshToken string `json:"refresh_token"`
    ExpiresIn    int    `json:"expires_in"`
    TokenType    string `json:"token_type"` // "Bearer"
    
    // For MFA challenge
    MFARequired bool   `json:"mfa_required,omitempty"`
    MFAToken    string `json:"mfa_token,omitempty"`
}

type MFASetup struct {
    Secret    string `json:"secret"`
    QRCodeURL string `json:"qr_code_url"`
    BackupCodes []string `json:"backup_codes"`
}
```

---

## Native Authentication

Built-in authentication with no external dependencies.

### Features
- Email + password registration
- Argon2id password hashing
- JWT access tokens (15 min)
- Opaque refresh tokens (7 days)
- TOTP-based MFA
- Email verification
- Password reset flow

### Configuration

```yaml
auth:
  provider: native
  
  native:
    # Password requirements
    password_min_length: 8
    password_require_uppercase: false
    password_require_number: false
    password_require_special: false
    
    # Token settings
    access_token_ttl: 15m
    refresh_token_ttl: 7d
    
    # Security
    max_login_attempts: 5
    lockout_duration: 15m
    
    # Email verification
    require_email_verification: false
    
    # MFA
    mfa_issuer: "Hearth"
```

### Database Tables

Uses Hearth's built-in tables:
- `users` - User accounts
- `sessions` - Refresh tokens
- `mfa_secrets` - TOTP secrets
- `password_reset_tokens` - Reset flow

### Flow Diagrams

**Registration:**
```
Client                    Hearth                    Database
  │                         │                          │
  │  POST /auth/register    │                          │
  │  {email, username, pw}  │                          │
  │────────────────────────>│                          │
  │                         │  Hash password           │
  │                         │  Generate discriminator  │
  │                         │────────────────────────>│
  │                         │                          │
  │                         │  Create user             │
  │                         │<────────────────────────│
  │                         │                          │
  │                         │  Generate tokens         │
  │                         │────────────────────────>│
  │                         │                          │
  │  {user, access, refresh}│                          │
  │<────────────────────────│                          │
```

**Login:**
```
Client                    Hearth                    Database
  │                         │                          │
  │  POST /auth/login       │                          │
  │  {email, password}      │                          │
  │────────────────────────>│                          │
  │                         │  Find user by email      │
  │                         │────────────────────────>│
  │                         │<────────────────────────│
  │                         │                          │
  │                         │  Verify password         │
  │                         │                          │
  │                         │  Check MFA required?     │
  │                         │                          │
  │  {mfa_required: true}   │  (if MFA enabled)       │
  │<────────────────────────│                          │
  │                         │                          │
  │  POST /auth/mfa/verify  │                          │
  │  {mfa_token, code}      │                          │
  │────────────────────────>│                          │
  │                         │  Verify TOTP             │
  │                         │                          │
  │  {user, access, refresh}│                          │
  │<────────────────────────│                          │
```

---

## FusionAuth Integration

Enterprise-grade authentication via FusionAuth.

### Features
- All FusionAuth features available
- User sync to Hearth database
- SSO with other FusionAuth apps
- Social login (Google, GitHub, etc.)
- Advanced MFA (WebAuthn, email, SMS)
- Passwordless login
- Consent management
- Audit logging

### Prerequisites
- FusionAuth instance (self-hosted or cloud)
- FusionAuth Application configured
- API key with appropriate permissions

### Configuration

```yaml
auth:
  provider: fusionauth
  
  fusionauth:
    # FusionAuth server
    host: https://auth.example.com
    
    # Application credentials
    application_id: "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
    client_id: "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
    client_secret: "${FUSIONAUTH_CLIENT_SECRET}"
    
    # API key for admin operations
    api_key: "${FUSIONAUTH_API_KEY}"
    
    # Tenant (if multi-tenant)
    tenant_id: "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
    
    # User sync
    sync_profile_fields: true  # Sync avatar, bio to Hearth
    sync_interval: 5m          # How often to sync changes
    
    # OAuth2/OIDC settings
    scopes:
      - openid
      - profile
      - email
    
    # Redirect URIs
    redirect_uri: https://chat.example.com/auth/callback
    post_logout_uri: https://chat.example.com/
```

### FusionAuth Setup

1. **Create Application:**
   ```json
   {
     "application": {
       "name": "Hearth",
       "oauthConfiguration": {
         "authorizedRedirectURLs": [
           "https://chat.example.com/auth/callback"
         ],
         "clientSecret": "...",
         "enabledGrants": [
           "authorization_code",
           "refresh_token"
         ],
         "generateRefreshTokens": true,
         "logoutURL": "https://chat.example.com/"
       },
       "registrationConfiguration": {
         "enabled": true,
         "type": "basic"
       }
     }
   }
   ```

2. **Configure Roles** (mapped to Hearth):
   - `hearth:admin` → Server administrators
   - `hearth:user` → Regular users

3. **Set up Webhooks** (optional):
   - User registration → Sync to Hearth
   - User update → Update Hearth profile
   - User delete → Handle account deletion

### User Sync

When using FusionAuth, Hearth maintains a local user record synced from FusionAuth:

```go
type FusionAuthUser struct {
    // FusionAuth fields
    FAUserID string `json:"fa_user_id"`
    
    // Synced to Hearth User
    Email    string `json:"email"`
    Username string `json:"username"` // From FA displayName or username
    Avatar   string `json:"avatar"`   // From FA imageUrl
    Verified bool   `json:"verified"` // From FA verified
}

// Sync on login
func (p *FusionAuthProvider) syncUser(faUser *fusionauth.User) (*User, error) {
    // Find or create local user
    user, err := p.userRepo.FindByExternalID("fusionauth", faUser.ID)
    if err == ErrNotFound {
        // Create new Hearth user from FA data
        user = &User{
            ExternalProvider: "fusionauth",
            ExternalID:       faUser.ID,
            Email:            faUser.Email,
            Username:         faUser.Username,
            AvatarURL:        faUser.ImageURL,
            Verified:         faUser.Verified,
        }
        return p.userRepo.Create(user)
    }
    
    // Update existing user
    user.Email = faUser.Email
    user.AvatarURL = faUser.ImageURL
    return p.userRepo.Update(user)
}
```

### Flow Diagrams

**Login with FusionAuth:**
```
Client              Hearth                 FusionAuth
  │                   │                        │
  │  GET /auth/login  │                        │
  │──────────────────>│                        │
  │                   │                        │
  │  Redirect to FA   │                        │
  │<──────────────────│                        │
  │                   │                        │
  │  FA Login Page    │                        │
  │───────────────────────────────────────────>│
  │                   │                        │
  │  (user logs in)   │                        │
  │<───────────────────────────────────────────│
  │                   │                        │
  │  Callback + code  │                        │
  │──────────────────>│                        │
  │                   │  Exchange code         │
  │                   │───────────────────────>│
  │                   │                        │
  │                   │  Tokens + user info    │
  │                   │<───────────────────────│
  │                   │                        │
  │                   │  Sync user to Hearth   │
  │                   │                        │
  │  {user, tokens}   │                        │
  │<──────────────────│                        │
```

---

## Generic OIDC Provider

For any OpenID Connect compatible identity provider.

### Configuration

```yaml
auth:
  provider: oidc
  
  oidc:
    # Discovery endpoint (auto-configures everything)
    issuer: https://auth.example.com
    
    # Or manual configuration
    authorization_endpoint: https://auth.example.com/oauth2/authorize
    token_endpoint: https://auth.example.com/oauth2/token
    userinfo_endpoint: https://auth.example.com/oauth2/userinfo
    jwks_uri: https://auth.example.com/.well-known/jwks.json
    
    # Client credentials
    client_id: "hearth"
    client_secret: "${OIDC_CLIENT_SECRET}"
    
    # Scopes
    scopes:
      - openid
      - profile
      - email
    
    # Claim mapping
    claims:
      user_id: sub
      email: email
      username: preferred_username
      avatar: picture
      display_name: name
```

### Supported Providers

Any OIDC-compliant provider works:
- **Authentik** - Open source, Kubernetes-native
- **Keycloak** - Red Hat, enterprise features
- **Auth0** - Cloud-hosted
- **Okta** - Enterprise
- **Azure AD** - Microsoft
- **Google Workspace** - Google

---

## Hybrid Mode

Run multiple providers simultaneously for flexibility.

### Configuration

```yaml
auth:
  # Primary provider for new registrations
  primary_provider: native
  
  # Enable additional providers
  providers:
    native:
      enabled: true
    
    fusionauth:
      enabled: true
      # ... fusionauth config
    
    oidc:
      # Multiple OIDC providers
      - name: google
        enabled: true
        issuer: https://accounts.google.com
        client_id: "..."
        client_secret: "..."
      
      - name: github
        enabled: true
        authorization_endpoint: https://github.com/login/oauth/authorize
        token_endpoint: https://github.com/login/oauth/access_token
        userinfo_endpoint: https://api.github.com/user
        client_id: "..."
        client_secret: "..."
  
  # Account linking
  allow_account_linking: true  # Link multiple providers to one account
  require_email_match: true    # Only link if emails match
```

### Login Flow with Multiple Providers

```
┌─────────────────────────────────────────────┐
│              Login Page                      │
├─────────────────────────────────────────────┤
│                                             │
│  ┌─────────────────────────────────────┐   │
│  │  Email: [________________]          │   │
│  │  Password: [________________]       │   │
│  │                                     │   │
│  │  [        Sign In        ]          │   │
│  └─────────────────────────────────────┘   │
│                                             │
│  ─────────── OR ───────────                │
│                                             │
│  [🔐 Sign in with FusionAuth]              │
│  [🔴 Sign in with Google]                  │
│  [⚫ Sign in with GitHub]                  │
│                                             │
│  Don't have an account? [Register]         │
│                                             │
└─────────────────────────────────────────────┘
```

---

## Token Management

### Access Tokens (JWT)

Regardless of provider, Hearth issues its own JWTs for API access:

```go
type HearthClaims struct {
    jwt.RegisteredClaims
    
    UserID    string `json:"uid"`
    Username  string `json:"usr"`
    SessionID string `json:"sid"`
    
    // Provider info
    Provider   string `json:"prv"`        // native, fusionauth, etc.
    ExternalID string `json:"ext,omitempty"` // ID in external provider
    
    // Permissions (optional, for faster checks)
    Flags int64 `json:"flg,omitempty"`
}

// Example token payload
{
  "iss": "hearth",
  "sub": "user_abc123",
  "aud": ["hearth-api"],
  "exp": 1707619200,
  "iat": 1707618300,
  "uid": "user_abc123",
  "usr": "johndoe",
  "sid": "session_xyz789",
  "prv": "fusionauth",
  "ext": "fa_user_123"
}
```

### Token Validation

```go
func (m *AuthMiddleware) Validate(token string) (*HearthClaims, error) {
    // Parse and validate JWT
    claims, err := jwt.ParseWithClaims(token, &HearthClaims{}, m.keyFunc)
    if err != nil {
        return nil, ErrInvalidToken
    }
    
    // Check if session is still valid
    session, err := m.sessionRepo.FindByID(claims.SessionID)
    if err != nil || session.RevokedAt != nil {
        return nil, ErrSessionRevoked
    }
    
    return claims, nil
}
```

---

## Migration Guide

### Native → FusionAuth

1. **Export users from Hearth:**
   ```bash
   hearth export users --format fusionauth > users.json
   ```

2. **Import to FusionAuth:**
   ```bash
   fusionauth-import users.json
   ```

3. **Update config:**
   ```yaml
   auth:
     provider: fusionauth
     # ... config
   ```

4. **Run migration:**
   ```bash
   hearth migrate auth --from native --to fusionauth
   ```

### FusionAuth → Native

1. **Export from FusionAuth**
2. **Import to Hearth with password reset:**
   ```bash
   hearth import users --from fusionauth --require-password-reset
   ```

---

## Security Considerations

### Token Storage (Client)
- Access token: Memory only (never localStorage)
- Refresh token: httpOnly cookie (preferred) or secure storage

### Token Rotation
- Refresh tokens are single-use
- Each refresh issues new access + refresh pair
- Old refresh token immediately invalidated

### Session Limits
- Max 10 active sessions per user (configurable)
- Oldest session removed when limit exceeded

### Audit Logging
All auth events logged:
- Registration
- Login (success/failure)
- Logout
- Password change
- MFA enable/disable
- Session revocation

---

## API Reference

### Endpoints

```
# Native auth
POST   /api/v1/auth/register
POST   /api/v1/auth/login
POST   /api/v1/auth/logout
POST   /api/v1/auth/refresh
POST   /api/v1/auth/password/reset
POST   /api/v1/auth/password/reset/confirm
POST   /api/v1/auth/mfa/enable
POST   /api/v1/auth/mfa/verify
DELETE /api/v1/auth/mfa

# OAuth/OIDC
GET    /api/v1/auth/oauth/{provider}
GET    /api/v1/auth/oauth/{provider}/callback
POST   /api/v1/auth/oauth/{provider}/link
DELETE /api/v1/auth/oauth/{provider}/unlink

# Sessions
GET    /api/v1/auth/sessions
DELETE /api/v1/auth/sessions/{id}
DELETE /api/v1/auth/sessions
```

---

*End of Authentication Architecture*
