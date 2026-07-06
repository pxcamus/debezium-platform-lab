package postgresql

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

const (
	categoriesTable    = "categories"
	productsTable      = "products"
	usersTable         = "users"
	ordersTable        = "orders"
	orderItemsTable    = "order_items"
	paymentsTable      = "payments"
	inventoryLogsTable = "inventory_logs"
)

var ecommerceTables = []string{
	categoriesTable,
	productsTable,
	usersTable,
	ordersTable,
	orderItemsTable,
	paymentsTable,
	inventoryLogsTable,
}

type PostgreSQL struct {
	DB              *sql.DB
	SuperDB         *sql.DB
	Host            string
	Port            int
	Database        string
	Username        string
	DebeziumUser    string
	PublicationName string
	SlotName        string
	Logger          *slog.Logger
}

func (p *PostgreSQL) logger() *slog.Logger {
	if p.Logger != nil {
		return p.Logger
	}

	return slog.Default()
}

func (p *PostgreSQL) Wait(ctx context.Context) error {
	log := p.logger()

	log.Info("Waiting for PostgreSQL", "database", p.Database, "host", p.Host, "port", p.Port)

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	timeout := time.NewTimer(2 * time.Minute)
	defer timeout.Stop()

	for {
		pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err := p.DB.PingContext(pingCtx)
		cancel()

		if err == nil {
			log.Info("PostgreSQL is reachable", "database", p.Database)
			return nil
		}

		log.Debug("PostgreSQL ping failed, retrying", "error", err)

		select {
		case <-ctx.Done():
			log.Error("PostgreSQL wait cancelled", "error", ctx.Err())
			return ctx.Err()
		case <-timeout.C:
			log.Error("Timed out waiting for PostgreSQL", "error", err)
			return fmt.Errorf("wait for PostgreSQL: %w", err)
		case <-ticker.C:
		}
	}
}

func (p *PostgreSQL) Setup(ctx context.Context) error {
	log := p.logger()

	log.Info("Setting up PostgreSQL database", "database", p.Database)

	if err := p.setupApicurio(ctx); err != nil {
		return err
	}

	if err := p.createTables(ctx); err != nil {
		return err
	}

	if err := p.createIndexes(ctx); err != nil {
		return err
	}

	if err := p.setupDebezium(ctx); err != nil {
		return err
	}

	log.Info("PostgreSQL setup completed", "database", p.Database)

	return nil
}

