#!/bin/bash
set -e

echo "Waiting for database..."
timeout=60
elapsed=0
while ! pg_isready -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" >/dev/null 2>&1; do
  sleep 2
  elapsed=$((elapsed + 2))
  if [ $elapsed -ge $timeout ]; then
    echo "Database not ready after $timeout seconds, exiting."
    exit 1
  fi
  echo "Still waiting for database..."
done
echo "Database ready!"

# Create migrations tracking table if it doesn't exist
PGPASSWORD=$DB_PASSWORD psql -h "$DB_HOST" -U "$DB_USER" -d "$DB_NAME" << EOF
CREATE TABLE IF NOT EXISTS schema_migrations (
  version VARCHAR(255) PRIMARY KEY,
  applied_at TIMESTAMP DEFAULT NOW()
);
EOF

# Run migrations from /sql/migrations directory
if [ -d /sql/migrations ]; then
  echo "Running database migrations..."
  for migration_file in /sql/migrations/*.sql; do
    if [ -f "$migration_file" ]; then
      migration_name=$(basename "$migration_file")

      # Check if migration has already been applied
      already_applied=$(PGPASSWORD=$DB_PASSWORD psql -h "$DB_HOST" -U "$DB_USER" -d "$DB_NAME" -tAc "SELECT COUNT(*) FROM schema_migrations WHERE version='$migration_name';" || echo "0")

      if [ "$already_applied" = "0" ]; then
        echo "Applying migration: $migration_name"
        if PGPASSWORD=$DB_PASSWORD psql -h "$DB_HOST" -U "$DB_USER" -d "$DB_NAME" -f "$migration_file"; then
          # Mark migration as applied
          PGPASSWORD=$DB_PASSWORD psql -h "$DB_HOST" -U "$DB_USER" -d "$DB_NAME" -c "INSERT INTO schema_migrations (version) VALUES ('$migration_name');"
          echo "✓ Successfully applied: $migration_name"
        else
          echo "✗ Failed to apply: $migration_name"
          echo "Migration failed. Please check the error above."
        fi
      else
        echo "⊘ Skipping (already applied): $migration_name"
      fi
    fi
  done
  echo "All migrations processed."
else
  echo "No migrations directory found at /sql/migrations"
fi

# Start the correct binary
if [ "$1" = "worker" ]; then
  echo "Starting Go Worker..."
  exec /app/worker
else
  echo "Starting Go API..."
  exec /app/api
fi
