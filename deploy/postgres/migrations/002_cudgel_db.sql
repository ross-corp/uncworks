-- 002_cudgel_db.sql
-- Sets up the cudgel user and grants for the cudgel database.
-- Pre-requisite: run as a postgres superuser AFTER creating the cudgel database:
--   psql -U postgres -c "CREATE DATABASE cudgel;"
--
-- Then apply this migration against the default postgres database:
--   psql -U postgres -f 002_cudgel_db.sql
--
-- Finally, enable pgvector in the cudgel database:
--   psql -U postgres -d cudgel -c "CREATE EXTENSION IF NOT EXISTS vector;"

-- Create the cudgel role if it does not already exist.
DO $$
BEGIN
  IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'cudgel') THEN
    CREATE ROLE cudgel WITH LOGIN;
  END IF;
END
$$;

-- Grant connect and create privileges on the cudgel database.
GRANT CONNECT ON DATABASE cudgel TO cudgel;
GRANT CREATE ON DATABASE cudgel TO cudgel;