func (p *PostgreSQL) setupApicurio(ctx context.Context) error {
	db := p.SuperDB
	if db == nil {
		db = p.DB
	}

	log := p.logger()

	log.Info("Setting up Apicurio PostgreSQL database")

	statements := []string{
		`DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1
				FROM pg_roles
				WHERE rolname = 'apicurio'
			) THEN
				CREATE ROLE apicurio WITH LOGIN PASSWORD 'apicurio';
			END IF;
		END
		$$`,
	}

	if err := p.execStatements(ctx, db, "setup Apicurio PostgreSQL role", statements); err != nil {
		return err
	}

	var exists bool
	if err := db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = 'apicurio')`).Scan(&exists); err != nil {
		log.Error("Failed to check Apicurio database existence", "error", err)
		return fmt.Errorf("check Apicurio database existence: %w", err)
	}

	if !exists {
		log.Info("Creating Apicurio PostgreSQL database")

		if _, err := db.ExecContext(ctx, `CREATE DATABASE apicurio OWNER apicurio`); err != nil {
			log.Error("Failed to create Apicurio PostgreSQL database", "error", err)
			return fmt.Errorf("create Apicurio PostgreSQL database: %w", err)
		}
	}

	if _, err := db.ExecContext(ctx, `GRANT CONNECT ON DATABASE apicurio TO apicurio`); err != nil {
		log.Error("Failed to grant Apicurio PostgreSQL database permissions", "error", err)
		return fmt.Errorf("grant Apicurio PostgreSQL database permissions: %w", err)
	}

	log.Info("Apicurio PostgreSQL database setup completed")

	return nil
}

func (p *PostgreSQL) Populate(ctx context.Context) error {
	log := p.logger()

	log.Info("Populating PostgreSQL database", "database", p.Database)

	tx, err := p.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin PostgreSQL population transaction: %w", err)
	}
	defer rollback(tx, log)

	if err := insertCategories(ctx, tx); err != nil {
		return err
	}

	if err := insertProducts(ctx, tx); err != nil {
		return err
	}

	if err := insertUsers(ctx, tx); err != nil {
		return err
	}

	if err := insertOrders(ctx, tx); err != nil {
		return err
	}

	if err := insertOrderItems(ctx, tx); err != nil {
		return err
	}

	if err := insertPayments(ctx, tx); err != nil {
		return err
	}

	if err := insertInventoryLogs(ctx, tx); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit PostgreSQL population transaction: %w", err)
	}

	log.Info("PostgreSQL population completed", "database", p.Database)

	return nil
}

func (p *PostgreSQL) Reset(ctx context.Context) error {
	log := p.logger()

	log.Warn("Resetting PostgreSQL database", "database", p.Database)

	if err := p.dropReplicationSlot(ctx); err != nil {
		return err
	}

	if err := p.dropPublication(ctx); err != nil {
		return err
	}

	if err := p.dropTables(ctx); err != nil {
		return err
	}

	if err := p.Setup(ctx); err != nil {
		return err
	}

	return p.Populate(ctx)
}

func (p *PostgreSQL) Close() error {
	log := p.logger()

	log.Debug("Closing PostgreSQL connections", "database", p.Database)

	var closeErr error

	if p.SuperDB != nil {
		if err := p.SuperDB.Close(); err != nil {
			closeErr = fmt.Errorf("close PostgreSQL superuser connection: %w", err)
		}
	}

	if p.DB != nil {
		if err := p.DB.Close(); err != nil {
			if closeErr != nil {
				return fmt.Errorf("%v; close PostgreSQL connection: %w", closeErr, err)
			}

			closeErr = fmt.Errorf("close PostgreSQL connection: %w", err)
		}
	}

	return closeErr
}

func (p *PostgreSQL) createTables(ctx context.Context) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS public.categories
		(
			category_id SERIAL PRIMARY KEY,
			name        TEXT NOT NULL,
			description TEXT
		)`,

		`CREATE TABLE IF NOT EXISTS public.products
		(
			product_id     SERIAL PRIMARY KEY,
			name           TEXT           NOT NULL,
			price          NUMERIC(10, 2) NOT NULL,
			stock_quantity INT            NOT NULL DEFAULT 0,
			category_id    INT            NOT NULL,
			updated_at     TIMESTAMP               DEFAULT NOW(),
			FOREIGN KEY (category_id) REFERENCES public.categories (category_id)
		)`,

		`CREATE TABLE IF NOT EXISTS public.users
		(
			user_id    SERIAL PRIMARY KEY,
			username   TEXT NOT NULL UNIQUE,
			email      TEXT NOT NULL UNIQUE,
			created_at TIMESTAMP DEFAULT NOW(),
			last_login TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS public.orders
		(
			order_id     SERIAL PRIMARY KEY,
			user_id      INT            NOT NULL,
			order_date   TIMESTAMP DEFAULT NOW(),
			status       TEXT           NOT NULL,
			total_amount NUMERIC(10, 2) NOT NULL,
			FOREIGN KEY (user_id) REFERENCES public.users (user_id)
		)`,

		`CREATE TABLE IF NOT EXISTS public.order_items
		(
			order_item_id SERIAL PRIMARY KEY,
			order_id      INT            NOT NULL,
			product_id    INT            NOT NULL,
			quantity      INT            NOT NULL,
			unit_price    NUMERIC(10, 2) NOT NULL,
			FOREIGN KEY (order_id) REFERENCES public.orders (order_id),
			FOREIGN KEY (product_id) REFERENCES public.products (product_id)
		)`,

		`CREATE TABLE IF NOT EXISTS public.payments
		(
			payment_id       SERIAL PRIMARY KEY,
			order_id         INT            NOT NULL,
			amount           NUMERIC(10, 2) NOT NULL,
			payment_method   TEXT           NOT NULL,
			status           TEXT           NOT NULL,
			transaction_date TIMESTAMP DEFAULT NOW(),
			FOREIGN KEY (order_id) REFERENCES public.orders (order_id)
		)`,

		`CREATE TABLE IF NOT EXISTS public.inventory_logs
		(
			log_id          SERIAL PRIMARY KEY,
			product_id      INT  NOT NULL,
			change_quantity INT  NOT NULL,
			reason          TEXT NOT NULL,
			created_at      TIMESTAMP DEFAULT NOW(),
			FOREIGN KEY (product_id) REFERENCES public.products (product_id)
		)`,
	}

	return p.execStatements(ctx, p.DB, "create PostgreSQL tables", statements)
}

