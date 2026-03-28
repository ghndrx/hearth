#!/bin/bash
#
# FusionAuth Setup Script for Hearth
# Automatically configures FusionAuth with optimal settings for Hearth
#
# Usage:
#   ./setup-fusionauth.sh
#   ./setup-fusionauth.sh --host http://fusionauth:9011 --hearth-url https://chat.example.com
#

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Defaults
FA_HOST="${FUSIONAUTH_HOST:-http://localhost:9011}"
HEARTH_URL="${HEARTH_URL:-http://localhost:8080}"
API_KEY=""
OUTPUT_FILE=".fusionauth-config"

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --host)
            FA_HOST="$2"
            shift 2
            ;;
        --hearth-url)
            HEARTH_URL="$2"
            shift 2
            ;;
        --api-key)
            API_KEY="$2"
            shift 2
            ;;
        --help|-h)
            echo "Usage: $0 [options]"
            echo ""
            echo "Options:"
            echo "  --host URL         FusionAuth host (default: http://localhost:9011)"
            echo "  --hearth-url URL   Hearth public URL (default: http://localhost:8080)"
            echo "  --api-key KEY      FusionAuth API key (will prompt if not provided)"
            exit 0
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            exit 1
            ;;
    esac
done

echo -e "${BLUE}"
cat << 'EOF'
  _____          _              _         _   _     
 |  ___|   _ ___(_) ___  _ __  / \  _   _| |_| |__  
 | |_ | | | / __| |/ _ \| '_ \/ _ \| | | | __| '_ \ 
 |  _|| |_| \__ \ | (_) | | | / ___ \ |_| | |_| | | |
 |_|   \__,_|___/_|\___/|_| |_/_/   \_\__,_|\__|_| |_|
                                                      
         Setup for Hearth
EOF
echo -e "${NC}"

# Check if FusionAuth is reachable
echo -e "${BLUE}Checking FusionAuth connectivity...${NC}"
if ! curl -sf "${FA_HOST}/api/status" > /dev/null 2>&1; then
    echo -e "${RED}Error: Cannot reach FusionAuth at ${FA_HOST}${NC}"
    echo "Make sure FusionAuth is running and accessible."
    exit 1
fi
echo -e "${GREEN}✓ FusionAuth is reachable${NC}"

# Get API key if not provided
if [ -z "$API_KEY" ]; then
    echo ""
    echo -e "${YELLOW}Please provide a FusionAuth API key with the following permissions:${NC}"
    echo "  - Application: Create, Read, Update"
    echo "  - User: Create, Read, Update"
    echo "  - Tenant: Read"
    echo ""
    echo "You can create an API key in FusionAuth: Settings → API Keys → Add"
    echo ""
    read -p "API Key: " API_KEY
    
    if [ -z "$API_KEY" ]; then
        echo -e "${RED}API key is required${NC}"
        exit 1
    fi
fi

# Test API key
echo -e "${BLUE}Validating API key...${NC}"
if ! curl -sf -H "Authorization: ${API_KEY}" "${FA_HOST}/api/tenant" > /dev/null 2>&1; then
    echo -e "${RED}Error: Invalid API key or insufficient permissions${NC}"
    exit 1
fi
echo -e "${GREEN}✓ API key is valid${NC}"

# Get tenant ID
TENANT_ID=$(curl -sf -H "Authorization: ${API_KEY}" "${FA_HOST}/api/tenant" | jq -r '.tenants[0].id')
echo -e "${GREEN}✓ Using tenant: ${TENANT_ID}${NC}"

# Generate secrets
CLIENT_SECRET=$(openssl rand -base64 32 | tr -d '/+=')
APP_ID=$(uuidgen | tr '[:upper:]' '[:lower:]')

# Create Hearth application
echo -e "${BLUE}Creating Hearth application...${NC}"

APP_CONFIG=$(cat << EOF
{
  "application": {
    "id": "${APP_ID}",
    "name": "Hearth",
    "tenantId": "${TENANT_ID}",
    "oauthConfiguration": {
      "authorizedRedirectURLs": [
        "${HEARTH_URL}/auth/callback",
        "${HEARTH_URL}/api/v1/auth/oauth/fusionauth/callback"
      ],
      "authorizedOriginURLs": [
        "${HEARTH_URL}"
      ],
      "clientSecret": "${CLIENT_SECRET}",
      "enabledGrants": [
        "authorization_code",
        "refresh_token"
      ],
      "generateRefreshTokens": true,
      "logoutBehavior": "RedirectOnly",
      "logoutURL": "${HEARTH_URL}",
      "requireClientAuthentication": true
    },
    "registrationConfiguration": {
      "enabled": true,
      "type": "basic"
    },
    "loginConfiguration": {
      "allowTokenRefresh": true,
      "generateRefreshTokens": true,
      "requireAuthentication": true
    },
    "jwtConfiguration": {
      "enabled": true,
      "timeToLiveInSeconds": 3600,
      "refreshTokenTimeToLiveInMinutes": 43200
    },
    "roles": [
      {
        "name": "admin",
        "description": "Hearth Administrator",
        "isDefault": false,
        "isSuperRole": false
      },
      {
        "name": "user",
        "description": "Regular Hearth User",
        "isDefault": true,
        "isSuperRole": false
      }
    ]
  }
}
EOF
)

