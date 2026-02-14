# Authentication API

## Endpoints

| Method | Path | Description | Auth |
|--------|------|-------------|------|
| POST | `/auth/register` | Create new account | No |
| POST | `/auth/login` | Login with credentials | No |
| POST | `/auth/refresh` | Refresh access token | No |
| POST | `/auth/logout` | Invalidate tokens | Yes |
| GET | `/auth/oauth/:provider` | OAuth redirect | No |
| GET | `/auth/oauth/:provider/callback` | OAuth callback | No |

---

## POST /auth/register

Create a new user account.

### Request Body

```json
{
  "email": "user@example.com",
  "username": "myuser",
  "display_name": "My Display Name",
  "password": "securepassword123",
  "invite_code": "abc123"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| email | string | Yes | Valid email address |
| username | string | Yes | 2-32 characters |
| display_name | string | No | Optional display name |
| password | string | Yes | Minimum 8 characters |
| invite_code | string | Conditional | Required if invite-only mode enabled |

### Response (201 Created)

```json
{
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "username": "myuser",
    "discriminator": "0001",
    "email": "user@example.com",
    "avatar_url": null,
    "flags": 0,
    "created_at": "2026-02-14T12:00:00Z"
  },
  "tokens": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
    "expires_in": 3600,
    "token_type": "Bearer"
  }
}
```

### Errors

| Code | Error | Description |
|------|-------|-------------|
| 400 | validation_error | Missing or invalid fields |
| 403 | registration_closed | Registration is disabled |
| 403 | invite_required | Valid invite code required |
| 409 | email_taken | Email already registered |
| 409 | username_taken | Username already in use |

---

## POST /auth/login

Authenticate with email and password.

### Request Body

```json
{
  "email": "user@example.com",
  "password": "securepassword123"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| email | string | Yes | Registered email |
| password | string | Yes | Account password |

### Response (200 OK)

```json
{
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "username": "myuser",
    "discriminator": "0001",
    "email": "user@example.com",
    "avatar_url": "https://...",
    "flags": 0,
    "created_at": "2026-02-14T12:00:00Z"
  },
  "tokens": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
    "expires_in": 3600,
    "token_type": "Bearer"
  }
}
```

### Errors

| Code | Error | Description |
|------|-------|-------------|
| 400 | validation_error | Missing email or password |
| 401 | invalid_credentials | Wrong email or password |

---

## POST /auth/refresh

Exchange a refresh token for a new access token.

### Request Body

```json
{
  "refresh_token": "eyJhbGciOiJIUzI1NiIs..."
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| refresh_token | string | Yes | Valid refresh token |

### Response (200 OK)

```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
  "expires_in": 3600,
  "token_type": "Bearer"
}
```

### Errors

| Code | Error | Description |
|------|-------|-------------|
| 400 | validation_error | Missing refresh_token |
| 401 | invalid_token | Expired or invalid refresh token |

---

## POST /auth/logout

Invalidate the current tokens.

### Headers

```
Authorization: Bearer <access_token>
```

### Request Body (optional)

```json
{
  "refresh_token": "eyJhbGciOiJIUzI1NiIs..."
}
```

### Response (204 No Content)

No response body.

---

## OAuth (Not Yet Implemented)

### GET /auth/oauth/:provider

Redirects to OAuth provider login page.

**Supported providers:** `google`, `github`, `discord`

### GET /auth/oauth/:provider/callback

Handles OAuth callback from provider.

**Query Parameters:**
- `code` - Authorization code from provider
- `state` - CSRF state token

### Errors

| Code | Error | Description |
|------|-------|-------------|
| 400 | invalid_provider | Unsupported OAuth provider |
| 400 | missing_code | Authorization code not provided |
| 501 | not_implemented | OAuth not yet available |

---

## Token Structure

### Access Token

JWT token valid for 1 hour. Include in `Authorization` header.

**Payload:**
```json
{
  "sub": "user-uuid",
  "username": "myuser",
  "exp": 1708430400,
  "iat": 1708426800
}
```

### Refresh Token

JWT token valid for 7 days. Use to obtain new access tokens.

---

## Best Practices

1. **Store tokens securely** - Use httpOnly cookies or secure storage
2. **Refresh proactively** - Refresh before access token expires
3. **Handle 401 errors** - Attempt refresh, then re-authenticate
4. **Logout on security events** - Password change, suspicious activity
