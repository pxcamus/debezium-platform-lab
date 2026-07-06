-- Run as postgres/superuser in database: ecommerce

-- Database privileges needed by Debezium.
GRANT CONNECT ON DATABASE ecommerce TO debezium;

-- Schema/table privileges needed for snapshots and metadata access.
GRANT USAGE ON SCHEMA public TO debezium;

-- GRANT SELECT ON ALL TABLES IN SCHEMA public TO debezium; (less restrictive)
GRANT SELECT ON TABLE
    public.categories,
    public.products,
    public.users,
    public.orders,
    public.order_items,
    public.payments,
    public.inventory_logs
TO debezium;
-- GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO debezium; (not needed?)

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_publication
        WHERE pubname = 'dbz_publication'
    ) THEN
        CREATE PUBLICATION dbz_publication
        FOR TABLE
            public.categories,
            public.products,
            public.users,
            public.orders,
            public.order_items,
            public.payments,
            public.inventory_logs;
ELSE
        ALTER PUBLICATION dbz_publication
        SET TABLE
            public.categories,
            public.products,
            public.users,
            public.orders,
            public.order_items,
            public.payments,
            public.inventory_logs;
END IF;
END
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_replication_slots
        WHERE slot_name = 'debezium_slot'
    ) THEN
        PERFORM pg_create_logical_replication_slot('debezium_slot', 'pgoutput');
END IF;
END
$$;