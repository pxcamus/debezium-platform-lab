-- Schema for CDC demo: E-Commerce
-- Tables live in the default "public" schema of the "ecommerce" database.

-- 1. Categories table (static, rarely updated)
CREATE TABLE IF NOT EXISTS public.categories
(
    category_id SERIAL PRIMARY KEY,
    name        TEXT NOT NULL,
    description TEXT
);

-- 2. Products table (frequently updated for pricing/stock)
CREATE TABLE IF NOT EXISTS public.products
(
    product_id     SERIAL PRIMARY KEY,
    name           TEXT           NOT NULL,
    price          NUMERIC(10, 2) NOT NULL,
    stock_quantity INT            NOT NULL DEFAULT 0,
    category_id    INT            NOT NULL,
    updated_at     TIMESTAMP               DEFAULT NOW(),
    FOREIGN KEY (category_id) REFERENCES public.categories (category_id)
);

-- 3. Users table (high-volume inserts, occasional updates)
CREATE TABLE IF NOT EXISTS public.users
(
    user_id    SERIAL PRIMARY KEY,
    username   TEXT NOT NULL UNIQUE,
    email      TEXT NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT NOW(),
    last_login TIMESTAMP
);

-- 4. Orders table (high-volume inserts, frequent updates for status)
CREATE TABLE IF NOT EXISTS public.orders
(
    order_id     SERIAL PRIMARY KEY,
    user_id      INT            NOT NULL,
    order_date   TIMESTAMP DEFAULT NOW(),
    status       TEXT           NOT NULL, -- e.g., 'pending', 'shipped', 'cancelled'
    total_amount NUMERIC(10, 2) NOT NULL,
    FOREIGN KEY (user_id) REFERENCES public.users (user_id)
);

-- 5. Order Items table (very high-volume inserts/updates)
CREATE TABLE IF NOT EXISTS public.order_items
(
    order_item_id SERIAL PRIMARY KEY,
    order_id      INT            NOT NULL,
    product_id    INT            NOT NULL,
    quantity      INT            NOT NULL,
    unit_price    NUMERIC(10, 2) NOT NULL,
    FOREIGN KEY (order_id) REFERENCES public.orders (order_id),
    FOREIGN KEY (product_id) REFERENCES public.products (product_id)
);

-- 6. Payments table (high-volume inserts, updates for status)
CREATE TABLE IF NOT EXISTS public.payments
(
    payment_id       SERIAL PRIMARY KEY,
    order_id         INT            NOT NULL,
    amount           NUMERIC(10, 2) NOT NULL,
    payment_method   TEXT           NOT NULL, -- e.g., 'credit_card', 'paypal'
    status           TEXT           NOT NULL, -- e.g., 'pending', 'completed', 'failed'
    transaction_date TIMESTAMP DEFAULT NOW(),
    FOREIGN KEY (order_id) REFERENCES public.orders (order_id)
);

-- 7. Inventory Logs (high-volume inserts for stock changes)
CREATE TABLE IF NOT EXISTS public.inventory_logs
(
    log_id          SERIAL PRIMARY KEY,
    product_id      INT  NOT NULL,
    change_quantity INT  NOT NULL, -- + for stock added, - for stock deducted
    reason          TEXT NOT NULL, -- e.g., 'order_fulfillment', 'restock', 'return'
    created_at      TIMESTAMP DEFAULT NOW(),
    FOREIGN KEY (product_id) REFERENCES public.products (product_id)
);

CREATE INDEX IF NOT EXISTS idx_products_category_id ON public.products(category_id);
CREATE INDEX IF NOT EXISTS idx_orders_user_id ON public.orders(user_id);
CREATE INDEX IF NOT EXISTS idx_order_items_order_id ON public.order_items(order_id);
CREATE INDEX IF NOT EXISTS idx_order_items_product_id ON public.order_items(product_id);
CREATE INDEX IF NOT EXISTS idx_payments_order_id ON public.payments(order_id);
CREATE INDEX IF NOT EXISTS idx_inventory_logs_product_id ON public.inventory_logs(product_id);