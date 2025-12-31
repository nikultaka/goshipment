#!/bin/bash

# Database Migration Script
# Usage: ./scripts/migrate.sh [up|down|reset]

set -e

# Load environment variables
if [ -f .env ]; then
    export $(cat .env | grep -v '^#' | xargs)
fi

# Default values
DB_HOST=${DB_HOST:-localhost}
DB_PORT=${DB_PORT:-5432}
DB_USER=${DB_USER:-postgres}
DB_NAME=${DB_NAME:-goapi}

MIGRATIONS_DIR="infra/database/migrations"

echo "🔄 Database Migration Tool"
echo "=========================="
echo "Host: $DB_HOST"
echo "Database: $DB_NAME"
echo ""

# Function to run migrations up
migrate_up() {
    echo "▶ Running migrations..."
    for file in "$MIGRATIONS_DIR"/*.sql; do
        if [ -f "$file" ]; then
            echo "  Applying: $(basename "$file")"
            PGPASSWORD=$DB_PASSWORD psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f "$file"
        fi
    done
    echo "✓ All migrations applied successfully!"
}

# Function to drop all tables (reset)
migrate_reset() {
    echo "⚠️  WARNING: This will drop all tables!"
    read -p "Are you sure? (yes/no): " confirm
    if [ "$confirm" = "yes" ]; then
        echo "▶ Dropping all tables..."
        PGPASSWORD=$DB_PASSWORD psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" << EOF
DROP TABLE IF EXISTS team_members CASCADE;
DROP TABLE IF EXISTS support_ticket_replies CASCADE;
DROP TABLE IF EXISTS support_tickets CASCADE;
DROP TABLE IF EXISTS reviews CASCADE;
DROP TABLE IF EXISTS files CASCADE;
DROP TABLE IF EXISTS settings CASCADE;
DROP TABLE IF EXISTS notifications CASCADE;
DROP TABLE IF EXISTS audit_logs CASCADE;
DROP TABLE IF EXISTS oauth_authorization_codes CASCADE;
DROP TABLE IF EXISTS oauth_tokens CASCADE;
DROP TABLE IF EXISTS oauth_clients CASCADE;
DROP TABLE IF EXISTS user_profiles CASCADE;
DROP TABLE IF EXISTS users CASCADE;
EOF
        echo "✓ All tables dropped!"
        echo ""
        migrate_up
    else
        echo "❌ Migration reset cancelled"
    fi
}

# Main logic
case "$1" in
    up)
        migrate_up
        ;;
    reset)
        migrate_reset
        ;;
    *)
        echo "Usage: $0 [up|reset]"
        echo ""
        echo "Commands:"
        echo "  up     - Run all migrations"
        echo "  reset  - Drop all tables and rerun migrations"
        exit 1
        ;;
esac
