-- Local Development Database Initialization Script
-- WARNING: This script is ONLY for local Docker development environment
-- DO NOT use this in production or cloud environments

-- Create development database only
CREATE DATABASE IF NOT EXISTS orris_dev CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- Create test database for integration tests
CREATE DATABASE IF NOT EXISTS orris_test CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- Create local development user with limited privileges
CREATE USER IF NOT EXISTS 'orris'@'%' IDENTIFIED BY 'orris_password';

-- Grant only necessary privileges for application operation
GRANT SELECT, INSERT, UPDATE, DELETE, CREATE, DROP, INDEX, ALTER 
ON orris_dev.* TO 'orris'@'%';

GRANT SELECT, INSERT, UPDATE, DELETE, CREATE, DROP, INDEX, ALTER 
ON orris_test.* TO 'orris'@'%';

FLUSH PRIVILEGES;

-- Note: Production database and users should be managed through cloud console
-- This script is mounted in docker-compose.yml for local development only