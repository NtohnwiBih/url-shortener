set -e

echo "Initializing URL Shortener Database..."

# Wait for PostgreSQL to be ready
echo "Waiting for PostgreSQL to be ready..."
while ! pg_isready -h localhost -p 5432 -U urlshortener; do
  sleep 1
done

# Run migrations
echo "Running database migrations..."
for migration in /docker-entrypoint-initdb.d/*.sql; do
    echo "Applying migration: $(basename $migration)"
    psql -h localhost -U urlshortener -d urlshortener -f "$migration"
done

# Create additional indexes if needed
echo "Creating performance indexes..."
psql -h localhost -U urlshortener -d urlshortener <<-EOSQL
    -- Additional composite index for common queries
    CREATE INDEX IF NOT EXISTS idx_urls_active_created 
    ON urls(is_active, created_at) 
    WHERE is_active = true;
    
    -- Index for cleanup job
    CREATE INDEX IF NOT EXISTS idx_urls_expires_active 
    ON urls(expires_at) 
    WHERE is_active = true AND expires_at IS NOT NULL;
    
    -- Update statistics
    ANALYZE;
EOSQL

echo "Database initialization completed successfully!"