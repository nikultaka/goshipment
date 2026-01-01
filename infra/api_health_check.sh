#!/bin/bash

# API Health Check Script
# Tests all major API endpoints and reports success/failure

BASE_URL="http://localhost:8081/api/v1"
TOKEN=$(cat /tmp/fresh_token.txt | tr -d '\n')

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

TOTAL=0
SUCCESS=0
FAILED=0

echo "=================================================="
echo "API Health Check - $(date)"
echo "=================================================="
echo ""

# Test function
test_endpoint() {
    local category=$1
    local method=$2
    local endpoint=$3
    local token_required=$4
    local data=$5
    local expected_status=$6

    TOTAL=$((TOTAL + 1))

    if [ "$token_required" = "true" ]; then
        if [ -z "$data" ]; then
            response=$(curl -s -w "\n%{http_code}" -X $method "$BASE_URL$endpoint" -H "Authorization: Bearer $TOKEN")
        else
            response=$(curl -s -w "\n%{http_code}" -X $method "$BASE_URL$endpoint" -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d "$data")
        fi
    else
        if [ -z "$data" ]; then
            response=$(curl -s -w "\n%{http_code}" -X $method "$BASE_URL$endpoint")
        else
            response=$(curl -s -w "\n%{http_code}" -X $method "$BASE_URL$endpoint" -H "Content-Type: application/json" -d "$data")
        fi
    fi

    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')

    # Check if status code matches expected (can be comma-separated list)
    if [[ ",$expected_status," == *",$http_code,"* ]]; then
        echo -e "${GREEN}✓${NC} [$category] $method $endpoint - HTTP $http_code"
        SUCCESS=$((SUCCESS + 1))
    else
        echo -e "${RED}✗${NC} [$category] $method $endpoint - HTTP $http_code (expected $expected_status)"
        FAILED=$((FAILED + 1))
    fi
}

echo "=== PUBLIC ENDPOINTS ==="
test_endpoint "Public" "GET" "/health" "false" "" "200"
test_endpoint "Public" "GET" "/status" "false" "" "200"
test_endpoint "Public" "GET" "/.well-known/openid-configuration" "false" "" "200"
test_endpoint "Public" "GET" "/oauth/jwks.json" "false" "" "200"
echo ""

echo "=== AUTHENTICATION ==="
test_endpoint "Auth" "POST" "/users/register" "false" '{"email":"test'$(date +%s)'@test.com","password":"Test123!@#","first_name":"Test","last_name":"User"}' "201,400"
test_endpoint "Auth" "POST" "/users/login" "false" '{"email":"admin@test.com","password":"Admin@123"}' "200,400"
test_endpoint "Auth" "POST" "/forgot-password" "false" '{"email":"admin@test.com"}' "200"
test_endpoint "Auth" "POST" "/auth/phone/send-otp" "false" '{"phone":"+1234567890"}' "200,400"
test_endpoint "Auth" "POST" "/auth/refresh" "false" '{"refresh_token":"dummy"}' "200,400,401"
echo ""

echo "=== USER PROFILE ==="
test_endpoint "User" "GET" "/users/me" "true" "" "200"
test_endpoint "User" "PUT" "/users/me" "true" '{"first_name":"Updated"}' "200,400"
test_endpoint "User" "GET" "/users/me/emails" "true" "" "200"
test_endpoint "User" "GET" "/users/me/phones" "true" "" "200"
test_endpoint "User" "GET" "/users/me/sessions" "true" "" "200"
test_endpoint "User" "GET" "/users/me/business" "true" "" "200,404"
echo ""

echo "=== ADMIN ENDPOINTS ==="
test_endpoint "Admin" "GET" "/admin/me" "true" "" "200"
test_endpoint "Admin" "GET" "/admin/me/sessions" "true" "" "200"
test_endpoint "Admin" "GET" "/admin/business-types" "true" "" "200"
test_endpoint "Admin" "GET" "/users" "true" "" "200"
test_endpoint "Admin" "GET" "/users?page=1&limit=10" "true" "" "200"
echo ""

echo "=== SUPPORT TICKETS ==="
test_endpoint "Tickets" "GET" "/tickets/my" "true" "" "200"
test_endpoint "Tickets" "POST" "/tickets" "true" '{"subject":"Test ticket","description":"Test description","priority":"medium"}' "201,400"
test_endpoint "Tickets" "GET" "/tickets" "true" "" "200"
echo ""

echo "=== SETTINGS ==="
test_endpoint "Settings" "GET" "/settings/user" "true" "" "200"
test_endpoint "Settings" "GET" "/settings/system" "true" "" "200"
echo ""

echo "=== FILE STORAGE ==="
test_endpoint "Storage" "GET" "/storage/files" "true" "" "200"
echo ""

echo "=== NOTIFICATIONS ==="
test_endpoint "Notifications" "GET" "/notifications" "true" "" "200"
test_endpoint "Notifications" "GET" "/admin/notifications" "true" "" "200"
echo ""

echo "=== REVIEWS ==="
test_endpoint "Reviews" "GET" "/reviews" "true" "" "200"
echo ""

echo "=== AUDIT LOGS ==="
test_endpoint "Audit" "GET" "/admin/audit-logs" "true" "" "200"
echo ""

echo "=== OAUTH CLIENTS ==="
test_endpoint "OAuth" "GET" "/clients" "true" "" "200"
echo ""

echo "=================================================="
echo "SUMMARY"
echo "=================================================="
echo -e "Total endpoints tested: $TOTAL"
echo -e "${GREEN}Successful: $SUCCESS ($(awk "BEGIN {printf \"%.2f\", ($SUCCESS/$TOTAL)*100}")%)${NC}"
echo -e "${RED}Failed: $FAILED ($(awk "BEGIN {printf \"%.2f\", ($FAILED/$TOTAL)*100}")%)${NC}"
echo "=================================================="
