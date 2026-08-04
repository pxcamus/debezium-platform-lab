# Concepts

Background for deciding whether this fits your problem, and for understanding what the
deployment actually did.

!!! warning "Placeholder"

    These pages are skeletons.

- **[Architecture](architecture.md)** — the components, and how a change in a database
  becomes an event on a topic.
- **[Environments](environments.md)** — why local, self-hosted and cloud deployments are one
  code path with different values rather than separate projects.

## Change data capture, briefly

Instead of polling a database for changes, change data capture reads its replication log —
the same stream the database uses for its own replicas. Every insert, update and delete
appears in order, with no load on the tables and no modification to the application writing
to them. Debezium turns that stream into events; the Debezium Platform is the control plane
that manages the connectors producing them.
