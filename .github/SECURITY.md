# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 1.x     | :white_check_mark: |
| < 1.0   | :x:                |

## Reporting a Vulnerability

We take security vulnerabilities seriously. If you discover a security issue, please report it responsibly.

### How to Report

**DO NOT** open a public GitHub issue for security vulnerabilities.

Instead, please email: **security@hearth.chat** (or the maintainer's email until this is set up)

Include:
1. Description of the vulnerability
2. Steps to reproduce
3. Potential impact
4. Any suggested fixes (optional)

### What to Expect

- **Acknowledgment:** Within 48 hours
- **Initial Assessment:** Within 7 days
- **Resolution Timeline:** Depends on severity
  - Critical: 24-48 hours
  - High: 7 days
  - Medium: 30 days
  - Low: 90 days

### Disclosure Policy

- We practice coordinated disclosure
- Credit will be given to reporters (unless anonymity is requested)
- We will not pursue legal action against good-faith security researchers

## Security Best Practices for Self-Hosters

### Required

1. **Use HTTPS** - Never run Hearth over plain HTTP in production
2. **Strong secrets** - Generate random SECRET_KEY (32+ bytes)
3. **Database security** - Use strong passwords, restrict network access
4. **Keep updated** - Apply security updates promptly

### Recommended

1. **Firewall** - Only expose necessary ports (80, 443)
2. **Reverse proxy** - Use Nginx/Caddy in front of Hearth
3. **Rate limiting** - Configure appropriate limits
4. **Backups** - Regular encrypted backups
5. **Monitoring** - Log aggregation and alerting

### Environment Variables

Sensitive values that should be treated as secrets:

```
SECRET_KEY
DATABASE_URL (contains password)
REDIS_URL (if password-protected)
STORAGE_ACCESS_KEY
STORAGE_SECRET_KEY
FUSIONAUTH_CLIENT_SECRET
FUSIONAUTH_API_KEY
```

Never commit these to version control. Use:
- Environment variables
- Docker secrets
- HashiCorp Vault
- Cloud secret managers (AWS Secrets Manager, etc.)

## Security Headers

Hearth includes these security headers by default:

```
X-Content-Type-Options: nosniff
X-Frame-Options: SAMEORIGIN
X-XSS-Protection: 1; mode=block
Referrer-Policy: strict-origin-when-cross-origin
```

Recommended additional headers for your reverse proxy:

```
Strict-Transport-Security: max-age=31536000; includeSubDomains
Content-Security-Policy: default-src 'self'
Permissions-Policy: camera=(), microphone=(), geolocation=()
```

## Known Limitations

1. **E2EE key backup** - Not yet implemented; key loss = message loss
2. **Admin access** - Server admins can see unencrypted channel metadata
3. **Federated trust** - Single-instance only; no federation security model yet

## Acknowledgments

We thank the following for responsibly disclosing vulnerabilities:

*No reports yet - be the first!*
