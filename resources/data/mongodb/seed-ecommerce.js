const dbName = "ecommerce";
const database = db.getSiblingDB(dbName);

print(`Using database: ${dbName}`);

function createCollectionIfMissing(collectionName) {
    const existingCollections = database
        .getCollectionNames()
        .filter((name) => name === collectionName);

    if (existingCollections.length === 0) {
        database.createCollection(collectionName);
        print(`Created collection: ${collectionName}`);
    } else {
        print(`Collection already exists: ${collectionName}`);
    }
}

createCollectionIfMissing("categories");
createCollectionIfMissing("products");
createCollectionIfMissing("orders");
createCollectionIfMissing("debezium_signal");

database.categories.createIndex({ name: 1 }, { unique: true, name: "ux_categories_name" });
database.products.createIndex({ category_id: 1 }, { name: "idx_products_category_id" });
database.products.createIndex({ name: 1 }, { unique: true, name: "ux_products_name" });
database.orders.createIndex({ status: 1 }, { name: "idx_orders_status" });
database.orders.createIndex({ "user.user_id": 1 }, { name: "idx_orders_user_id" });
database.debezium_signal.createIndex({ id: 1 }, { unique: true, name: "ux_debezium_signal_id" });

const seedCreatedAt = new Date("2026-01-01T00:00:00Z");
const seedUpdatedAt = new Date("2026-01-01T00:00:00Z");

database.categories.updateOne(
    { _id: "cat-electronics" },
    {
        $set: {
            name: "Electronics",
            description: "Devices, gadgets, and accessories"
        },
        $setOnInsert: {
            created_at: seedCreatedAt
        }
    },
    { upsert: true }
);

database.categories.updateOne(
    { _id: "cat-books" },
    {
        $set: {
            name: "Books",
            description: "Printed and digital books"
        },
        $setOnInsert: {
            created_at: seedCreatedAt
        }
    },
    { upsert: true }
);

database.categories.updateOne(
    { _id: "cat-home" },
    {
        $set: {
            name: "Home",
            description: "Home and kitchen products"
        },
        $setOnInsert: {
            created_at: seedCreatedAt
        }
    },
    { upsert: true }
);

database.products.updateOne(
    { _id: "prod-laptop-1" },
    {
        $set: {
            name: "Developer Laptop",
            category_id: "cat-electronics",
            price: NumberDecimal("1299.99"),
            stock_quantity: 25,
            tags: ["computer", "developer", "portable"],
            updated_at: seedUpdatedAt
        },
        $setOnInsert: {
            created_at: seedCreatedAt
        }
    },
    { upsert: true }
);

database.products.updateOne(
    { _id: "prod-keyboard-1" },
    {
        $set: {
            name: "Mechanical Keyboard",
            category_id: "cat-electronics",
            price: NumberDecimal("149.99"),
            stock_quantity: 100,
            tags: ["keyboard", "accessory"],
            updated_at: seedUpdatedAt
        },
        $setOnInsert: {
            created_at: seedCreatedAt
        }
    },
    { upsert: true }
);

database.products.updateOne(
    { _id: "prod-book-1" },
    {
        $set: {
            name: "Change Data Capture Handbook",
            category_id: "cat-books",
            price: NumberDecimal("39.99"),
            stock_quantity: 50,
            tags: ["book", "data", "cdc"],
            updated_at: seedUpdatedAt
        },
        $setOnInsert: {
            created_at: seedCreatedAt
        }
    },
    { upsert: true }
);

database.orders.updateOne(
    { _id: "order-1001" },
    {
        $set: {
            user: {
                user_id: "user-1",
                username: "alice",
                email: "alice@example.com"
            },
            status: "pending",
            total_amount: NumberDecimal("1449.98"),
            items: [
                {
                    product_id: "prod-laptop-1",
                    name: "Developer Laptop",
                    quantity: 1,
                    unit_price: NumberDecimal("1299.99")
                },
                {
                    product_id: "prod-keyboard-1",
                    name: "Mechanical Keyboard",
                    quantity: 1,
                    unit_price: NumberDecimal("149.99")
                }
            ],
            updated_at: seedUpdatedAt
        },
        $setOnInsert: {
            created_at: new Date("2026-01-01T10:00:00Z")
        }
    },
    { upsert: true }
);

database.orders.updateOne(
    { _id: "order-1002" },
    {
        $set: {
            user: {
                user_id: "user-2",
                username: "bob",
                email: "bob@example.com"
            },
            status: "paid",
            total_amount: NumberDecimal("39.99"),
            items: [
                {
                    product_id: "prod-book-1",
                    name: "Change Data Capture Handbook",
                    quantity: 1,
                    unit_price: NumberDecimal("39.99")
                }
            ],
            updated_at: seedUpdatedAt
        },
        $setOnInsert: {
            created_at: new Date("2026-01-02T11:00:00Z")
        }
    },
    { upsert: true }
);

print("Seed completed.");
printjson({
    categories: database.categories.countDocuments(),
    products: database.products.countDocuments(),
    orders: database.orders.countDocuments(),
    signals: database.debezium_signal.countDocuments()
});