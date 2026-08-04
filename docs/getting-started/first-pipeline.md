# Run your first pipeline

Seed the demo database, create the platform resources, and watch change events land in
Kafka. This is the point of everything above it.

!!! warning "Placeholder"

    This page is a skeleton.

## Seed the source database

The repository ships an e-commerce schema with customers, orders and products, for both
PostgreSQL and MongoDB. Seeding is idempotent and can be re-run.

## Create the platform resources

A scenario creates, in order, the four objects the Debezium Platform needs:

| Object | What it is |
|---|---|
| Connection | Credentials and address of the source database |
| Source | What to capture from that connection |
| Destination | Where change events are written — here, Kafka |
| Pipeline | Binds a source to a destination, optionally through transforms |

Every object is created only if an object with the same deterministic name does not
already exist, so re-running a scenario converges rather than duplicating.

## Watch it work

Change a row in the source database and observe the corresponding event on the Kafka
topic. The platform UI shows the same pipeline and its status.

<!-- TODO: the concrete commands, the topic name, a sample change event payload, and a
     screenshot of the pipeline in the UI. -->

## Where next

- Deploy it somewhere real: [Guides](../guides/index.md)
- Understand what you just built: [Concepts](../concepts/index.md)
