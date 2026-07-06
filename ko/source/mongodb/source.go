package mongodb

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

const (
	categoriesCollection     = "categories"
	productsCollection       = "products"
	ordersCollection         = "orders"
	debeziumSignalCollection = "debezium_signal"
)

type MongoDB struct {
	Client   *mongo.Client
	URI      string
	Database string
	Logger   *slog.Logger
}

func (m *MongoDB) logger() *slog.Logger {
	if m.Logger != nil {
		return m.Logger
	}

	return slog.Default()
}

func (m *MongoDB) Wait(ctx context.Context) error {
	log := m.logger()

	log.Info("Waiting for MongoDB", "database", m.Database)

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	timeout := time.NewTimer(2 * time.Minute)
	defer timeout.Stop()

	for {
		pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err := m.Client.Ping(pingCtx, readpref.Primary())
		cancel()

		if err == nil {
			log.Info("MongoDB is reachable", "database", m.Database)
			return nil
		}

		log.Debug("MongoDB ping failed, retrying", "error", err)

		select {
		case <-ctx.Done():
			log.Error("MongoDB wait cancelled", "error", ctx.Err())
			return ctx.Err()
		case <-timeout.C:
			log.Error("Timed out waiting for MongoDB", "error", err)
			return fmt.Errorf("wait for MongoDB: %w", err)
		case <-ticker.C:
		}
	}
}

func (m *MongoDB) Setup(ctx context.Context) error {
	log := m.logger()
	database := m.Client.Database(m.Database)

	log.Info("Setting up MongoDB database", "database", m.Database)

	if err := createCollectionIfMissing(ctx, log, database, categoriesCollection); err != nil {
		return err
	}

	if err := createCollectionIfMissing(ctx, log, database, productsCollection); err != nil {
		return err
	}

	if err := createCollectionIfMissing(ctx, log, database, ordersCollection); err != nil {
		return err
	}

	if err := createCollectionIfMissing(ctx, log, database, debeziumSignalCollection); err != nil {
		return err
	}

	if err := createIndexes(ctx, log, database); err != nil {
		return err
	}

	log.Info("MongoDB setup completed", "database", m.Database)

	return nil
}

func (m *MongoDB) Populate(ctx context.Context) error {
	log := m.logger()
	database := m.Client.Database(m.Database)

	log.Info("Populating MongoDB database", "database", m.Database)

	seedCreatedAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	seedUpdatedAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	categories := database.Collection(categoriesCollection)
	products := database.Collection(productsCollection)
	orders := database.Collection(ordersCollection)

	if err := upsertCategory(ctx, log, categories, "cat-electronics", "Electronics", "Devices, gadgets, and accessories", seedCreatedAt); err != nil {
		return err
	}

	if err := upsertCategory(ctx, log, categories, "cat-books", "Books", "Printed and digital books", seedCreatedAt); err != nil {
		return err
	}

	if err := upsertCategory(ctx, log, categories, "cat-home", "Home", "Home and kitchen products", seedCreatedAt); err != nil {
		return err
	}

	if err := upsertProduct(ctx, log, products, bson.M{
		"_id":          "prod-laptop-1",
		"name":         "Developer Laptop",
		"description":  "High-performance laptop for development workloads",
		"category_id":  "cat-electronics",
		"price":        1999.99,
		"stock":        15,
		"created_at":   seedCreatedAt,
		"updated_at":   seedUpdatedAt,
		"discontinued": false,
	}); err != nil {
		return err
	}

	if err := upsertProduct(ctx, log, products, bson.M{
		"_id":          "prod-keyboard-1",
		"name":         "Mechanical Keyboard",
		"description":  "Compact mechanical keyboard",
		"category_id":  "cat-electronics",
		"price":        129.99,
		"stock":        40,
		"created_at":   seedCreatedAt,
		"updated_at":   seedUpdatedAt,
		"discontinued": false,
	}); err != nil {
		return err
	}

	if err := upsertProduct(ctx, log, products, bson.M{
		"_id":          "prod-book-1",
		"name":         "Change Data Capture Handbook",
		"description":  "Practical guide to CDC patterns",
		"category_id":  "cat-books",
		"price":        49.99,
		"stock":        100,
		"created_at":   seedCreatedAt,
		"updated_at":   seedUpdatedAt,
		"discontinued": false,
	}); err != nil {
		return err
	}

	if err := upsertOrder(ctx, log, orders, bson.M{
		"_id":    "order-1001",
		"status": "PLACED",
		"user": bson.M{
			"user_id": "user-1",
			"name":    "Alice Example",
			"email":   "alice@example.com",
		},
		"items": bson.A{
			bson.M{
				"product_id": "prod-laptop-1",
				"name":       "Developer Laptop",
				"quantity":   1,
				"unit_price": 1999.99,
			},
			bson.M{
				"product_id": "prod-keyboard-1",
				"name":       "Mechanical Keyboard",
				"quantity":   1,
				"unit_price": 129.99,
			},
		},
		"total":      2129.98,
		"created_at": seedCreatedAt,
		"updated_at": seedUpdatedAt,
	}); err != nil {
		return err
	}

	if err := upsertOrder(ctx, log, orders, bson.M{
		"_id":    "order-1002",
		"status": "SHIPPED",
		"user": bson.M{
			"user_id": "user-2",
			"name":    "Bob Example",
			"email":   "bob@example.com",
		},
		"items": bson.A{
			bson.M{
				"product_id": "prod-book-1",
				"name":       "Change Data Capture Handbook",
				"quantity":   2,
				"unit_price": 49.99,
			},
		},
		"total":      99.98,
		"created_at": seedCreatedAt,
		"updated_at": seedUpdatedAt,
	}); err != nil {
		return err
	}

	log.Info("MongoDB population completed", "database", m.Database)

	return nil
}

