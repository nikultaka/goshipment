#!/bin/bash

# Script to create an admin user
# Usage: ./scripts/create-admin.sh

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Load environment variables
if [ -f .env ]; then
    export $(grep -v '^#' .env | xargs)
else
    echo -e "${RED}Error: .env file not found${NC}"
    exit 1
fi

# API endpoint
API_URL="http://localhost:${APP_PORT:-8081}/api/v1"

echo -e "${YELLOW}=== Admin User Creation Script ===${NC}\n"

# Prompt for user details
read -p "Email: " EMAIL
read -sp "Password (min 8 chars): " PASSWORD
echo
read -p "First Name: " FIRST_NAME
read -p "Last Name: " LAST_NAME

echo -e "\n${YELLOW}Creating user account...${NC}"

# Register user via API
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
    echo -e "${YELLOW}Try running manually:${NC}"
    echo "PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c \"UPDATE users SET role = 'admin' WHERE email = '$EMAIL';\""
    exit 1
fi

# Verify admin user
echo -e "\n${YELLOW}Verifying admin user...${NC}"

RESULT=$(PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -t -c \
    "SELECT email, role, status FROM users WHERE email = '$EMAIL';")

echo -e "${GREEN}Admin user details:${NC}"
echo "$RESULT"

echo -e "\n${GREEN}✓ Admin user created successfully!${NC}"
echo -e "${YELLOW}You can now login with:${NC}"
echo "  Email: $EMAIL"
echo "  Password: [your password]"
echo -e "\n${YELLOW}Login endpoint:${NC} POST $API_URL/users/login"
