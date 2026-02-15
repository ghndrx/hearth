#!/bin/bash
# Hearth E2E Test Suite
# Tests critical flows against live services

# Don't exit on error - we want to run all tests
set +e

API_URL="${API_URL:-http://localhost:8080}"
WS_URL="${WS_URL:-ws://localhost:8080}"
FRONTEND_URL="${FRONTEND_URL:-http://localhost:3000}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counters
PASSED=0
FAILED=0
TOTAL=0

# Test data
TEST_EMAIL="e2e_test_$(date +%s)@test.local"
TEST_USERNAME="e2etest$(date +%s)"
TEST_PASSWORD="TestPass123!"
ACCESS_TOKEN=""
REFRESH_TOKEN=""
USER_ID=""
SERVER_ID=""
CHANNEL_ID=""
MESSAGE_ID=""

# Helper functions
log_test() {
    ((TOTAL++))
    echo -e "\n${YELLOW}[TEST $TOTAL]${NC} $1"
}

log_pass() {
    ((PASSED++))
    echo -e "${GREEN}✓ PASSED${NC}: $1"
}

log_fail() {
    ((FAILED++))
    echo -e "${RED}✗ FAILED${NC}: $1"
}

check_response() {
    local response="$1"
    local expected="$2"
    local description="$3"
    
    if echo "$response" | grep -q "$expected"; then
        log_pass "$description"
        return 0
    else
        log_fail "$description"
        echo "  Expected: $expected"
        echo "  Got: $response"
        return 1
    fi
}

measure_time() {
    local start=$(date +%s%N)
    eval "$1"
    local end=$(date +%s%N)
    local duration=$(( (end - start) / 1000000 ))
    echo "$duration"
}

# Test: Health Check
test_health() {
    log_test "API Health Check"
    local response=$(curl -s "$API_URL/health")
    check_response "$response" '"status":"ok"' "Health endpoint returns ok"
}

# Test: Frontend Accessibility
test_frontend() {
    log_test "Frontend Accessibility"
    local status=$(curl -s -o /dev/null -w "%{http_code}" "$FRONTEND_URL")
    if [ "$status" == "200" ]; then
        log_pass "Frontend returns HTTP 200"
    else
        log_fail "Frontend returns HTTP $status"
    fi
}

# Test: User Registration
test_register() {
    log_test "User Registration"
    local response=$(curl -s -X POST "$API_URL/api/v1/auth/register" \
        -H "Content-Type: application/json" \
        -d "{\"email\":\"$TEST_EMAIL\",\"username\":\"$TEST_USERNAME\",\"password\":\"$TEST_PASSWORD\"}")
    
    if echo "$response" | grep -q '"access_token"'; then
        ACCESS_TOKEN=$(echo "$response" | jq -r '.access_token')
        REFRESH_TOKEN=$(echo "$response" | jq -r '.refresh_token')
        USER_ID=$(echo "$response" | jq -r '.user.id')
        log_pass "User registered successfully"
        echo "  User ID: $USER_ID"
    else
        log_fail "User registration"
        echo "  Response: $response"
        return 1
    fi
}

# Test: Duplicate Registration Prevention
test_duplicate_register() {
    log_test "Duplicate Registration Prevention"
    local response=$(curl -s -X POST "$API_URL/api/v1/auth/register" \
        -H "Content-Type: application/json" \
        -d "{\"email\":\"$TEST_EMAIL\",\"username\":\"$TEST_USERNAME\",\"password\":\"$TEST_PASSWORD\"}")
    
    if echo "$response" | grep -qi "exists\|already\|taken\|duplicate"; then
        log_pass "Duplicate registration rejected"
    else
        log_fail "Duplicate registration not rejected"
        echo "  Response: $response"
    fi
}

# Test: User Login
test_login() {
    log_test "User Login"
    local response=$(curl -s -X POST "$API_URL/api/v1/auth/login" \
        -H "Content-Type: application/json" \
        -d "{\"email\":\"$TEST_EMAIL\",\"password\":\"$TEST_PASSWORD\"}")
    
    if echo "$response" | grep -q '"access_token"'; then
        ACCESS_TOKEN=$(echo "$response" | jq -r '.access_token')
        REFRESH_TOKEN=$(echo "$response" | jq -r '.refresh_token')
        log_pass "Login successful"
    else
        log_fail "Login failed"
        echo "  Response: $response"
        return 1
    fi
}

# Test: Get Current User
test_get_me() {
    log_test "Get Current User (/users/@me)"
    local response=$(curl -s "$API_URL/api/v1/users/@me" \
        -H "Authorization: Bearer $ACCESS_TOKEN")
    
    if echo "$response" | grep -q "\"username\":\"$TEST_USERNAME\""; then
        log_pass "Current user retrieved"
    else
        log_fail "Failed to get current user"
        echo "  Response: $response"
    fi
}

