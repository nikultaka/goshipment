#!/bin/bash

# API Test Script
# Tests all endpoints from swagger documentation

BASE_URL="http://localhost:8081/api/v1"
ADMIN_TOKEN=""
USER_TOKEN=""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Counters
TOTAL=0
PASSED=0
FAILED=0
SKIPPED=0

# Test result arrays
declare -a PASSED_APIS
declare -a FAILED_APIS
declare -a SKIPPED_APIS

# Function to test endpoint
test_endpoint() {
    local method=$1
    local path=$2
    local token=$3
    local data=$4
    local description=$5

    TOTAL=$((TOTAL + 1))

    local url="${BASE_URL}${path}"
    local auth_header=""

    if [ -n "$token" ]; then
        auth_header="-H \"Authorization: Bearer $token\""
    fi

    if [ -n "$data" ]; then
        local response=$(curl -s -w "\n%{http_code}" -X $method "$url" \
            -H "Content-Type: application/json" \
            ${token:+-H "Authorization: Bearer $token"} \
            -d "$data" 2>/dev/null)
    else
        local response=$(curl -s -w "\n%{http_code}" -X $method "$url" \
            ${token:+-H "Authorization: Bearer $token"} 2>/dev/null)
    fi

    local http_code=$(echo "$response" | tail -n1)
    local body=$(echo "$response" | sed '$d')

    # Check if response is successful (2xx or expected error codes)
    if [[ $http_code =~ ^2[0-9][0-9]$ ]]; then
        echo -e "${GREEN}✓${NC} [$http_code] $method $path - $description"
        PASSED=$((PASSED + 1))
        PASSED_APIS+=("[$http_code] $method $path - $description")
    elif [[ $http_code == "401" ]] || [[ $http_code == "403" ]]; then
        echo -e "${YELLOW}⊘${NC} [$http_code] $method $path - $description (Auth required)"
        SKIPPED=$((SKIPPED + 1))
        SKIPPED_APIS+=("[$http_code] $method $path - $description")
    else
        echo -e "${RED}✗${NC} [$http_code] $method $path - $description"
        FAILED=$((FAILED + 1))
        FAILED_APIS+=("[$http_code] $method $path - $description")
    fi
}

echo "========================================="
echo "  API Endpoint Testing"
echo "========================================="
echo ""

# Get admin token
echo "Setting up authentication..."
LOGIN_RESPONSE=$(curl -s -X POST "$BASE_URL/users/login" \
    -H "Content-Type: application/json" \
    -d '{"email":"admin@test.com","password":"Admin123456"}')

ADMIN_TOKEN=$(echo $LOGIN_RESPONSE | grep -o '"access_token":"[^"]*' | cut -d'"' -f4)

if [ -n "$ADMIN_TOKEN" ]; then
    echo -e "${GREEN}✓${NC} Admin authentication successful"
else
    echo -e "${RED}✗${NC} Failed to get admin token"
fi
echo ""

echo "Testing Public Endpoints..."
echo "-------------------------------------------"

# Public endpoints
test_endpoint "GET" "/health" "" "" "Health check"
test_endpoint "GET" "/status" "" "" "Status check"
test_endpoint "GET" "/.well-known/openid-configuration" "" "" "OIDC discovery"
test_endpoint "GET" "/oauth/jwks.json" "" "" "JWKS endpoint"

echo ""
echo "Testing Authentication Endpoints..."
echo "-------------------------------------------"

# Auth endpoints
test_endpoint "POST" "/users/register" "" '{"email":"test'$(date +%s)'@test.com","password":"Test123456","first_name":"Test","last_name":"User"}' "User registration"
test_endpoint "POST" "/users/login" "" '{"email":"admin@test.com","password":"Admin123456"}' "User login"
test_endpoint "POST" "/forgot-password" "" '{"email":"admin@test.com"}' "Forgot password"
test_endpoint "POST" "/auth/phone/send-otp" "" '{"phone":"+1234567890"}' "Send phone OTP"

echo ""
echo "Testing User Endpoints (Authenticated)..."
echo "-------------------------------------------"

if [ -n "$ADMIN_TOKEN" ]; then
    test_endpoint "GET" "/users/me" "$ADMIN_TOKEN" "" "Get user profile"
    test_endpoint "GET" "/users/me/emails" "$ADMIN_TOKEN" "" "List user emails"
    test_endpoint "GET" "/users/me/phones" "$ADMIN_TOKEN" "" "List user phones"
    test_endpoint "GET" "/users/me/sessions" "$ADMIN_TOKEN" "" "List user sessions"
    test_endpoint "GET" "/users/me/business" "$ADMIN_TOKEN" "" "Get user business profile"
    test_endpoint "GET" "/auth/2fa/status" "$ADMIN_TOKEN" "" "Get 2FA status"
