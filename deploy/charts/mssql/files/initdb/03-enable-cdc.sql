-- 03-enable-cdc.sql
-- Enables Change Data Capture on the ecommerce database and all data schema tables.
USE ecommerce;
GO

-- Check SQL Server Agent status (REQUIRED for CDC)
PRINT 'Checking SQL Server Agent status...';
DECLARE @AgentStatus INT;
BEGIN TRY
    EXEC master.dbo.xp_servicecontrol 'QueryState', N'SQLServerAGENT', @AgentStatus OUTPUT;

    IF @AgentStatus = 1
        BEGIN
            PRINT 'SQL Server Agent is running';
        END
    ELSE
        BEGIN
            PRINT 'WARNING: SQL Server Agent is not running (Status: ' + CAST(ISNULL(@AgentStatus, -1) AS VARCHAR(10)) + ')';
            PRINT 'CDC requires SQL Server Agent to be active!';
        END
END TRY
BEGIN CATCH
    PRINT 'Unable to check SQL Server Agent status: ' + ERROR_MESSAGE();
END CATCH
GO

-- Enable CDC at database level if not already enabled
USE ecommerce;
GO

IF (SELECT is_cdc_enabled FROM sys.databases WHERE name = N'ecommerce') = 0
    BEGIN
        PRINT 'Enabling CDC at database level...';
        EXEC sys.sp_cdc_enable_db;
        PRINT 'CDC enabled at database level';
    END
ELSE
    BEGIN
        PRINT 'CDC already enabled at database level';
    END
GO

-- Verify CDC is enabled
PRINT '';
PRINT 'CDC Status:';
SELECT
    name AS DatabaseName,
    is_cdc_enabled AS IsCDCEnabled,
    CASE is_cdc_enabled
        WHEN 1 THEN 'Enabled'
        ELSE 'Disabled'
        END AS Status
FROM sys.databases
WHERE name = N'ecommerce';
GO

-- Enable CDC table-by-table for data schema (requires PK)
PRINT '';
PRINT 'Enabling CDC on individual tables...';

DECLARE @schema sysname, @table sysname;
DECLARE @enabledCount INT = 0, @skippedCount INT = 0, @errorCount INT = 0;

DECLARE c CURSOR FAST_FORWARD FOR
    SELECT s.name, t.name
    FROM sys.tables t
             JOIN sys.schemas s ON s.schema_id = t.schema_id
    WHERE s.name = N'data'
      AND EXISTS (
        SELECT 1
        FROM sys.indexes i
        WHERE i.object_id = t.object_id
          AND i.is_primary_key = 1
    );

OPEN c;
FETCH NEXT FROM c INTO @schema, @table;

WHILE @@FETCH_STATUS = 0
    BEGIN
        BEGIN TRY
            IF (SELECT is_tracked_by_cdc
                FROM sys.tables
                WHERE object_id = OBJECT_ID(QUOTENAME(@schema) + '.' + QUOTENAME(@table))) = 0
                BEGIN
                    PRINT '  Enabling CDC on ' + @schema + '.' + @table + '...';

                    EXEC sys.sp_cdc_enable_table
                         @source_schema = @schema,
                         @source_name = @table,
                         @role_name = NULL,
                         @supports_net_changes = 1;

                    SET @enabledCount = @enabledCount + 1;
                    PRINT '  Success';
                END
            ELSE
                BEGIN
                    SET @skippedCount = @skippedCount + 1;
                    PRINT '  CDC already enabled on ' + @schema + '.' + @table;
                END
        END TRY
        BEGIN CATCH
            SET @errorCount = @errorCount + 1;
            PRINT '  Failed: ' + ERROR_MESSAGE();
        END CATCH;

        FETCH NEXT FROM c INTO @schema, @table;
    END

CLOSE c;
DEALLOCATE c;
GO

-- List CDC-enabled tables
PRINT '';
PRINT 'Tables with CDC enabled:';
SELECT
    SCHEMA_NAME(schema_id) AS SchemaName,
    name AS TableName
FROM sys.tables
WHERE is_tracked_by_cdc = 1
  AND SCHEMA_NAME(schema_id) = 'data'
ORDER BY name;
GO