# Test: Token Refresh
test_refresh_token() {
    log_test "Token Refresh"
    local response=$(curl -s -X POST "$API_URL/api/v1/auth/refresh" \
        -H "Content-Type: application/json" \
        -d "{\"refresh_token\":\"$REFRESH_TOKEN\"}")
    
    if echo "$response" | grep -q '"access_token"'; then
        ACCESS_TOKEN=$(echo "$response" | jq -r '.access_token')
        log_pass "Token refreshed"
    else
        log_fail "Token refresh failed"
        echo "  Response: $response"
    fi
}

# Test: Create Server
test_create_server() {
    log_test "Create Server"
    local response=$(curl -s -X POST "$API_URL/api/v1/servers" \
        -H "Authorization: Bearer $ACCESS_TOKEN" \
        -H "Content-Type: application/json" \
        -d '{"name":"E2E Test Server"}')
    
    if echo "$response" | grep -q '"id"'; then
        SERVER_ID=$(echo "$response" | jq -r '.id')
        CHANNEL_ID=$(echo "$response" | jq -r '.channels[0].id // empty')
        log_pass "Server created"
        echo "  Server ID: $SERVER_ID"
        echo "  Default Channel ID: $CHANNEL_ID"
    else
        log_fail "Server creation failed"
        echo "  Response: $response"
        return 1
    fi
}

# Test: Get Server
test_get_server() {
    log_test "Get Server"
    local response=$(curl -s "$API_URL/api/v1/servers/$SERVER_ID" \
        -H "Authorization: Bearer $ACCESS_TOKEN")
    
    if echo "$response" | grep -q '"name":"E2E Test Server"'; then
        log_pass "Server retrieved"
    else
        log_fail "Failed to get server"
        echo "  Response: $response"
    fi
}

# Test: Create Channel
test_create_channel() {
    log_test "Create Channel"
    local response=$(curl -s -X POST "$API_URL/api/v1/servers/$SERVER_ID/channels" \
        -H "Authorization: Bearer $ACCESS_TOKEN" \
        -H "Content-Type: application/json" \
        -d '{"name":"e2e-test-channel","type":"text"}')
    
    if echo "$response" | grep -q '"id"'; then
        CHANNEL_ID=$(echo "$response" | jq -r '.id')
        log_pass "Channel created"
        echo "  Channel ID: $CHANNEL_ID"
    else
        log_fail "Channel creation failed"
        echo "  Response: $response"
    fi
}

# Test: Send Message
test_send_message() {
    log_test "Send Message"
    local response=$(curl -s -X POST "$API_URL/api/v1/channels/$CHANNEL_ID/messages" \
        -H "Authorization: Bearer $ACCESS_TOKEN" \
        -H "Content-Type: application/json" \
        -d '{"content":"Hello from E2E test!"}')
    
    if echo "$response" | grep -q '"id"'; then
        MESSAGE_ID=$(echo "$response" | jq -r '.id')
        log_pass "Message sent"
        echo "  Message ID: $MESSAGE_ID"
    else
        log_fail "Message sending failed"
        echo "  Response: $response"
        return 1
    fi
}

# Test: Get Messages
test_get_messages() {
    log_test "Get Messages"
    local response=$(curl -s "$API_URL/api/v1/channels/$CHANNEL_ID/messages" \
        -H "Authorization: Bearer $ACCESS_TOKEN")
    
    if echo "$response" | grep -q '"Hello from E2E test!"'; then
        log_pass "Messages retrieved"
    else
        log_fail "Failed to get messages"
        echo "  Response: $response"
    fi
}

# Test: Edit Message
test_edit_message() {
    log_test "Edit Message"
    local response=$(curl -s -X PATCH "$API_URL/api/v1/channels/$CHANNEL_ID/messages/$MESSAGE_ID" \
        -H "Authorization: Bearer $ACCESS_TOKEN" \
        -H "Content-Type: application/json" \
        -d '{"content":"Edited E2E test message"}')
    
    if echo "$response" | grep -q '"Edited E2E test message"'; then
        log_pass "Message edited"
    else
        log_fail "Message edit failed"
        echo "  Response: $response"
    fi
}

# Test: Delete Message
test_delete_message() {
    log_test "Delete Message"
    local status=$(curl -s -o /dev/null -w "%{http_code}" \
        -X DELETE "$API_URL/api/v1/channels/$CHANNEL_ID/messages/$MESSAGE_ID" \
        -H "Authorization: Bearer $ACCESS_TOKEN")
    
    if [ "$status" == "200" ] || [ "$status" == "204" ]; then
        log_pass "Message deleted (HTTP $status)"
    else
        log_fail "Message deletion failed (HTTP $status)"
    fi
}

