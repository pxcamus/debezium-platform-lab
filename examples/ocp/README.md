# OpenShift Local

The platform, a demo PostgreSQL source, an HTTP sink and the CDC dashboard on
[OpenShift Local](https://developers.redhat.com/products/openshift-local) (`crc`).

**Experimental.** Used on OpenShift Local 4.22 only. Single-node OpenShift and full
clusters are not covered.

## Before you start

The profile installs no operators. These come from OperatorHub and must be there first:

| What | From OperatorHub |
|---|---|
| CloudNativePG | `cloudnative-pg` (certified operators), all namespaces |
| OpenTelemetry | Red Hat build of OpenTelemetry (`opentelemetry-product`) |

User workload monitoring must be enabled (`enableUserWorkload: true` in the
`cluster-monitoring-config` ConfigMap).

## Deploy

```shell
cp examples/ocp/.env.example .env
just ocp-releases   # what will be installed, without touching the cluster
just ocp
```

The platform is then at <http://dmp.apps-crc.testing>.

## Known limitations

| Limitation | Workaround in this profile | Waiting for |
|---|---|---|
| The conductor's in-UI monitoring panels do not render: the conductor cannot query a Prometheus that requires a bearer token and TLS | None. Metrics are in the console under Observe > Metrics | [dbz#3049](https://github.com/debezium/dbz/issues/3049) |
| The OpenTelemetry collector fails with the Red Hat collector image, which has no `nop` exporter | The upstream collector image is pinned | [dbz#3042](https://github.com/debezium/dbz/issues/3042) |
| The nightly Debezium Server image fails to start when the JMX exporter is enabled | Pipelines run `quay.io/debezium/server:3.7.1.Final` | [dbz#3043](https://github.com/debezium/dbz/issues/3043) |

Single sign-on (`components.sso`) is not available on this profile.
