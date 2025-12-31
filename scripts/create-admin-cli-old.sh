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

# API endpoint
API_URL="http://localhost:${APP_PORT:-8081}/api/v1"

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

# Update user role to admin in database
echo -e "${YELLOW}Updating user role to admin...${NC}"

PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c \
    "UPDATE users SET role = 'admin' WHERE email = '$EMAIL';" > /dev/null 2>&1

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ User role updated to admin${NC}"
else
    echo -e "${RED}Error: Failed to update user role${NC}"
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
