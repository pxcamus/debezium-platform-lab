//go:build mage

//mage:multiline

// Set the general description you want to have displayed with mage -l here.
package main

import (
	"context"
	"dbz-mage/ko/automation"
	"dbz-mage/ko/cluster"
	"dbz-mage/ko/dmp"
	mongodbsource "dbz-mage/ko/source/mongodb"
	postgresqlsource "dbz-mage/ko/source/postgresql"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	"github.com/magefile/mage/mg"
)

type Cluster mg.Namespace
type Helm mg.Namespace
type Data mg.Namespace
type Scenario mg.Namespace

// Recreate deletes and recreates the remote k3s cluster, then refreshes local kubeconfig.
func (Cluster) Recreate() error {
	if err := automation.LoadEnv(); err != nil {
		panic(err)
	}
	clusterInstance, err := cluster.NewFromEnv()
	if err != nil {
		return err
	}

	return clusterInstance.Recreate(context.Background())
}

// All applies all releases from deploy/helmfile.yaml.
func (Helm) All() error {
	if err := automation.LoadEnv(); err != nil {
		return err
	}

	if err := run("helmfile", "--file", "deploy/helmfile.yaml.gotmpl", "--selector", "app=strimzi-cluster-operator", "apply"); err != nil {
		return err
	}

	if err := run("helmfile", "--file", "deploy/helmfile.yaml.gotmpl", "--selector", "app=cnpg-operator", "apply"); err != nil {
		return err
	}

	if err := run("helmfile", "--file", "deploy/helmfile.yaml.gotmpl", "--selector", "app=mongodb-community-operator", "apply"); err != nil {
		return err
	}

	// --skip-diff-on-install: the infra batch includes kube-prometheus-stack, whose
	// own CRDs (Prometheus/Alertmanager/PrometheusRule) can't be resolved by the
	// helm-diff preview on a fresh cluster. Skipping diff for not-yet-installed
	// releases lets Helm install those CRDs before the resources that use them.
	if err := run("helmfile", "--file", "deploy/helmfile.yaml.gotmpl", "--selector", "infra=true", "apply", "--skip-diff-on-install"); err != nil {
		return err
	}

	if err := run("helmfile", "--file", "deploy/helmfile.yaml.gotmpl", "--selector", "app=debezium-operator", "apply"); err != nil {
		return err
	}

	return run("helmfile", "--file", "deploy/helmfile.yaml.gotmpl", "--selector", "app=debezium-platform", "apply")
}

func (Helm) Misc() error {
	if err := automation.LoadEnv(); err != nil {
		return err
	}

	return run("helmfile", "--file", "deploy/helmfile.yaml.gotmpl", "--selector", "app=cert-manager", "apply")
}

func (Helm) HttpServer() error {
	if err := automation.LoadEnv(); err != nil {
		return err
	}

	return run("helmfile", "--file", "deploy/helmfile.yaml.gotmpl", "--selector", "app=http-server", "apply")
}

// DbzOperator applies only the Debezium Operator release.
func (Helm) DbzOperator() error {
	if err := automation.LoadEnv(); err != nil {
		return err
	}

	return run("helmfile", "--file", "deploy/helmfile.yaml.gotmpl", "--selector", "app=debezium-operator", "apply")
}

// Platform applies only the Debezium Platform release.
func (Helm) Platform() error {
	if err := automation.LoadEnv(); err != nil {
		return err
	}

	return run("helmfile", "--file", "deploy/helmfile.yaml.gotmpl", "--selector", "app=debezium-platform", "apply")
}

// Infra applies the full infra=true batch (Kafka, databases, and their
// operators, plus homelab monitoring/registry/cert-manager) in one helmfile
// invocation, ordered by the releases' `needs` dependencies.
//
// --skip-diff-on-install: on a fresh cluster the helm-diff preview can't resolve
// CRDs that the operators install (Kafka/KafkaNodePool, CNPG Cluster,
// MongoDBCommunity, Prometheus/Alertmanager/OpenTelemetryCollector), which aborts
// the apply with "no matches for kind ... ensure CRDs are installed first".
// Skipping the diff for not-yet-installed releases lets Helm install the operators
// (and their CRDs) before the resources that use them, so this target is safe to
// run standalone.
func (Helm) Infra() error {
	if err := automation.LoadEnv(); err != nil {
		return err
	}

	return run("helmfile", "--file", "deploy/helmfile.yaml.gotmpl", "--selector", "infra=true", "apply", "--skip-diff-on-install")
}

// Monitoring applies the monitoring foundation (kube-prometheus-stack, then the
// OpenTelemetry Operator). Homelab only; both are also part of the infra=true batch.
//
// --skip-diff-on-install: on a fresh cluster the helm-diff preview can't resolve the
// chart's own CRDs (Prometheus/Alertmanager/PrometheusRule, OpenTelemetryCollector),
// which aborts the apply with "no matches for kind ... ensure CRDs are installed first".
// Skipping the diff for not-yet-installed releases lets Helm install the CRDs first.
func (Helm) Monitoring() error {
	if err := automation.LoadEnv(); err != nil {
		return err
	}

	if err := run("helmfile", "--file", "deploy/helmfile.yaml.gotmpl", "--selector", "app=kube-prometheus-stack", "apply", "--skip-diff-on-install"); err != nil {
		return err
	}

	return run("helmfile", "--file", "deploy/helmfile.yaml.gotmpl", "--selector", "app=opentelemetry-operator", "apply", "--skip-diff-on-install")
}

