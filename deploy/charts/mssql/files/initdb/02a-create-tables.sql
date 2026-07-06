-- 02a-create-tables.sql
-- Creates the ecommerce tables in the data schema.
USE ecommerce;
GO

SET ANSI_NULLS ON;
SET ANSI_PADDING ON;
SET ANSI_WARNINGS ON;
SET ARITHABORT ON;
SET CONCAT_NULL_YIELDS_NULL ON;
SET QUOTED_IDENTIFIER ON;
SET NUMERIC_ROUNDABORT OFF;
GO

-- categories
IF OBJECT_ID(N'[data].[categories]', N'U') IS NULL
BEGIN
    PRINT 'Creating table data.categories...';
    CREATE TABLE data.categories (
        id          NVARCHAR(255) NOT NULL
            CONSTRAINT PK_categories PRIMARY KEY,
        name        NVARCHAR(255) NOT NULL,
        description NVARCHAR(MAX),
        created_at  DATETIME2     NOT NULL,
        updated_at  DATETIME2     NOT NULL
    );
END
GO

-- products
IF OBJECT_ID(N'[data].[products]', N'U') IS NULL
BEGIN
    PRINT 'Creating table data.products...';
    CREATE TABLE data.products (
        id             NVARCHAR(255)   NOT NULL
            CONSTRAINT PK_products PRIMARY KEY,
        name           NVARCHAR(255)   NOT NULL,
        category_id    NVARCHAR(255)
            CONSTRAINT FK_products_categories REFERENCES data.categories(id),
        price          NUMERIC(12,2)   NOT NULL,
        stock_quantity INT             NOT NULL,
        tags           NVARCHAR(MAX),  -- JSON array of tags
        created_at     DATETIME2       NOT NULL,
        updated_at     DATETIME2       NOT NULL
    );
END
GO

-- users
IF OBJECT_ID(N'[data].[users]', N'U') IS NULL
BEGIN
    PRINT 'Creating table data.users...';
    CREATE TABLE data.users (
        id       NVARCHAR(255) NOT NULL
            CONSTRAINT PK_users PRIMARY KEY,
        username NVARCHAR(255) NOT NULL,
        email    NVARCHAR(255) NOT NULL
    );
END
GO

-- orders
IF OBJECT_ID(N'[data].[orders]', N'U') IS NULL
BEGIN
    PRINT 'Creating table data.orders...';
    CREATE TABLE data.orders (
        id           NVARCHAR(255) NOT NULL
            CONSTRAINT PK_orders PRIMARY KEY,
        user_id      NVARCHAR(255) NOT NULL
            CONSTRAINT FK_orders_users REFERENCES data.users(id),
        status       NVARCHAR(50)  NOT NULL,
        total_amount NUMERIC(12,2) NOT NULL,
        created_at   DATETIME2     NOT NULL,
        updated_at   DATETIME2     NOT NULL
    );
END
GO

-- order_items
IF OBJECT_ID(N'[data].[order_items]', N'U') IS NULL
BEGIN
    PRINT 'Creating table data.order_items...';
    CREATE TABLE data.order_items (
        order_id   NVARCHAR(255) NOT NULL
            CONSTRAINT FK_order_items_orders REFERENCES data.orders(id) ON DELETE CASCADE,
        product_id NVARCHAR(255) NOT NULL,
        name       NVARCHAR(255) NOT NULL,
        quantity   INT           NOT NULL,
        unit_price NUMERIC(12,2) NOT NULL,
        CONSTRAINT PK_order_items PRIMARY KEY (order_id, product_id)
    );
END
GO