func (m *MongoDB) Reset(ctx context.Context) error {
	log := m.logger()
	database := m.Client.Database(m.Database)

	log.Warn("Resetting MongoDB database", "database", m.Database)

	collections := []string{
		categoriesCollection,
		productsCollection,
		ordersCollection,
		debeziumSignalCollection,
	}

	for _, collectionName := range collections {
		log.Info("Dropping MongoDB collection", "collection", collectionName)

		if err := database.Collection(collectionName).Drop(ctx); err != nil {
			log.Error("Failed to drop MongoDB collection", "collection", collectionName, "error", err)
			return fmt.Errorf("drop MongoDB collection %q: %w", collectionName, err)
		}
	}

	if err := m.Setup(ctx); err != nil {
		return err
	}

	return m.Populate(ctx)
}

func (m *MongoDB) Close(ctx context.Context) error {
	m.logger().Debug("Closing MongoDB client", "database", m.Database)
	return m.Client.Disconnect(ctx)
}

func createCollectionIfMissing(ctx context.Context, log *slog.Logger, database *mongo.Database, collectionName string) error {
	names, err := database.ListCollectionNames(ctx, bson.M{"name": collectionName})
	if err != nil {
		log.Error("Failed to list MongoDB collections", "error", err)
		return fmt.Errorf("list MongoDB collections: %w", err)
	}

	if len(names) > 0 {
		log.Info("MongoDB collection already exists", "collection", collectionName)
		return nil
	}

	log.Info("Creating MongoDB collection", "collection", collectionName)

	if err := database.CreateCollection(ctx, collectionName); err != nil {
		log.Error("Failed to create MongoDB collection", "collection", collectionName, "error", err)
		return fmt.Errorf("create MongoDB collection %q: %w", collectionName, err)
	}

	return nil
}

func createIndexes(ctx context.Context, log *slog.Logger, database *mongo.Database) error {
	indexes := map[string][]mongo.IndexModel{
		categoriesCollection: {
			{
				Keys:    bson.D{{Key: "name", Value: 1}},
				Options: options.Index().SetUnique(true).SetName("ux_categories_name"),
			},
		},
		productsCollection: {
			{
				Keys:    bson.D{{Key: "category_id", Value: 1}},
				Options: options.Index().SetName("idx_products_category_id"),
			},
			{
				Keys:    bson.D{{Key: "name", Value: 1}},
				Options: options.Index().SetUnique(true).SetName("ux_products_name"),
			},
		},
		ordersCollection: {
			{
				Keys:    bson.D{{Key: "status", Value: 1}},
				Options: options.Index().SetName("idx_orders_status"),
			},
			{
				Keys:    bson.D{{Key: "user.user_id", Value: 1}},
				Options: options.Index().SetName("idx_orders_user_id"),
			},
		},
		debeziumSignalCollection: {
			{
				Keys:    bson.D{{Key: "id", Value: 1}},
				Options: options.Index().SetUnique(true).SetName("ux_debezium_signal_id"),
			},
		},
	}

	for collectionName, collectionIndexes := range indexes {
		log.Info("Creating MongoDB indexes", "collection", collectionName, "count", len(collectionIndexes))

		collection := database.Collection(collectionName)

		if _, err := collection.Indexes().CreateMany(ctx, collectionIndexes); err != nil {
			log.Error("Failed to create MongoDB indexes", "collection", collectionName, "error", err)
			return fmt.Errorf("create MongoDB indexes for collection %q: %w", collectionName, err)
		}
	}

	return nil
}

func upsertCategory(ctx context.Context, log *slog.Logger, collection *mongo.Collection, id string, name string, description string, createdAt time.Time) error {
	_, err := collection.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{
			"$set": bson.M{
				"name":        name,
				"description": description,
			},
			"$setOnInsert": bson.M{
				"created_at": createdAt,
			},
		},
		options.UpdateOne().SetUpsert(true),
	)
	if err != nil {
		log.Error("Failed to upsert MongoDB category", "id", id, "error", err)
		return fmt.Errorf("upsert MongoDB category %q: %w", id, err)
	}

	log.Debug("Upserted MongoDB category", "id", id)

	return nil
}

func upsertProduct(ctx context.Context, log *slog.Logger, collection *mongo.Collection, product bson.M) error {
	id := product["_id"]

	_, err := collection.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{
			"$set": product,
		},
		options.UpdateOne().SetUpsert(true),
	)
	if err != nil {
		log.Error("Failed to upsert MongoDB product", "id", id, "error", err)
		return fmt.Errorf("upsert MongoDB product %q: %w", id, err)
	}

	log.Debug("Upserted MongoDB product", "id", id)

	return nil
}

func upsertOrder(ctx context.Context, log *slog.Logger, collection *mongo.Collection, order bson.M) error {
	id := order["_id"]

	_, err := collection.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{
			"$set": order,
		},
		options.UpdateOne().SetUpsert(true),
	)
	if err != nil {
		log.Error("Failed to upsert MongoDB order", "id", id, "error", err)
		return fmt.Errorf("upsert MongoDB order %q: %w", id, err)
	}

	log.Debug("Upserted MongoDB order", "id", id)

	return nil
}
