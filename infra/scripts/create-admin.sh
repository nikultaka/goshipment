#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Default values
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-zyrobyte}"
DB_USER="${DB_USER:-postgres}"
DB_PASSWORD="${DB_PASSWORD:-postgres}"

# Function to print colored messages
print_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to generate bcrypt hash
generate_bcrypt_hash() {
    local password="$1"

    # Try using htpasswd (Apache utils) if available
    if command -v htpasswd &> /dev/null; then
        htpasswd -bnBC 10 "" "$password" | tr -d ':\n' | sed 's/^\$2y/\$2a/'
        return 0
    fi

    # Try using Python if available
    if command -v python3 &> /dev/null; then
        python3 -c "import bcrypt; print(bcrypt.hashpw(b'$password', bcrypt.gensalt()).decode())" 2>/dev/null
        return 0
    fi

    # Fallback: Use Docker with Go
    docker run --rm golang:1.24 sh -c 'cat > /tmp/hash.go <<'"'"'EOF'"'"'
package main
import (
    "fmt"
    "golang.org/x/crypto/bcrypt"
)
func main() {
    hash, _ := bcrypt.GenerateFromPassword([]byte("'"$password"'"), bcrypt.DefaultCost)
    fmt.Print(string(hash))
}
EOF
go run /tmp/hash.go' 2>/dev/null
}

# Function to generate UUID
generate_uuid() {
    if command -v uuidgen &> /dev/null; then
        uuidgen | tr '[:upper:]' '[:lower:]'
    else
        # Fallback: use PostgreSQL to generate UUID
        docker exec go_skeleton_db psql -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT gen_random_uuid();" | tr -d ' \n'
    fi
}

# Function to execute SQL
execute_sql() {
    local sql="$1"
    docker exec -i go_skeleton_db psql -U "$DB_USER" -d "$DB_NAME" -c "$sql"
}

# Parse command line arguments
show_usage() {
    cat <<USAGE
Usage: $0 [OPTIONS]

Create an admin user in the database.

OPTIONS:
    -e, --email EMAIL          Admin email address (required)
    -p, --password PASSWORD    Admin password (required, min 8 characters)
    -f, --first-name NAME      First name (default: Admin)
    -l, --last-name NAME       Last name (default: User)
    -u, --username USERNAME    Username (optional)
    -h, --help                 Show this help message

EXAMPLES:
    # Create admin user with email and password
    $0 -e admin@example.com -p SecurePass123

    # Create admin user with custom names
    $0 -e admin@myapp.com -p MyPassword123 -f John -l Doe

    # Using environment variables for DB connection
    DB_HOST=db DB_USER=postgres $0 -e admin@example.com -p pass123

USAGE
}

# Initialize variables
ADMIN_EMAIL=""
ADMIN_PASSWORD=""
ADMIN_FIRST_NAME="Admin"
ADMIN_LAST_NAME="User"
ADMIN_USERNAME=""

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -e|--email)
            ADMIN_EMAIL="$2"
            shift 2
            ;;
        -p|--password)
            ADMIN_PASSWORD="$2"
            shift 2
            ;;
        -f|--first-name)
            ADMIN_FIRST_NAME="$2"
            shift 2
            ;;
        -l|--last-name)
            ADMIN_LAST_NAME="$2"
            shift 2
            ;;
        -u|--username)
            ADMIN_USERNAME="$2"
            shift 2
            ;;
        -h|--help)
            show_usage
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            show_usage
            exit 1
            ;;
    esac
done

# Validate required parameters
if [ -z "$ADMIN_EMAIL" ]; then
    print_error "Email is required!"
    show_usage
    exit 1
fi

if [ -z "$ADMIN_PASSWORD" ]; then
    print_error "Password is required!"
    show_usage
    exit 1
fi

