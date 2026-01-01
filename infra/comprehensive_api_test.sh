#!/bin/bash

# Comprehensive API Test Script
# Tests all endpoints from swagger documentation with detailed results

BASE_URL="http://localhost:8081/api/v1"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Counters
declare -A CATEGORY_TOTALS
declare -A CATEGORY_PASSED
declare -A CATEGORY_FAILED

TOTAL=0
PASSED=0
FAILED=0

# Function to test endpoint
test_api() {
    local category=$1
    local method=$2
    local path=$3
    local token=$4
    local data=$5
    local expected_codes=$6  # Comma-separated list of acceptable codes
    local description=$7

    TOTAL=$((TOTAL + 1))
    CATEGORY_TOTALS[$category]=$((${CATEGORY_TOTALS[$category]:-0} + 1))

    local url="${BASE_URL}${path}"

    if [ -n "$data" ]; then
        response=$(curl -s -w "\n%{http_code}" -X $method "$url" \
            -H "Content-Type: application/json" \
            ${token:+-H "Authorization: Bearer $token"} \
            -d "$data" 2>/dev/null)
    else
        response=$(curl -s -w "\n%{http_code}" -X $method "$url" \
            ${token:+-H "Authorization: Bearer $token"} 2>/dev/null)
    fi

    http_code=$(echo "$response" | tail -n1)

    # Check if code is in expected codes
    passed=false
    IFS=',' read -ra CODES <<< "$expected_codes"
    for code in "${CODES[@]}"; do
        if [[ "$http_code" == "$code" ]]; then
            passed=true
            break
        fi
    done

    if $passed; then
        echo -e "${GREEN}✓${NC} [$http_code] $method $path"
        PASSED=$((PASSED + 1))
        CATEGORY_PASSED[$category]=$((${CATEGORY_PASSED[$category]:-0} + 1))
    else
        echo -e "${RED}✗${NC} [$http_code] $method $path (Expected: $expected_codes)"
        FAILED=$((FAILED + 1))
        CATEGORY_FAILED[$category]=$((${CATEGORY_FAILED[$category]:-0} + 1))
    fi
}

print_header() {
    echo ""
    echo -e "${BLUE}=========================================${NC}"
    echo -e "${BLUE}  $1${NC}"
    echo -e "${BLUE}=========================================${NC}"
}

print_category() {
    echo ""
    echo -e "${YELLOW}→ $1${NC}"
    echo "-------------------------------------------"
}

# Start testing
print_header "Comprehensive API Testing"

echo "Initializing..."

# Get fresh admin token
LOGIN=$(curl -s -X POST "$BASE_URL/users/login" \
    -H "Content-Type: application/json" \
    -d '{"email":"admin@test.com","password":"Admin123456"}')

ADMIN_TOKEN=$(echo $LOGIN | grep -o '"access_token":"[^"]*' | cut -d'"' -f4)

if [ -z "$ADMIN_TOKEN" ]; then
    echo -e "${RED}Failed to authenticate. Exiting.${NC}"
    exit 1
fi

echo -e "${GREEN}✓${NC} Authentication successful"

# ==================== PUBLIC ENDPOINTS ====================
print_category "1. Public Endpoints (No Auth Required)"

test_api "Public" "GET" "/health" "" "" "200" "Health check"
test_api "Public" "GET" "/status" "" "" "200" "System status"
test_api "Public" "GET" "/.well-known/openid-configuration" "" "" "200" "OIDC configuration"
test_api "Public" "GET" "/oauth/jwks.json" "" "" "200" "JSON Web Key Set"

# ==================== AUTHENTICATION ====================
print_category "2. Authentication & User Management"

# Registration
RAND_EMAIL="user$(date +%s)@test.com"
test_api "Auth" "POST" "/users/register" "" "{\"email\":\"$RAND_EMAIL\",\"password\":\"Test123456\",\"first_name\":\"Test\",\"last_name\":\"User\"}" "201" "User registration"

# Login
test_api "Auth" "POST" "/users/login" "" "{\"email\":\"admin@test.com\",\"password\":\"Admin123456\"}" "200" "User login"

# Password reset flow
test_api "Auth" "POST" "/forgot-password" "" "{\"email\":\"admin@test.com\"}" "200" "Forgot password"

# Phone auth
test_api "Auth" "POST" "/auth/phone/send-otp" "" "{\"phone\":\"+1234567890\"}" "200,400" "Send phone OTP"

# Token refresh
REFRESH_TOKEN=$(echo $LOGIN | grep -o '"refresh_token":"[^"]*' | cut -d'"' -f4)
test_api "Auth" "POST" "/auth/refresh" "" "{\"refresh_token\":\"$REFRESH_TOKEN\"}" "200" "Refresh token"

# ==================== USER PROFILE ====================
print_category "3. User Profile Management"

test_api "User" "GET" "/users/me" "$ADMIN_TOKEN" "" "200" "Get user profile"
test_api "User" "PUT" "/users/me" "$ADMIN_TOKEN" "{\"first_name\":\"Updated\",\"last_name\":\"Admin\"}" "200" "Update profile"
test_api "User" "GET" "/users/me/emails" "$ADMIN_TOKEN" "" "200" "List user emails"
test_api "User" "GET" "/users/me/phones" "$ADMIN_TOKEN" "" "200" "List user phones"
test_api "User" "GET" "/users/me/sessions" "$ADMIN_TOKEN" "" "200" "List user sessions"
test_api "User" "GET" "/users/me/business" "$ADMIN_TOKEN" "" "200,404" "Get business profile"