// PlatformDestroy uninstalls the Debezium Platform and Operator releases.
func (Helm) PlatformDestroy() error {
	if err := automation.LoadEnv(); err != nil {
		return err
	}

	if err := run("helmfile", "--file", "deploy/helmfile.yaml.gotmpl", "--selector", "app=debezium-platform", "destroy"); err != nil {
		return err
	}

	return run("helmfile", "--file", "deploy/helmfile.yaml.gotmpl", "--selector", "app=debezium-operator", "destroy")
}

// AllDestroy uninstalls all releases in reverse dependency order.
func (Helm) AllDestroy() error {
	if err := automation.LoadEnv(); err != nil {
		return err
	}

	if err := run("helmfile", "--file", "deploy/helmfile.yaml.gotmpl", "--selector", "app=debezium-platform", "destroy"); err != nil {
		return err
	}

	if err := run("helmfile", "--file", "deploy/helmfile.yaml.gotmpl", "--selector", "app=debezium-operator", "destroy"); err != nil {
		return err
	}

	if err := run("helmfile", "--file", "deploy/helmfile.yaml.gotmpl", "--selector", "infra=true", "destroy"); err != nil {
		return err
	}

	if err := run("helmfile", "--file", "deploy/helmfile.yaml.gotmpl", "--selector", "app=mongodb-community-operator", "destroy"); err != nil {
		return err
	}

	if err := run("helmfile", "--file", "deploy/helmfile.yaml.gotmpl", "--selector", "app=cnpg-operator", "destroy"); err != nil {
		return err
	}

	return run("helmfile", "--file", "deploy/helmfile.yaml.gotmpl", "--selector", "app=strimzi-cluster-operator", "destroy")
}

// Diff shows pending Helmfile changes from deploy/helmfile.yaml.
func (Helm) Diff() error {
	if err := automation.LoadEnv(); err != nil {
		return err
	}

	return run("helmfile", "--file", "deploy/helmfile.yaml.gotmpl", "diff")
}

// Mongo prepares MongoDB collections, indexes, and seed data.
func (Data) Mongo() error {
	if err := automation.LoadEnv(); err != nil {
		return err
	}
	configureLogger()

	slog.Info("Starting MongoDB data setup")

	ctx := context.Background()

	mongodb, err := mongodbsource.NewFromEnv(ctx)
	if err != nil {
		slog.Error("Failed to create MongoDB source", "error", err)
		return err
	}
	defer func() {
		if err := mongodb.Close(context.Background()); err != nil {
			slog.Warn("Failed to close MongoDB client", "error", err)
		}
	}()

	if err := mongodb.Wait(ctx); err != nil {
		return err
	}

	if err := mongodb.Setup(ctx); err != nil {
		return err
	}

	if err := mongodb.Populate(ctx); err != nil {
		return err
	}

	slog.Info("MongoDB data setup completed")

	return nil
}

// ResetMongo drops MongoDB ecommerce collections, recreates them, and reloads seed data.
func (Data) ResetMongo() error {
	if err := automation.LoadEnv(); err != nil {
		return err
	}
	configureLogger()

	slog.Warn("Starting MongoDB data reset")

	ctx := context.Background()

	mongodb, err := mongodbsource.NewFromEnv(ctx)
	if err != nil {
		slog.Error("Failed to create MongoDB source", "error", err)
		return err
	}
	defer func() {
		if err := mongodb.Close(context.Background()); err != nil {
			slog.Warn("Failed to close MongoDB client", "error", err)
		}
	}()

	if err := mongodb.Wait(ctx); err != nil {
		return err
	}

	if err := mongodb.Reset(ctx); err != nil {
		return err
	}

	slog.Info("MongoDB data reset completed")

	return nil
}

// Pg prepares PostgreSQL tables, indexes, Debezium publication/slot, and seed data.
func (Data) Pg() error {
	if err := automation.LoadEnv(); err != nil {
		return err
	}
	configureLogger()

	slog.Info("Starting PostgreSQL data setup")

	ctx := context.Background()

	postgresql, err := postgresqlsource.NewFromEnv(ctx)
	if err != nil {
		slog.Error("Failed to create PostgreSQL source", "error", err)
		return err
	}
	defer func() {
		if err := postgresql.Close(); err != nil {
			slog.Warn("Failed to close PostgreSQL connection", "error", err)
		}
	}()

	if err := postgresql.Wait(ctx); err != nil {
		return err
	}

	if err := postgresql.Setup(ctx); err != nil {
		return err
	}

	if err := postgresql.Populate(ctx); err != nil {
		return err
	}

	slog.Info("PostgreSQL data setup completed")

	return nil
}