RESULT=$(curl -sf -X POST \
    -H "Authorization: ${API_KEY}" \
    -H "Content-Type: application/json" \
    -d "${APP_CONFIG}" \
    "${FA_HOST}/api/application/${APP_ID}" 2>&1) || {
    echo -e "${RED}Failed to create application${NC}"
    echo "$RESULT"
    exit 1
}

echo -e "${GREEN}✓ Created Hearth application${NC}"

# Create email templates
echo -e "${BLUE}Creating email templates...${NC}"

# Email verification template
curl -sf -X POST \
    -H "Authorization: ${API_KEY}" \
    -H "Content-Type: application/json" \
    -d '{
      "emailTemplate": {
        "name": "Hearth - Email Verification",
        "fromEmail": "noreply@example.com",
        "subject": "Verify your Hearth account",
        "htmlTemplate": "<p>Welcome to Hearth!</p><p>Please verify your email by clicking: <a href=\"${verificationURL}\">Verify Email</a></p>",
        "textTemplate": "Welcome to Hearth!\n\nPlease verify your email: ${verificationURL}"
      }
    }' \
    "${FA_HOST}/api/email/template" > /dev/null 2>&1 || true

# Password reset template
curl -sf -X POST \
    -H "Authorization: ${API_KEY}" \
    -H "Content-Type: application/json" \
    -d '{
      "emailTemplate": {
        "name": "Hearth - Password Reset",
        "fromEmail": "noreply@example.com",
        "subject": "Reset your Hearth password",
        "htmlTemplate": "<p>Click to reset your password: <a href=\"${changePasswordURL}\">Reset Password</a></p>",
        "textTemplate": "Reset your password: ${changePasswordURL}"
      }
    }' \
    "${FA_HOST}/api/email/template" > /dev/null 2>&1 || true

echo -e "${GREEN}✓ Created email templates${NC}"

# Save configuration
echo -e "${BLUE}Saving configuration...${NC}"

cat > "${OUTPUT_FILE}" << EOF
# FusionAuth Configuration for Hearth
# Generated on $(date)
# Add these to your Hearth .env file

# Auth Provider
AUTH_PROVIDER=fusionauth

# FusionAuth Settings
FUSIONAUTH_HOST=${FA_HOST}
FUSIONAUTH_APPLICATION_ID=${APP_ID}
FUSIONAUTH_CLIENT_ID=${APP_ID}
FUSIONAUTH_CLIENT_SECRET=${CLIENT_SECRET}
FUSIONAUTH_API_KEY=${API_KEY}
FUSIONAUTH_TENANT_ID=${TENANT_ID}

# OAuth Redirect
FUSIONAUTH_REDIRECT_URI=${HEARTH_URL}/auth/callback
EOF

# Print summary
echo ""
echo -e "${GREEN}╔════════════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║            FusionAuth Setup Complete!                      ║${NC}"
echo -e "${GREEN}╚════════════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "  ${BLUE}Application ID:${NC}  ${APP_ID}"
echo -e "  ${BLUE}Client Secret:${NC}   ${CLIENT_SECRET:0:20}..."
echo -e "  ${BLUE}Tenant ID:${NC}       ${TENANT_ID}"
echo ""
echo -e "  ${BLUE}Configuration saved to:${NC} ${OUTPUT_FILE}"
echo ""
echo -e "${YELLOW}Next steps:${NC}"
echo "  1. Add the configuration to your Hearth .env file:"
echo "     cat ${OUTPUT_FILE} >> .env"
echo ""
echo "  2. Configure SMTP in FusionAuth for email verification:"
echo "     ${FA_HOST}/admin/tenant/edit/${TENANT_ID}#email"
echo ""
echo "  3. Restart Hearth to apply changes:"
echo "     docker compose restart hearth"
echo ""
echo -e "${GREEN}FusionAuth is ready for Hearth! 🔥${NC}"
