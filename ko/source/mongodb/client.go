package mongodb

import (
	"context"
	"dbz-mage/ko/automation"
	"fmt"
	"net/url"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const (
	defaultDatabase   = "ecommerce"
	defaultUsername   = "app_owner"
	defaultPassword   = "app_owner"
	defaultReplicaSet = "mongodb"
	defaultAuthSource = "admin"
	defaultURI        = "mongodb://mongodb-0.mongodb-svc.databases.svc.cluster.local:27017,mongodb-1.mongodb-svc.databases.svc.cluster.local:27017,mongodb-2.mongodb-svc.databases.svc.cluster.local:27017"
)

type Config struct {
	URI              string
	Database         string
	Username         string
	Password         string
	ReplicaSet       string
	AuthSource       string
	DirectConnection string
}

func NewFromEnv(ctx context.Context) (*MongoDB, error) {
	// TODO: kubectl get svc mongodb-0-lb -n databases -o jsonpath='{.status.loadBalancer.ingress[0].ip}'

	config := Config{
		URI:              automation.Env("MONGODB_URI", defaultURI),
		Database:         automation.Env("MONGODB_DATABASE", defaultDatabase),
		Username:         automation.Env("MONGODB_USERNAME", defaultUsername),
		Password:         automation.Env("MONGODB_PASSWORD", defaultPassword),
		ReplicaSet:       automation.Env("MONGODB_REPLICA_SET", defaultReplicaSet),
		AuthSource:       automation.Env("MONGODB_AUTH_SOURCE", defaultAuthSource),
		DirectConnection: automation.Env("MONGODB_DIRECT_CONNECTION", ""),
	}

	return New(ctx, config)
}

func New(ctx context.Context, config Config) (*MongoDB, error) {
	uri := buildURI(config)

	clientOptions := options.Client().ApplyURI(uri)

	client, err := mongo.Connect(clientOptions)
	if err != nil {
		return nil, fmt.Errorf("connect to MongoDB: %w", err)
	}

	//pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	//defer cancel()
	//
	//if err := client.Ping(pingCtx, nil); err != nil {
	//	_ = client.Disconnect(context.Background())
	//	return nil, fmt.Errorf("ping MongoDB: %w", err)
	//}

	return &MongoDB{
		Client:   client,
		URI:      uri,
		Database: config.Database,
	}, nil
}

func buildURI(config Config) string {
	uri := config.URI

	parsed, err := url.Parse(uri)
	if err == nil {
		if config.Username != "" && config.Password != "" && parsed.User == nil {
			parsed.User = url.UserPassword(config.Username, config.Password)
		}

		if parsed.Path == "" {
			parsed.Path = "/"
		}

		query := parsed.Query()

		if config.AuthSource != "" {
			query.Set("authSource", config.AuthSource)
		}

		if config.DirectConnection != "" {
			query.Set("directConnection", config.DirectConnection)
		}

		if config.DirectConnection == "" && config.ReplicaSet != "" {
			query.Set("replicaSet", config.ReplicaSet)
		}

		parsed.RawQuery = query.Encode()

		return parsed.String()
	}

	if config.Username != "" && config.Password != "" {
		parsed, err := url.Parse(uri)
		if err == nil && parsed.User == nil {
			parsed.User = url.UserPassword(config.Username, config.Password)
			uri = parsed.String()
		}
	}

	query := url.Values{}

	if config.AuthSource != "" {
		query.Set("authSource", config.AuthSource)
	}

	if config.DirectConnection != "" {
		query.Set("directConnection", config.DirectConnection)
	}

	if config.DirectConnection == "" && config.ReplicaSet != "" {
		query.Set("replicaSet", config.ReplicaSet)
	}

	if encodedQuery := query.Encode(); encodedQuery != "" {
		separator := "?"
		if parsed, err := url.Parse(uri); err == nil {
			if parsed.Path == "" {
				parsed.Path = "/"
			}

			existingQuery := parsed.Query()
			for key, values := range query {
				for _, value := range values {
					existingQuery.Set(key, value)
				}
			}

			parsed.RawQuery = existingQuery.Encode()
			return parsed.String()
		}

		uri = uri + separator + encodedQuery
	}

	return uri
}
