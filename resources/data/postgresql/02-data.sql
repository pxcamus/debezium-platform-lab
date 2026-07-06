INSERT INTO public.categories (name, description)
SELECT
    'Category ' || n AS name,
    'Description for Category ' || n AS description
FROM generate_series(1, 10) AS n;  -- Only 10 categories needed for demo

INSERT INTO public.products (name, price, stock_quantity, category_id)
SELECT
    'Product ' || n AS name,
    ROUND((RANDOM() * 90 + 10)::NUMERIC, 2) AS price,
    floor(random() * 101)::int AS stock_quantity,
    floor(random() * 10)::int + 1 AS category_id
FROM generate_series(1, 1000) AS n;

INSERT INTO public.users (username, email, created_at, last_login)
SELECT
    'user_' || n AS username,
    'user_' || n || '@example.com' AS email,
    NOW() - (n || ' minutes')::INTERVAL AS created_at,
    NOW() - (n || ' minutes')::INTERVAL AS last_login
FROM generate_series(1, 1000) AS n;

INSERT INTO public.orders (user_id, status, total_amount)
SELECT
    floor(random() * 1000)::int + 1 AS user_id,
    CASE floor(random() * 3)::int
        WHEN 0 THEN 'pending'
        WHEN 1 THEN 'shipped'
        ELSE 'cancelled'
    END AS status,
    ROUND((RANDOM() * 900 + 100)::NUMERIC, 2) AS total_amount
FROM generate_series(1, 1000) AS n;

INSERT INTO public.order_items (order_id, product_id, quantity, unit_price)
SELECT
    floor(random() * 1000)::int + 1 AS order_id,
    floor(random() * 1000)::int + 1 AS product_id,
    floor(random() * 5)::int + 1 AS quantity,
    ROUND((RANDOM() * 90 + 10)::NUMERIC, 2) AS unit_price
FROM generate_series(1, 1000) AS n;

INSERT INTO public.payments (order_id, amount, payment_method, status)
SELECT
    n AS order_id,  -- Link each payment to an order (1:1 for simplicity)
    ROUND((RANDOM() * 900 + 100)::NUMERIC, 2) AS amount,  -- Random amount between $100 and $1000
    CASE (RANDOM() * 2)::INT
        WHEN 0 THEN 'credit_card'
        ELSE 'paypal'
        END AS payment_method,
    CASE (RANDOM() * 2)::INT
        WHEN 0 THEN 'completed'
        ELSE 'pending'
        END AS status
FROM generate_series(1, 1000) AS n;

INSERT INTO public.inventory_logs (product_id, change_quantity, reason)
SELECT
    floor(random() * 1000)::int + 1 AS product_id,
    floor(random() * 21)::int - 10 AS change_quantity,
    CASE floor(random() * 2)::int
        WHEN 0 THEN 'order_fulfillment'
        ELSE 'restock'
    END AS reason
FROM generate_series(1, 1000) AS n;