# Test: WebSocket Connection
test_websocket() {
    log_test "WebSocket Gateway Connection"
    
    # Check if wscat is available
    if ! command -v wscat &> /dev/null; then
        echo "  Skipping: wscat not installed"
        return
    fi
    
    # Test WebSocket handshake
    local response=$(timeout 5 wscat -c "$WS_URL/gateway" -x '{"op":2,"d":{"token":"'"$ACCESS_TOKEN"'"}}' 2>&1 || true)
    
    if echo "$response" | grep -qi "connected\|ready\|hello"; then
        log_pass "WebSocket connection established"
    else
        log_fail "WebSocket connection failed"
        echo "  Response: $response"
    fi
}

# Test: Response Time
test_response_times() {
    log_test "API Response Times"
    
    # Health check timing
    local start=$(date +%s%N)
    curl -s "$API_URL/health" > /dev/null
    local end=$(date +%s%N)
    local health_time=$(( (end - start) / 1000000 ))
    
    # Auth timing
    start=$(date +%s%N)
    curl -s "$API_URL/api/v1/users/@me" -H "Authorization: Bearer $ACCESS_TOKEN" > /dev/null
    end=$(date +%s%N)
    local auth_time=$(( (end - start) / 1000000 ))
    
    echo "  Health check: ${health_time}ms"
    echo "  Authenticated request: ${auth_time}ms"
    
    if [ "$health_time" -lt 200 ] && [ "$auth_time" -lt 500 ]; then
        log_pass "Response times acceptable"
    else
        log_fail "Response times too slow"
    fi
}

# Test: Error Handling
test_error_handling() {
    log_test "Error Handling"
    
    # Test 401 for unauthenticated
    local status=$(curl -s -o /dev/null -w "%{http_code}" "$API_URL/api/v1/users/@me")
    if [ "$status" == "401" ]; then
        log_pass "Returns 401 for unauthenticated request"
    else
        log_fail "Expected 401, got $status"
    fi
    
    # Test 404 for non-existent
    status=$(curl -s -o /dev/null -w "%{http_code}" \
        "$API_URL/api/v1/servers/00000000-0000-0000-0000-000000000000" \
        -H "Authorization: Bearer $ACCESS_TOKEN")
    if [ "$status" == "404" ]; then
        log_pass "Returns 404 for non-existent resource"
    else
        log_fail "Expected 404, got $status"
    fi
}

# Test: Server Roles
test_server_roles() {
    log_test "Server Roles"
    local response=$(curl -s "$API_URL/api/v1/servers/$SERVER_ID/roles" \
        -H "Authorization: Bearer $ACCESS_TOKEN")
    
    if echo "$response" | grep -q '"name"'; then
        log_pass "Roles retrieved"
    else
        log_fail "Failed to get roles"
        echo "  Response: $response"
    fi
}

# Test: Logout
test_logout() {
    log_test "User Logout"
    local status=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$API_URL/api/v1/auth/logout" \
        -H "Authorization: Bearer $ACCESS_TOKEN")
    
    if [ "$status" == "200" ] || [ "$status" == "204" ]; then
        log_pass "Logout successful"
    else
        log_fail "Logout failed (HTTP $status)"
    fi
}

# Cleanup: Delete Server
cleanup() {
    if [ -n "$SERVER_ID" ]; then
        echo -e "\n${YELLOW}[CLEANUP]${NC} Deleting test server..."
        curl -s -X DELETE "$API_URL/api/v1/servers/$SERVER_ID" \
            -H "Authorization: Bearer $ACCESS_TOKEN" > /dev/null 2>&1 || true
    fi
}

# Main test execution
main() {
    echo "======================================"
    echo "  Hearth E2E Test Suite"
    echo "======================================"
    echo "API: $API_URL"
    echo "WS:  $WS_URL"
    echo "Frontend: $FRONTEND_URL"
    echo ""
    
    # Run tests
    test_health
    test_frontend
    test_register
    test_duplicate_register
    test_login
    test_get_me
    test_refresh_token
    test_create_server
    test_get_server
    test_create_channel
    test_send_message
    test_get_messages
    test_edit_message
    test_delete_message
    test_server_roles
    test_response_times
    test_error_handling
    test_websocket
    test_logout
    
    # Cleanup
    cleanup
    
    # Summary
    echo ""
    echo "======================================"
    echo "  Test Results"
    echo "======================================"
    echo -e "Passed: ${GREEN}$PASSED${NC}"
    echo -e "Failed: ${RED}$FAILED${NC}"
    echo -e "Total:  $TOTAL"
    echo ""
    
    if [ "$FAILED" -gt 0 ]; then
        echo -e "${RED}Some tests failed!${NC}"
        exit 1
    else
        echo -e "${GREEN}All tests passed!${NC}"
        exit 0
    fi
}

main "$@"