func (p *PostgreSQL) createIndexes(ctx context.Context) error {
	statements := []string{
		`CREATE INDEX IF NOT EXISTS idx_products_category_id ON public.products(category_id)`,
		`CREATE INDEX IF NOT EXISTS idx_orders_user_id ON public.orders(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_order_items_order_id ON public.order_items(order_id)`,
		`CREATE INDEX IF NOT EXISTS idx_order_items_product_id ON public.order_items(product_id)`,
		`CREATE INDEX IF NOT EXISTS idx_payments_order_id ON public.payments(order_id)`,
		`CREATE INDEX IF NOT EXISTS idx_inventory_logs_product_id ON public.inventory_logs(product_id)`,
	}

	return p.execStatements(ctx, p.DB, "create PostgreSQL indexes", statements)
}

func (p *PostgreSQL) setupDebezium(ctx context.Context) error {
	db := p.SuperDB
	if db == nil {
		db = p.DB
	}

	log := p.logger()

	log.Info(
		"Configuring PostgreSQL Debezium permissions",
		"database", p.Database,
		"user", p.DebeziumUser,
		"publication", p.PublicationName,
		"slot", p.SlotName,
	)

	statements := []string{
		fmt.Sprintf(`GRANT CONNECT ON DATABASE %s TO %s`, quoteIdentifier(p.Database), quoteIdentifier(p.DebeziumUser)),
		fmt.Sprintf(`GRANT USAGE ON SCHEMA public TO %s`, quoteIdentifier(p.DebeziumUser)),
		fmt.Sprintf(`GRANT SELECT ON TABLE %s TO %s`, qualifiedTableList(ecommerceTables), quoteIdentifier(p.DebeziumUser)),
		fmt.Sprintf(
			`DO $$
			BEGIN
				IF NOT EXISTS (
					SELECT 1
					FROM pg_publication
					WHERE pubname = %s
				) THEN
					CREATE PUBLICATION %s FOR TABLE %s;
				ELSE
					ALTER PUBLICATION %s SET TABLE %s;
				END IF;
			END
			$$`,
			quoteLiteral(p.PublicationName),
			quoteIdentifier(p.PublicationName),
			qualifiedTableList(ecommerceTables),
			quoteIdentifier(p.PublicationName),
			qualifiedTableList(ecommerceTables),
		),
		fmt.Sprintf(
			`DO $$
			BEGIN
				IF NOT EXISTS (
					SELECT 1
					FROM pg_replication_slots
					WHERE slot_name = %s
				) THEN
					PERFORM pg_create_logical_replication_slot(%s, 'pgoutput');
				END IF;
			END
			$$`,
			quoteLiteral(p.SlotName),
			quoteLiteral(p.SlotName),
		),
	}

	return p.execStatements(ctx, db, "configure PostgreSQL Debezium permissions", statements)
}

func (p *PostgreSQL) dropTables(ctx context.Context) error {
	statement := `DROP TABLE IF EXISTS
		public.inventory_logs,
		public.payments,
		public.order_items,
		public.orders,
		public.products,
		public.users,
		public.categories
	CASCADE`

	if _, err := p.DB.ExecContext(ctx, statement); err != nil {
		p.logger().Error("Failed to drop PostgreSQL tables", "error", err)
		return fmt.Errorf("drop PostgreSQL tables: %w", err)
	}

	return nil
}