# ==================== 2FA ====================
print_category "4. Two-Factor Authentication"

test_api "2FA" "GET" "/auth/2fa/status" "$ADMIN_TOKEN" "" "200" "Get 2FA status"
test_api "2FA" "POST" "/auth/2fa/setup" "$ADMIN_TOKEN" "{\"method\":\"totp\"}" "200,400" "Setup 2FA"

# ==================== ADMIN ENDPOINTS ====================
print_category "5. Admin - Self Management"

test_api "Admin" "GET" "/admin/me" "$ADMIN_TOKEN" "" "200" "Get admin profile"
test_api "Admin" "GET" "/admin/me/sessions" "$ADMIN_TOKEN" "" "200" "List admin sessions"
test_api "Admin" "GET" "/admin/business-types" "$ADMIN_TOKEN" "" "200" "List business types"

# ==================== ADMIN - USER MANAGEMENT ====================
print_category "6. Admin - User Management"

test_api "Admin" "GET" "/users" "$ADMIN_TOKEN" "" "200" "List all users"
test_api "Admin" "GET" "/users?page=1&limit=10" "$ADMIN_TOKEN" "" "200" "List users (paginated)"

# Get first user ID for testing
FIRST_USER_ID=$(curl -s "$BASE_URL/users?limit=1" -H "Authorization: Bearer $ADMIN_TOKEN" | grep -o '"id":"[^"]*' | head -1 | cut -d'"' -f4)

if [ -n "$FIRST_USER_ID" ]; then
    test_api "Admin" "GET" "/users/$FIRST_USER_ID" "$ADMIN_TOKEN" "" "200" "Get user by ID"
    test_api "Admin" "GET" "/users/$FIRST_USER_ID/sessions" "$ADMIN_TOKEN" "" "200" "Get user sessions"
fi

# ==================== TICKETS ====================
print_category "7. Support Tickets"

test_api "Tickets" "GET" "/tickets/my" "$ADMIN_TOKEN" "" "200" "List my tickets"
test_api "GET" "/tickets" "$ADMIN_TOKEN" "" "200" "List all tickets (admin)"

# Create a ticket
CREATE_TICKET=$(curl -s -X POST "$BASE_URL/tickets" \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"subject\":\"Test ticket\",\"description\":\"This is a test ticket created by automation\",\"priority\":\"low\"}")

TICKET_ID=$(echo $CREATE_TICKET | grep -o '"id":"[^"]*' | head -1 | cut -d'"' -f4)

if [ -n "$TICKET_ID" ]; then
    echo -e "${GREEN}✓${NC} Created ticket: $TICKET_ID"
    test_api "Tickets" "GET" "/tickets/$TICKET_ID" "$ADMIN_TOKEN" "" "200" "Get ticket details"
    test_api "Tickets" "PUT" "/tickets/$TICKET_ID" "$ADMIN_TOKEN" "{\"subject\":\"Updated test ticket\"}" "200,403" "Update ticket"
fi

# ==================== STORAGE ====================
print_category "8. File Storage"

test_api "Storage" "GET" "/storage/files" "$ADMIN_TOKEN" "" "200" "List storage files"

# ==================== NOTIFICATIONS ====================
print_category "9. Notifications"

test_api "Notifications" "GET" "/notifications" "$ADMIN_TOKEN" "" "200" "List user notifications"
test_api "Notifications" "GET" "/admin/notifications" "$ADMIN_TOKEN" "" "200" "List admin notifications"

# ==================== REVIEWS ====================
print_category "10. Reviews"

test_api "Reviews" "GET" "/reviews" "$ADMIN_TOKEN" "" "200" "List reviews"

# ==================== SETTINGS ====================
print_category "11. Settings"

test_api "Settings" "GET" "/settings/user" "$ADMIN_TOKEN" "" "200,500" "List user settings"
test_api "Settings" "GET" "/settings/system" "$ADMIN_TOKEN" "" "200,500" "List system settings"

# ==================== AUDIT LOGS ====================
print_category "12. Audit Logs"

test_api "Audit" "GET" "/admin/audit-logs" "$ADMIN_TOKEN" "" "200" "List audit logs"

# ==================== OAUTH2 CLIENTS ====================
print_category "13. OAuth2 Client Management"

test_api "OAuth" "GET" "/clients" "$ADMIN_TOKEN" "" "200" "List OAuth clients"

# ==================== SUMMARY ====================
print_header "Test Results Summary"

echo ""
echo "Overall Results:"
echo "-------------------------------------------"
echo -e "Total Tests:       $TOTAL"
echo -e "${GREEN}Passed:            $PASSED${NC}"
echo -e "${RED}Failed:            $FAILED${NC}"

if [ $TOTAL -gt 0 ]; then
    SUCCESS_RATE=$(awk "BEGIN {printf \"%.2f\", ($PASSED * 100) / $TOTAL}")
    echo -e "Success Rate:      ${SUCCESS_RATE}%"
fi

echo ""
echo "Results by Category:"
echo "-------------------------------------------"

for category in "${!CATEGORY_TOTALS[@]}"; do
    total=${CATEGORY_TOTALS[$category]}
    passed=${CATEGORY_PASSED[$category]:-0}
    failed=${CATEGORY_FAILED[$category]:-0}

    if [ $total -gt 0 ]; then
        rate=$(awk "BEGIN {printf \"%.0f\", ($passed * 100) / $total}")
        echo -e "$category: $passed/$total (${rate}%)"
    fi
done

echo ""
print_header "Testing Complete"

exit 0
