-- 02b-constraints-and-indexes.sql
-- Secondary indexes for the ecommerce data tables.
USE ecommerce;
GO

-- Unique index on users.username
IF OBJECT_ID(N'[data].[users]', N'U') IS NOT NULL
AND NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'UQ_users_username' AND object_id = OBJECT_ID(N'[data].[users]'))
BEGIN
    PRINT 'Creating unique index UQ_users_username on data.users';
    CREATE UNIQUE INDEX UQ_users_username ON data.users(username);
END
GO

-- Unique index on users.email
IF OBJECT_ID(N'[data].[users]', N'U') IS NOT NULL
AND NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'UQ_users_email' AND object_id = OBJECT_ID(N'[data].[users]'))
BEGIN
    PRINT 'Creating unique index UQ_users_email on data.users';
    CREATE UNIQUE INDEX UQ_users_email ON data.users(email);
END
GO

-- Index on products.category_id for FK lookups
IF OBJECT_ID(N'[data].[products]', N'U') IS NOT NULL
AND NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_products_category_id' AND object_id = OBJECT_ID(N'[data].[products]'))
BEGIN
    PRINT 'Creating index IX_products_category_id on data.products';
    CREATE INDEX IX_products_category_id ON data.products(category_id);
END
GO

-- Index on orders.user_id for FK lookups
IF OBJECT_ID(N'[data].[orders]', N'U') IS NOT NULL
AND NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_orders_user_id' AND object_id = OBJECT_ID(N'[data].[orders]'))
BEGIN
    PRINT 'Creating index IX_orders_user_id on data.orders';
    CREATE INDEX IX_orders_user_id ON data.orders(user_id);
END
GO

-- Index on orders.status for filtering
IF OBJECT_ID(N'[data].[orders]', N'U') IS NOT NULL
AND NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_orders_status' AND object_id = OBJECT_ID(N'[data].[orders]'))
BEGIN
    PRINT 'Creating index IX_orders_status on data.orders';
    CREATE INDEX IX_orders_status ON data.orders(status);
END
GO

-- Index on orders.created_at for range queries
IF OBJECT_ID(N'[data].[orders]', N'U') IS NOT NULL
AND NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_orders_created_at' AND object_id = OBJECT_ID(N'[data].[orders]'))
BEGIN
    PRINT 'Creating index IX_orders_created_at on data.orders';
    CREATE INDEX IX_orders_created_at ON data.orders(created_at);
END
GO