func (p *PostgreSQL) dropPublication(ctx context.Context) error {
	db := p.SuperDB
	if db == nil {
		db = p.DB
	}

	statement := fmt.Sprintf(`DROP PUBLICATION IF EXISTS %s`, quoteIdentifier(p.PublicationName))

	if _, err := db.ExecContext(ctx, statement); err != nil {
		p.logger().Error("Failed to drop PostgreSQL publication", "publication", p.PublicationName, "error", err)
		return fmt.Errorf("drop PostgreSQL publication %q: %w", p.PublicationName, err)
	}

	return nil
}

func (p *PostgreSQL) dropReplicationSlot(ctx context.Context) error {
	db := p.SuperDB
	if db == nil {
		db = p.DB
	}

	statement := fmt.Sprintf(
		`SELECT pg_drop_replication_slot(slot_name)
		FROM pg_replication_slots
		WHERE slot_name = %s`,
		quoteLiteral(p.SlotName),
	)

	if _, err := db.ExecContext(ctx, statement); err != nil {
		p.logger().Error("Failed to drop PostgreSQL replication slot", "slot", p.SlotName, "error", err)
		return fmt.Errorf("drop PostgreSQL replication slot %q: %w", p.SlotName, err)
	}

	return nil
}

func (p *PostgreSQL) execStatements(ctx context.Context, db *sql.DB, action string, statements []string) error {
	log := p.logger()

	for _, statement := range statements {
		log.Debug("Executing PostgreSQL statement", "action", action, "statement", statement)

		if _, err := db.ExecContext(ctx, statement); err != nil {
			log.Error("Failed to execute PostgreSQL statement", "action", action, "error", err)
			return fmt.Errorf("%s: %w", action, err)
		}
	}

	return nil
}

func insertCategories(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO public.categories (category_id, name, description)
		SELECT
			n,
			'Category ' || n,
			'Description for Category ' || n
		FROM generate_series(1, 10) AS n
		ON CONFLICT (category_id) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description
	`)

	if err != nil {
		return fmt.Errorf("insert PostgreSQL categories: %w", err)
	}

	return resetSequence(ctx, tx, "public.categories_category_id_seq", "public.categories", "category_id")
}

func insertProducts(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO public.products (product_id, name, price, stock_quantity, category_id)
		SELECT
			n,
			'Product ' || n,
			ROUND((RANDOM() * 90 + 10)::NUMERIC, 2),
			floor(random() * 101)::int,
			floor(random() * 10)::int + 1
		FROM generate_series(1, 1000) AS n
		ON CONFLICT (product_id) DO UPDATE SET
			name = EXCLUDED.name,
			price = EXCLUDED.price,
			stock_quantity = EXCLUDED.stock_quantity,
			category_id = EXCLUDED.category_id,
			updated_at = NOW()
	`)

	if err != nil {
		return fmt.Errorf("insert PostgreSQL products: %w", err)
	}

	return resetSequence(ctx, tx, "public.products_product_id_seq", "public.products", "product_id")
}

func insertUsers(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO public.users (user_id, username, email, created_at, last_login)
		SELECT
			n,
			'user_' || n,
			'user_' || n || '@example.com',
			NOW() - (n || ' minutes')::INTERVAL,
			NOW() - (n || ' minutes')::INTERVAL
		FROM generate_series(1, 1000) AS n
		ON CONFLICT (user_id) DO UPDATE SET
			username = EXCLUDED.username,
			email = EXCLUDED.email,
			created_at = EXCLUDED.created_at,
			last_login = EXCLUDED.last_login
	`)

	if err != nil {
		return fmt.Errorf("insert PostgreSQL users: %w", err)
	}

	return resetSequence(ctx, tx, "public.users_user_id_seq", "public.users", "user_id")
}

func insertOrders(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO public.orders (order_id, user_id, status, total_amount)
		SELECT
			n,
			floor(random() * 1000)::int + 1,
			CASE floor(random() * 3)::int
				WHEN 0 THEN 'pending'
				WHEN 1 THEN 'shipped'
				ELSE 'cancelled'
			END,
			ROUND((RANDOM() * 900 + 100)::NUMERIC, 2)
		FROM generate_series(1, 1000) AS n
		ON CONFLICT (order_id) DO UPDATE SET
			user_id = EXCLUDED.user_id,
			status = EXCLUDED.status,
			total_amount = EXCLUDED.total_amount
	`)

	if err != nil {
		return fmt.Errorf("insert PostgreSQL orders: %w", err)
	}

	return resetSequence(ctx, tx, "public.orders_order_id_seq", "public.orders", "order_id")
}