// ResetPg drops PostgreSQL ecommerce tables/publication/slot, recreates them, and reloads seed data.
func (Data) ResetPg() error {
	if err := automation.LoadEnv(); err != nil {
		return err
	}
	configureLogger()

	slog.Warn("Starting PostgreSQL data reset")

	ctx := context.Background()

	postgresql, err := postgresqlsource.NewFromEnv(ctx)
	if err != nil {
		slog.Error("Failed to create PostgreSQL source", "error", err)
		return err
	}
	defer func() {
		if err := postgresql.Close(); err != nil {
			slog.Warn("Failed to close PostgreSQL connection", "error", err)
		}
	}()

	if err := postgresql.Wait(ctx); err != nil {
		return err
	}

	if err := postgresql.Reset(ctx); err != nil {
		return err
	}

	slog.Info("PostgreSQL data reset completed")

	return nil
}

//func (Data) All() error
//func (Data) ResetAll() error
//

// MongodbRs applies the MongoDB replica-set → Kafka scenario: it ensures the
// shared DMP connections, destinations, and transforms, then creates the
// scenario source and pipeline. Safe to re-run — existing resources are reused.
func (Scenario) MongodbRs() error {
	return mongoRsBasic()
}

// All applies every wired DMP scenario in dependency order.
func (Scenario) All() error {
	return mongoRsBasic()
}

// mongoRsBasic ensures the common DMP artifacts and the MongoDB replica-set
// scenario's source and pipeline, reusing any resources that already exist.
func mongoRsBasic() error {
	if err := automation.LoadEnv(); err != nil {
		return err
	}
	configureLogger()

	slog.Info("Creating DMP common artifacts")

	client := dmp.NewHTTPClient()
	resolver := dmp.NewResourceResolver(client)

	connections, err := resolver.EnsureCommonConnections("ko/scenarios", []string{
		"mongodb",
		"postgres",
		"http-server",
		"kafka",
		"sqlserver",
	})
	if err != nil {
		return err
	}

	slog.Info("DMP common connections are ready")

	destinations, err := resolver.EnsureCommonDestinations("ko/scenarios", []dmp.CommonDestinationSpec{
		{
			Key:           "kafka-string",
			ConnectionKey: "kafka",
		},
		{
			Key:           "http-server-base",
			ConnectionKey: "http-server",
		},
	}, connections)
	if err != nil {
		return err
	}

	_, err = resolver.EnsureCommonTransforms("ko/scenarios", []dmp.CommonTransformSpec{
		{Key: "mongo-extract-state"},
	})
	if err != nil {
		return err
	}

	sources, err := resolver.EnsureScenarioSources("ko/scenarios", "mongodb-rs-basic", []dmp.ScenarioSourceSpec{
		{
			Key:           "mongodb",
			File:          "payloads/source.json",
			ConnectionKey: "mongodb",
		},
	}, connections)
	if err != nil {
		return err
	}

	pipelines, err := resolver.EnsureScenarioPipelines("ko/scenarios", "mongodb-rs-basic", []dmp.ScenarioPipelineSpec{
		{
			Key:            "mongodb-to-kafka",
			File:           "payloads/pipeline.json",
			SourceKey:      "mongodb",
			DestinationKey: "kafka-string",
		},
	}, sources, destinations)
	if err != nil {
		return err
	}

	slog.Info(
		"DMP scenario artifacts are ready",
		"connections", len(connections),
		"destinations", len(destinations),
		"sources", len(sources),
		"pipelines", len(pipelines),
	)

	return nil
}

func configureLogger() {
	level := new(slog.LevelVar)

	switch strings.ToLower(automation.Env("LOG_LEVEL", "info")) {
	case "debug":
		level.Set(slog.LevelDebug)
	case "warn", "warning":
		level.Set(slog.LevelWarn)
	case "error":
		level.Set(slog.LevelError)
	default:
		level.Set(slog.LevelInfo)
	}

	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})

	slog.SetDefault(slog.New(handler))
}

func run(command string, args ...string) error {
	fmt.Printf("Running: %s %v\n", command, args)

	cmd := exec.Command(command, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Env = commandEnv()

	return cmd.Run()
}

// commandEnv builds the environment for helmfile/kubectl child processes. When
// CLUSTER_TYPE=k3s it pins KUBECONFIG to K3S_LOCAL_KUBECONFIG so every helm target
// always acts on the intended remote k3s cluster, rather than whatever context
// happens to be current in the default ~/.kube/config (which can silently retarget
// the wrong cluster). Any inherited KUBECONFIG is stripped so ours always wins.
func commandEnv() []string {
	env := os.Environ()

	if !strings.EqualFold(automation.Env("CLUSTER_TYPE", "kind"), "k3s") {
		return env
	}

	kubeconfig := automation.ExpandPath(automation.Env("K3S_LOCAL_KUBECONFIG", "~/.kube/k3s-aws.yaml"))

	result := make([]string, 0, len(env)+1)
	for _, kv := range env {
		if !strings.HasPrefix(kv, "KUBECONFIG=") {
			result = append(result, kv)
		}
	}

	return append(result, "KUBECONFIG="+kubeconfig)
}
