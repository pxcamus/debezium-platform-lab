-- 01-create-db-and-schema.sql
-- Creates the ecommerce database, data schema, and application logins/users.
SET NOCOUNT ON;

-- Create database
IF DB_ID(N'ecommerce') IS NULL
BEGIN
    PRINT 'Creating database ecommerce...';
    CREATE DATABASE ecommerce;
END
GO

ALTER DATABASE ecommerce SET RECOVERY FULL;
GO

USE ecommerce;
GO

-- Create schema
IF NOT EXISTS (SELECT 1 FROM sys.schemas WHERE name = N'data')
BEGIN
    PRINT 'Creating schema data...';
    EXEC(N'CREATE SCHEMA data');
END
GO

-- Create server logins if they don't exist
IF NOT EXISTS (SELECT 1 FROM sys.server_principals WHERE name = N'app_owner')
BEGIN
    PRINT 'Creating login app_owner...';
    CREATE LOGIN app_owner WITH PASSWORD = 'app_owner', CHECK_POLICY = OFF;
END
GO

IF NOT EXISTS (SELECT 1 FROM sys.server_principals WHERE name = N'debezium')
BEGIN
    PRINT 'Creating login debezium...';
    CREATE LOGIN debezium WITH PASSWORD = 'debezium', CHECK_POLICY = OFF;
END
GO

-- Create database users and assign roles
IF NOT EXISTS (SELECT 1 FROM sys.database_principals WHERE name = N'app_owner')
BEGIN
    PRINT 'Creating user app_owner...';
    CREATE USER app_owner FOR LOGIN app_owner;
    ALTER ROLE db_owner ADD MEMBER app_owner;
END
GO

IF NOT EXISTS (SELECT 1 FROM sys.database_principals WHERE name = N'debezium')
BEGIN
    PRINT 'Creating user debezium...';
    CREATE USER debezium FOR LOGIN debezium;
    ALTER ROLE db_datareader ADD MEMBER debezium;
    GRANT VIEW DATABASE STATE TO debezium;
END
GO
