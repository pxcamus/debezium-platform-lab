package postgresql

import (
	"context"
	"database/sql"
	"dbz-mage/ko/automation"
	"fmt"
	"strconv"
	"strings"

	_ "github.com/lib/pq"
)

const (
	defaultHost            = "postgresql-rw.databases.svc.cluster.local"
	defaultPort            = 5432
	defaultDatabase        = "ecommerce"
	defaultUsername        = "app_owner"
	defaultPassword        = "app_owner"
	defaultSSLMode         = "disable"
	defaultSuperUsername   = "postgres"
	defaultSuperPassword   = "postgres"
	defaultDebeziumUser    = "debezium"
	defaultPublicationName = "dbz_publication"
	defaultSlotName        = "debezium_slot"
)

type Config struct {
	Host            string
	Port            int
	Database        string
	Username        string
	Password        string
	SSLMode         string
	SuperUsername   string
	SuperPassword   string
	DebeziumUser    string
	PublicationName string
	SlotName        string
}

func NewFromEnv(ctx context.Context) (*PostgreSQL, error) {
	port, err := strconv.Atoi(automation.Env("POSTGRESQL_PORT", strconv.Itoa(defaultPort)))
	if err != nil {
		return nil, fmt.Errorf("parse POSTGRESQL_PORT: %w", err)
	}

	config := Config{
		Host:            automation.Env("POSTGRESQL_HOST", defaultHost),
		Port:            port,
		Database:        automation.Env("POSTGRESQL_DATABASE", defaultDatabase),
		Username:        automation.Env("POSTGRESQL_USERNAME", defaultUsername),
		Password:        automation.Env("POSTGRESQL_PASSWORD", defaultPassword),
		SSLMode:         automation.Env("POSTGRESQL_SSLMODE", defaultSSLMode),
		SuperUsername:   automation.Env("POSTGRESQL_SUPER_USERNAME", defaultSuperUsername),
		SuperPassword:   automation.Env("POSTGRESQL_SUPER_PASSWORD", defaultSuperPassword),
		DebeziumUser:    automation.Env("POSTGRESQL_DEBEZIUM_USER", defaultDebeziumUser),
		PublicationName: automation.Env("POSTGRESQL_PUBLICATION", defaultPublicationName),
		SlotName:        automation.Env("POSTGRESQL_SLOT", defaultSlotName),
	}

	return New(ctx, config)
}

func New(ctx context.Context, config Config) (*PostgreSQL, error) {
	if config.Port == 0 {
		config.Port = defaultPort
	}

	if config.SSLMode == "" {
		config.SSLMode = defaultSSLMode
	}

	if config.PublicationName == "" {
		config.PublicationName = defaultPublicationName
	}

	if config.SlotName == "" {
		config.SlotName = defaultSlotName
	}

	if config.DebeziumUser == "" {
		config.DebeziumUser = defaultDebeziumUser
	}

	dsn := buildDSN(config.Host, config.Port, config.Database, config.Username, config.Password, config.SSLMode)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open PostgreSQL connection: %w", err)
	}

	//pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	//defer cancel()
	//
	//if err := db.PingContext(pingCtx); err != nil {
	//	_ = db.Close()
	//	return nil, fmt.Errorf("ping PostgreSQL: %w", err)
	//}

	var superDB *sql.DB
	if config.SuperUsername != "" && config.SuperPassword != "" {
		superDSN := buildDSN(config.Host, config.Port, config.Database, config.SuperUsername, config.SuperPassword, config.SSLMode)

		superDB, err = sql.Open("postgres", superDSN)
		if err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("open PostgreSQL superuser connection: %w", err)
		}

		//if err := superDB.PingContext(pingCtx); err != nil {
		//	_ = superDB.Close()
		//	_ = db.Close()
		//	return nil, fmt.Errorf("ping PostgreSQL superuser connection: %w", err)
		//}
	}

	return &PostgreSQL{
		DB:              db,
		SuperDB:         superDB,
		Host:            config.Host,
		Port:            config.Port,
		Database:        config.Database,
		Username:        config.Username,
		DebeziumUser:    config.DebeziumUser,
		PublicationName: config.PublicationName,
		SlotName:        config.SlotName,
	}, nil
}

func buildDSN(host string, port int, database string, username string, password string, sslMode string) string {
	values := map[string]string{
		"host":     host,
		"port":     strconv.Itoa(port),
		"dbname":   database,
		"user":     username,
		"password": password,
		"sslmode":  sslMode,
	}

	parts := make([]string, 0, len(values))

	for key, value := range values {
		parts = append(parts, fmt.Sprintf("%s=%s", key, quoteConnStringValue(value)))
	}

	return strings.Join(parts, " ")
}

func quoteConnStringValue(value string) string {
	escaped := strings.ReplaceAll(value, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `'`, `\'`)

	return "'" + escaped + "'"
}