else
    echo -e "${YELLOW}Skipping authenticated user endpoints (no token)${NC}"
fi

echo ""
echo "Testing Admin Endpoints..."
echo "-------------------------------------------"

if [ -n "$ADMIN_TOKEN" ]; then
    test_endpoint "GET" "/admin/me" "$ADMIN_TOKEN" "" "Get admin profile"
    test_endpoint "GET" "/admin/me/sessions" "$ADMIN_TOKEN" "" "List admin sessions"
    test_endpoint "GET" "/users" "$ADMIN_TOKEN" "" "List all users"
    test_endpoint "GET" "/admin/business-types" "$ADMIN_TOKEN" "" "List business types"
    test_endpoint "GET" "/admin/audit-logs" "$ADMIN_TOKEN" "" "List audit logs"
    test_endpoint "GET" "/admin/notifications" "$ADMIN_TOKEN" "" "List admin notifications"
    test_endpoint "GET" "/clients" "$ADMIN_TOKEN" "" "List OAuth clients"
else
    echo -e "${YELLOW}Skipping admin endpoints (no token)${NC}"
fi

echo ""
echo "Testing Ticket Endpoints..."
echo "-------------------------------------------"

if [ -n "$ADMIN_TOKEN" ]; then
    test_endpoint "GET" "/tickets" "$ADMIN_TOKEN" "" "List all tickets"
    test_endpoint "GET" "/tickets/my" "$ADMIN_TOKEN" "" "List my tickets"
else
    echo -e "${YELLOW}Skipping ticket endpoints (no token)${NC}"
fi

echo ""
echo "Testing Storage Endpoints..."
echo "-------------------------------------------"

if [ -n "$ADMIN_TOKEN" ]; then
    test_endpoint "GET" "/storage/files" "$ADMIN_TOKEN" "" "List storage files"
else
    echo -e "${YELLOW}Skipping storage endpoints (no token)${NC}"
fi

echo ""
echo "Testing Notification Endpoints..."
echo "-------------------------------------------"

if [ -n "$ADMIN_TOKEN" ]; then
    test_endpoint "GET" "/notifications" "$ADMIN_TOKEN" "" "List notifications"
else
    echo -e "${YELLOW}Skipping notification endpoints (no token)${NC}"
fi

echo ""
echo "Testing Review Endpoints..."
echo "-------------------------------------------"

if [ -n "$ADMIN_TOKEN" ]; then
    test_endpoint "GET" "/reviews" "$ADMIN_TOKEN" "" "List reviews"
else
    echo -e "${YELLOW}Skipping review endpoints (no token)${NC}"
fi

echo ""
echo "Testing Settings Endpoints..."
echo "-------------------------------------------"

if [ -n "$ADMIN_TOKEN" ]; then
    test_endpoint "GET" "/settings/user" "$ADMIN_TOKEN" "" "List user settings"
    test_endpoint "GET" "/settings/system" "$ADMIN_TOKEN" "" "List system settings"
else
    echo -e "${YELLOW}Skipping settings endpoints (no token)${NC}"
fi

echo ""
echo "Testing OAuth2 Endpoints..."
echo "-------------------------------------------"

test_endpoint "GET" "/oauth/authorize" "" "" "OAuth authorize (expect error)"
test_endpoint "POST" "/oauth/token" "" '{"grant_type":"client_credentials"}' "OAuth token"

echo ""
echo "========================================="
echo "  Test Summary"
echo "========================================="
echo -e "Total Tests:    $TOTAL"
echo -e "${GREEN}Passed:         $PASSED${NC}"
echo -e "${RED}Failed:         $FAILED${NC}"
echo -e "${YELLOW}Auth Required:  $SKIPPED${NC}"
echo ""

# Calculate success rate
if [ $TOTAL -gt 0 ]; then
    SUCCESS_RATE=$(echo "scale=2; ($PASSED * 100) / $TOTAL" | bc)
    echo "Success Rate:   ${SUCCESS_RATE}%"
fi

echo ""
echo "========================================="

# Show failed endpoints if any
if [ $FAILED -gt 0 ]; then
    echo ""
    echo "Failed Endpoints:"
    echo "-------------------------------------------"
    for api in "${FAILED_APIS[@]}"; do
        echo -e "${RED}✗${NC} $api"
    done
fi

exit 0
