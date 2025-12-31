#!/bin/bash
# Script to create an admin user (non-interactive version)
# Usage: ./scripts/create-admin-cli.sh EMAIL PASSWORD FIRST_NAME LAST_NAME
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check arguments
if [ "$#" -ne 4 ]; then
    echo -e "${RED}Usage: $0 EMAIL PASSWORD FIRST_NAME LAST_NAME${NC}"
    echo "Example: $0 admin@example.com SecurePass123 John Doe"
    exit 1
fi

EMAIL=$1
PASSWORD=$2
FIRST_NAME=$3
LAST_NAME=$4

# Load environment variables
if [ -f .env ]; then
    export $(grep -v '^#' .env | xargs)
else
    echo -e "${RED}Error: .env file not found${NC}"
    exit 1
fi

# API endpoint - use localhost for host access
API_URL="http://localhost:${APP_PORT:-8081}/api/v1"

# Database - use localhost instead of docker hostname
DB_HOST_LOCAL="localhost"

echo -e "${YELLOW}=== Creating Admin User ===${NC}\n"
echo "Email: $EMAIL"
echo "Name: $FIRST_NAME $LAST_NAME"
echo

# Register user via API
echo -e "${YELLOW}Creating user account...${NC}"
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$API_URL/users/register" \
    -H "Content-Type: application/json" \
    -d "{
        \"email\": \"$EMAIL\",
        \"password\": \"$PASSWORD\",
        \"first_name\": \"$FIRST_NAME\",
        \"last_name\": \"$LAST_NAME\"
    }")

# Extract HTTP status code
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | sed '$d')

if [ "$HTTP_CODE" -ne 201 ]; then
    echo -e "${RED}Error: Failed to create user (HTTP $HTTP_CODE)${NC}"
    echo "$BODY"
    exit 1
fi

echo -e "${GREEN}✓ User created successfully${NC}"

# Get user ID from the created user
echo -e "${YELLOW}Assigning admin role...${NC}"

# Use docker exec to run psql inside the container
USER_ID=$(docker exec go_skeleton_db psql -U $DB_USER -d $DB_NAME -t -c \
    "SELECT u.id FROM users u 
     JOIN user_emails ue ON u.primary_email_id = ue.id 
     WHERE ue.email_lower = lower('$EMAIL');" | xargs)

if [ -z "$USER_ID" ]; then
    echo -e "${RED}Error: Could not find user ID${NC}"
    exit 1
fi

echo "User ID: $USER_ID"

# Get admin role ID
ADMIN_ROLE_ID=$(docker exec go_skeleton_db psql -U $DB_USER -d $DB_NAME -t -c \
    "SELECT id FROM roles WHERE name = 'admin';" | xargs)

if [ -z "$ADMIN_ROLE_ID" ]; then
    echo -e "${RED}Error: Admin role not found in database${NC}"
    exit 1
fi

# Assign admin role to user
docker exec go_skeleton_db psql -U $DB_USER -d $DB_NAME -c \
    "INSERT INTO user_roles (user_id, role_id, is_active) 
     VALUES ('$USER_ID', '$ADMIN_ROLE_ID', true)
     ON CONFLICT (user_id, role_id) DO UPDATE SET is_active = true;" > /dev/null 2>&1

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Admin role assigned${NC}"
else
    echo -e "${RED}Error: Failed to assign admin role${NC}"
    exit 1
fi

# Verify admin user
echo -e "\n${GREEN}✓ Admin user created successfully!${NC}"
echo -e "${YELLOW}Login credentials:${NC}"
echo "  Email: $EMAIL"
echo "  Role: admin"
echo -e "\n${YELLOW}Test login:${NC}"
echo "curl -X POST $API_URL/users/login \\"
echo "  -H 'Content-Type: application/json' \\"
echo "  -d '{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}'"