if [ ${#ADMIN_PASSWORD} -lt 8 ]; then
    print_error "Password must be at least 8 characters long!"
    exit 1
fi

# Validate email format
if ! echo "$ADMIN_EMAIL" | grep -qE '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}$'; then
    print_error "Invalid email format!"
    exit 1
fi

print_info "Creating admin user..."
print_info "Email: $ADMIN_EMAIL"
print_info "Name: $ADMIN_FIRST_NAME $ADMIN_LAST_NAME"

# Check if Docker container is running
if ! docker ps | grep -q go_skeleton_db; then
    print_error "Database container 'go_skeleton_db' is not running!"
    print_error "Please start the Docker containers first: docker-compose up -d"
    exit 1
fi

# Generate password hash
print_info "Generating password hash..."
PASSWORD_HASH=$(generate_bcrypt_hash "$ADMIN_PASSWORD")

if [ -z "$PASSWORD_HASH" ]; then
    print_error "Failed to generate password hash!"
    exit 1
fi

# Generate UUIDs
USER_ID=$(generate_uuid)
EMAIL_ID=$(generate_uuid)
PHONE_ID=$(generate_uuid)
USER_ROLE_ID=$(generate_uuid)
EXTERNAL_ID="usr_$(date +%s)"

if [ -z "$ADMIN_USERNAME" ]; then
    ADMIN_USERNAME="admin_$(date +%s)"
fi

print_info "Generated User ID: $USER_ID"

# Check if user with this email already exists
EXISTING_EMAIL=$(docker exec go_skeleton_db psql -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT email FROM user_emails WHERE email = '$ADMIN_EMAIL';" | tr -d ' \n')

if [ ! -z "$EXISTING_EMAIL" ]; then
    print_warn "User with email $ADMIN_EMAIL already exists!"
    read -p "Do you want to update the password for this user? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        print_info "Operation cancelled."
        exit 0
    fi

    # Update existing user's password
    print_info "Updating password for existing user..."
    execute_sql "UPDATE users SET password_hash = '$PASSWORD_HASH', updated_at = NOW() WHERE primary_email_id IN (SELECT id FROM user_emails WHERE email = '$ADMIN_EMAIL');"

    print_info "✅ Password updated successfully for $ADMIN_EMAIL"
    exit 0
fi

# Get admin role ID
ADMIN_ROLE_ID=$(docker exec go_skeleton_db psql -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT id FROM roles WHERE name = 'admin' LIMIT 1;" | tr -d ' \n')

if [ -z "$ADMIN_ROLE_ID" ]; then
    print_error "Admin role not found in database! Please run migrations first."
    exit 1
fi

print_info "Admin Role ID: $ADMIN_ROLE_ID"

# Begin transaction
print_info "Inserting user into database..."

# Create user
execute_sql "
BEGIN;

-- Insert user
INSERT INTO users (
    id,
    external_id,
    username,
    password_hash,
    first_name,
    last_name,
    display_name,
    timezone,
    locale,
    date_format,
    time_format,
    is_active,
    is_verified,
    verification_level,
    token_version,
    has_2fa_enabled,
    created_at,
    updated_at
) VALUES (
    '$USER_ID',
    '$EXTERNAL_ID',
    '$ADMIN_USERNAME',
    '$PASSWORD_HASH',
    '$ADMIN_FIRST_NAME',
    '$ADMIN_LAST_NAME',
    '$ADMIN_FIRST_NAME $ADMIN_LAST_NAME',
    'UTC',
    'en',
    'YYYY-MM-DD',
    '24h',
    TRUE,
    TRUE,
    3,
    1,
    FALSE,
    NOW(),
    NOW()
);

-- Insert email
INSERT INTO user_emails (
    id,
    user_id,
    email,
    is_primary,
    is_recovery,
    is_verified,
    verified_at,
    is_active,
    created_at,
    updated_at
) VALUES (
    '$EMAIL_ID',
    '$USER_ID',
    '$ADMIN_EMAIL',
    TRUE,
    TRUE,
    TRUE,
    NOW(),
    TRUE,
    NOW(),
    NOW()
);

-- Update user with primary email
UPDATE users SET primary_email_id = '$EMAIL_ID' WHERE id = '$USER_ID';

-- Assign admin role
INSERT INTO user_roles (
    id,
    user_id,
    role_id,
    granted_at,
    is_active,
    created_at,
    updated_at
) VALUES (
    '$USER_ROLE_ID',
    '$USER_ID',
    '$ADMIN_ROLE_ID',
    NOW(),
    TRUE,
    NOW(),
    NOW()
);

COMMIT;
"

if [ $? -eq 0 ]; then
    print_info "✅ Admin user created successfully!"
    echo ""
    echo "═══════════════════════════════════════"
    echo "  Admin User Credentials"
    echo "═══════════════════════════════════════"
    echo "  Email:    $ADMIN_EMAIL"
    echo "  Password: $ADMIN_PASSWORD"
    echo "  Name:     $ADMIN_FIRST_NAME $ADMIN_LAST_NAME"
    echo "  Role:     admin"
    echo "═══════════════════════════════════════"
    echo ""
    print_warn "⚠️  Please save these credentials securely and change the password after first login!"
    echo ""
    print_info "You can now login at: http://localhost:8081/api/v1/users/login"
else
    print_error "Failed to create admin user!"
    exit 1
fi