func insertOrderItems(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO public.order_items (order_item_id, order_id, product_id, quantity, unit_price)
		SELECT
			n,
			floor(random() * 1000)::int + 1,
			floor(random() * 1000)::int + 1,
			floor(random() * 5)::int + 1,
			ROUND((RANDOM() * 90 + 10)::NUMERIC, 2)
		FROM generate_series(1, 1000) AS n
		ON CONFLICT (order_item_id) DO UPDATE SET
			order_id = EXCLUDED.order_id,
			product_id = EXCLUDED.product_id,
			quantity = EXCLUDED.quantity,
			unit_price = EXCLUDED.unit_price
	`)

	if err != nil {
		return fmt.Errorf("insert PostgreSQL order items: %w", err)
	}

	return resetSequence(ctx, tx, "public.order_items_order_item_id_seq", "public.order_items", "order_item_id")
}

func insertPayments(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO public.payments (payment_id, order_id, amount, payment_method, status)
		SELECT
			n,
			n,
			ROUND((RANDOM() * 900 + 100)::NUMERIC, 2),
			CASE (RANDOM() * 2)::INT
				WHEN 0 THEN 'credit_card'
				ELSE 'paypal'
			END,
			CASE (RANDOM() * 2)::INT
				WHEN 0 THEN 'completed'
				ELSE 'pending'
			END
		FROM generate_series(1, 1000) AS n
		ON CONFLICT (payment_id) DO UPDATE SET
			order_id = EXCLUDED.order_id,
			amount = EXCLUDED.amount,
			payment_method = EXCLUDED.payment_method,
			status = EXCLUDED.status
	`)

	if err != nil {
		return fmt.Errorf("insert PostgreSQL payments: %w", err)
	}

	return resetSequence(ctx, tx, "public.payments_payment_id_seq", "public.payments", "payment_id")
}

func insertInventoryLogs(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO public.inventory_logs (log_id, product_id, change_quantity, reason)
		SELECT
			n,
			floor(random() * 1000)::int + 1,
			floor(random() * 21)::int - 10,
			CASE floor(random() * 2)::int
				WHEN 0 THEN 'order_fulfillment'
				ELSE 'restock'
			END
		FROM generate_series(1, 1000) AS n
		ON CONFLICT (log_id) DO UPDATE SET
			product_id = EXCLUDED.product_id,
			change_quantity = EXCLUDED.change_quantity,
			reason = EXCLUDED.reason
	`)

	if err != nil {
		return fmt.Errorf("insert PostgreSQL inventory logs: %w", err)
	}

	return resetSequence(ctx, tx, "public.inventory_logs_log_id_seq", "public.inventory_logs", "log_id")
}

func resetSequence(ctx context.Context, tx *sql.Tx, sequenceName string, tableName string, columnName string) error {
	statement := fmt.Sprintf(
		`SELECT setval(%s, COALESCE((SELECT MAX(%s) FROM %s), 1), true)`,
		quoteLiteral(sequenceName),
		quoteIdentifier(columnName),
		tableName,
	)

	if _, err := tx.ExecContext(ctx, statement); err != nil {
		return fmt.Errorf("reset PostgreSQL sequence %s: %w", sequenceName, err)
	}

	return nil
}

func rollback(tx *sql.Tx, log *slog.Logger) {
	if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
		log.Warn("Failed to rollback PostgreSQL transaction", "error", err)
	}
}

func qualifiedTableList(tables []string) string {
	qualified := make([]string, 0, len(tables))

	for _, table := range tables {
		qualified = append(qualified, "public."+quoteIdentifier(table))
	}

	return strings.Join(qualified, ", ")
}

func quoteIdentifier(identifier string) string {
	return `"` + strings.ReplaceAll(identifier, `"`, `""`) + `"`
}

func quoteLiteral(value string) string {
	return `'` + strings.ReplaceAll(value, `'`, `''`) + `'`
}
