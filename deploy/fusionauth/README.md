# FusionAuth Deployment for Hearth

Optional SSO/OIDC identity provider for Hearth using [FusionAuth](https://fusionauth.io/).

## Quick Start (Standalone Stack)

The fastest way to get FusionAuth running:

```bash
cd deploy/fusionauth
./quickstart.sh
```

This prints a status message and runs `kickstart.sh`, which:
- Generates random secrets and creates `.env` from the example template
- Starts FusionAuth and PostgreSQL via docker compose
- Polls the health endpoint until FusionAuth is ready
- Prints the admin URL and next steps

The script is idempotent -- safe to run multiple times. If `.env` already exists it will be reused.

### Manual Setup

If you prefer to configure things yourself:

```bash
cd deploy/fusionauth

# Configure environment
cp .env.fusionauth.example .env
# Edit .env - at minimum set FUSIONAUTH_DB_PASSWORD

# Create the shared network (if not already created)
docker network create hearth-network

# Start FusionAuth
docker compose -f docker-compose.fusionauth.yml up -d

# With Elasticsearch for advanced search:
docker compose -f docker-compose.fusionauth.yml --profile search up -d
```

FusionAuth admin UI will be available at `http://localhost:9011`.

## Integrated Profile (with main Hearth stack)

If you prefer running everything from the main docker-compose:

```bash
cd deploy/docker-compose
# Add FusionAuth vars to your .env (see .env.fusionauth.example)

# Start Hearth with FusionAuth profile
docker compose --profile fusionauth up -d
```

## FusionAuth Setup

Once FusionAuth is running, complete the setup wizard at `http://localhost:9011`:

1. **Create admin account** - Set up your FusionAuth admin user
2. **Create an Application**:
   - Navigate to Applications > Add
   - Name: `Hearth`
   - OAuth tab:
     - Authorized redirect URLs: `{HEARTH_PUBLIC_URL}/api/v1/auth/oauth/fusionauth/callback`
     - Authorized request origin URLs: `{HEARTH_PUBLIC_URL}`
     - Logout URL: `{HEARTH_PUBLIC_URL}`
   - Save and note the **Application ID**, **Client ID**, and **Client Secret**
3. **Create an API Key**:
   - Navigate to Settings > API Keys > Add
   - Create a key with appropriate permissions (or superuser for dev)
   - Save the generated key

## Hearth Configuration

Add these to your main Hearth `.env` or environment variables:

```bash
# Switch auth provider to FusionAuth
AUTH_PROVIDER=fusionauth

# FusionAuth connection (use container name on Docker network)
FUSIONAUTH_HOST=http://hearth-fusionauth:9011
FUSIONAUTH_APPLICATION_ID=<from-step-2>
FUSIONAUTH_CLIENT_ID=<from-step-2>
FUSIONAUTH_CLIENT_SECRET=<from-step-2>
FUSIONAUTH_API_KEY=<from-step-3>
```

## Architecture

```
                    +-----------------+
                    |   Hearth App    |
                    |   :8080         |
                    +--------+--------+
                             |
                    (hearth-network)
                             |
                    +--------+--------+
                    |   FusionAuth    |
                    |   :9011         |
                    +--------+--------+
                             |
                  (fusionauth-internal)
                             |
              +--------------+--------------+
              |                             |
     +--------+--------+         +---------+--------+
     |   PostgreSQL     |         |  Elasticsearch   |
     |   (fusionauth)   |         |  (optional)      |
     +------------------+         +------------------+
```

- **hearth-network**: Shared network allowing Hearth to reach FusionAuth
- **fusionauth-internal**: Private network for FusionAuth's database and search

## Volumes

| Volume | Purpose |
|--------|---------|
| `fusionauth-config` | FusionAuth configuration |
| `fusionauth-db-data` | PostgreSQL data for FusionAuth |
| `fusionauth-es-data` | Elasticsearch data (if using search profile) |

## Production Notes

- Set `FUSIONAUTH_RUNTIME_MODE=production` in `.env`
- Use strong, unique passwords for `FUSIONAUTH_DB_PASSWORD`
- Generate a proper `FUSIONAUTH_API_KEY` with minimal required permissions
- Consider placing FusionAuth behind a reverse proxy with TLS
- Back up `fusionauth-db-data` volume regularly
- Elasticsearch is optional - `database` search mode works fine for most deployments

## Troubleshooting

```bash
# Check FusionAuth logs
docker compose -f docker-compose.fusionauth.yml logs -f fusionauth

# Check database connectivity
docker compose -f docker-compose.fusionauth.yml logs fusionauth-db

# Verify FusionAuth health
curl http://localhost:9011/api/status

# Restart FusionAuth
docker compose -f docker-compose.fusionauth.yml restart fusionauth
```
